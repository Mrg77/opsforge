// Package envvault is opsforge's encrypted environment store: a vault of
// KEY=value pairs that persists across shells WITHOUT writing a secret in
// cleartext to disk, and WITHOUT retyping a passphrase on every operation.
//
// A DevOps re-exports the same AWS/Vault/registry credentials in every new
// terminal. Putting them in ~/.zshrc persists them — but in cleartext, where a
// dotfiles backup or `opsforge audit --secrets` finds them leaking. This vault
// keeps the convenience (set once, load anywhere) while the file at rest is
// encrypted.
//
// # Wrapped-key design (why there's an "agent")
//
// The vault is encrypted to a random X25519 *identity* (age's public-key mode),
// NOT directly to the passphrase. That identity is itself encrypted with the
// passphrase (age scrypt) and stored beside the vault (identity.age). So:
//
//   - The passphrase unlocks the identity — nothing else.
//   - `unlock` decrypts the identity once and caches it in a RAM session file
//     ($TMPDIR, 0600, with a TTL). Subsequent set/list/load read the cached
//     identity — no passphrase prompt — until the TTL lapses or `lock` runs.
//
// This is the ssh-agent model: type the passphrase once, work for a while.
// What lives in RAM is the vault KEY, never the passphrase, and it self-expires.
//
// Encryption is delegated to age (filippo.io/age); opsforge rolls no crypto.
//
// Honesty about the threat model: once loaded into a shell, the values are
// ordinary environment variables in that session — visible to child processes.
// And while a session is unlocked, anyone able to read your $TMPDIR as you
// could use the cached key. Both are inherent to the convenience; the wins are
// nothing in cleartext at rest, no retyping, and a key that self-expires.
package envvault

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"filippo.io/age"
)

// DefaultTTL is how long an unlocked session stays valid.
const DefaultTTL = 15 * time.Minute

// dir is ~/.config/opsforge; the vault and wrapped identity live here.
func dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "opsforge"), nil
}

// VaultPath is ~/.config/opsforge/env.age — the encrypted KEY=value store.
func VaultPath() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "env.age"), nil
}

// identityPath is ~/.config/opsforge/identity.age — the vault's X25519 identity,
// itself encrypted with the master passphrase.
func identityPath() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "identity.age"), nil
}

// Exists reports whether a vault has been created (its identity is present).
func Exists() bool {
	p, err := identityPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// --- the in-memory Vault (unchanged shape) ---------------------------------

// Vault is the decrypted set of variables. It exists only in memory; it is
// never written unencrypted.
type Vault struct {
	vars map[string]string
}

func New() *Vault { return &Vault{vars: map[string]string{}} }

// Set adds or replaces a variable, validating the name and rejecting newlines
// (the on-disk body is one KEY=value per line).
func (v *Vault) Set(key, value string) error {
	if !ValidName(key) {
		return fmt.Errorf("invalid variable name %q (letters, digits, underscore; not starting with a digit)", key)
	}
	if strings.ContainsAny(value, "\n\r") {
		return fmt.Errorf("value for %q contains a newline, which the vault can't store", key)
	}
	v.vars[key] = value
	return nil
}

func (v *Vault) Remove(key string) (ok bool) {
	if _, ok = v.vars[key]; ok {
		delete(v.vars, key)
	}
	return ok
}

func (v *Vault) Names() []string {
	names := make([]string, 0, len(v.vars))
	for k := range v.vars {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func (v *Vault) Len() int { return len(v.vars) }

func (v *Vault) Get(key string) (string, bool) {
	val, ok := v.vars[key]
	return val, ok
}

// ExportLines renders the vault as shell `export KEY='value'` lines, ready to
// eval. Single-quoting with the '\” escape makes any value injection-proof.
func (v *Vault) ExportLines() string {
	var b strings.Builder
	for _, k := range v.Names() {
		fmt.Fprintf(&b, "export %s=%s\n", k, shellQuote(v.vars[k]))
	}
	return b.String()
}

func ValidName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_':
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func (v *Vault) body() string {
	var b strings.Builder
	for _, k := range v.Names() {
		fmt.Fprintf(&b, "%s=%s\n", k, v.vars[k])
	}
	return b.String()
}

func parse(data []byte) (*Vault, error) {
	v := New()
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return nil, fmt.Errorf("corrupt vault: line without '='")
		}
		key := line[:eq]
		if !ValidName(key) {
			return nil, fmt.Errorf("corrupt vault: invalid key %q", key)
		}
		v.vars[key] = line[eq+1:]
	}
	return v, nil
}

// --- identity: the vault's X25519 key, wrapped by the passphrase -----------

// Create initializes a brand-new vault: generates a random X25519 identity,
// wraps it with the passphrase, and writes an empty encrypted vault. Fails if a
// vault already exists (callers should check Exists first for a clean message).
func Create(passphrase string) (*age.X25519Identity, error) {
	if Exists() {
		return nil, fmt.Errorf("a vault already exists")
	}
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, err
	}
	if err := writeWrappedIdentity(id, passphrase); err != nil {
		return nil, err
	}
	if err := SaveWith(New(), id); err != nil {
		return nil, err
	}
	return id, nil
}

// writeWrappedIdentity encrypts the identity's secret string with the
// passphrase (age scrypt) and writes identity.age atomically, 0600.
func writeWrappedIdentity(id *age.X25519Identity, passphrase string) error {
	p, err := identityPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	rcpt, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, rcpt)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w, id.String()); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return atomicWrite(p, buf.Bytes())
}

// UnwrapIdentity decrypts the vault's X25519 identity using the passphrase.
// This is the ONLY operation that needs the passphrase.
func UnwrapIdentity(passphrase string) (*age.X25519Identity, error) {
	p, err := identityPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	scryptID, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return nil, err
	}
	r, err := age.Decrypt(bytes.NewReader(data), scryptID)
	if err != nil {
		return nil, fmt.Errorf("could not unlock the vault — wrong passphrase?")
	}
	secret, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return age.ParseX25519Identity(strings.TrimSpace(string(secret)))
}

// --- load / save the vault with an identity (no passphrase needed) ---------

// LoadWith decrypts the vault using an already-unwrapped identity.
func LoadWith(id *age.X25519Identity) (*Vault, error) {
	p, err := VaultPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return New(), nil
	}
	if err != nil {
		return nil, err
	}
	r, err := age.Decrypt(bytes.NewReader(data), id)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt the vault (key mismatch?)")
	}
	plain, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return parse(plain)
}

// SaveWith encrypts the vault to disk for the identity's recipient (public
// key), atomically and 0600.
func SaveWith(v *Vault, id *age.X25519Identity) error {
	p, err := VaultPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, id.Recipient())
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(v.body())); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return atomicWrite(p, buf.Bytes())
}

func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// --- session agent: cache the unwrapped identity in RAM with a TTL ---------

// sessionPath is the RAM-backed session file. On macOS $TMPDIR is per-user and
// cleared on reboot; on Linux we prefer $XDG_RUNTIME_DIR (tmpfs) when present.
func sessionPath() (string, error) {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		base = os.TempDir()
	}
	uid := os.Getuid()
	return filepath.Join(base, fmt.Sprintf("opsforge-vault-%d.session", uid)), nil
}

// OpenSession decrypts the identity with the passphrase and caches it in the
// RAM session file with an expiry of now+ttl. This is `unlock`.
func OpenSession(passphrase string, ttl time.Duration) error {
	id, err := UnwrapIdentity(passphrase)
	if err != nil {
		return err
	}
	p, err := sessionPath()
	if err != nil {
		return err
	}
	// Line 1: unix expiry. Line 2: the identity secret string.
	expiry := time.Now().Add(ttl).Unix()
	body := strconv.FormatInt(expiry, 10) + "\n" + id.String() + "\n"
	return os.WriteFile(p, []byte(body), 0o600)
}

// Session returns the cached identity if a session is unlocked and unexpired;
// ok is false otherwise (caller then prompts for the passphrase).
func Session() (id *age.X25519Identity, ok bool) {
	p, err := sessionPath()
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	lines := strings.SplitN(strings.TrimRight(string(data), "\n"), "\n", 2)
	if len(lines) != 2 {
		return nil, false
	}
	expiry, err := strconv.ParseInt(lines[0], 10, 64)
	if err != nil || time.Now().Unix() >= expiry {
		_ = os.Remove(p) // expired — clear it
		return nil, false
	}
	id, err = age.ParseX25519Identity(lines[1])
	if err != nil {
		return nil, false
	}
	return id, true
}

// CloseSession removes the RAM session file (this is `lock`). Returns whether a
// session was actually present.
func CloseSession() bool {
	p, err := sessionPath()
	if err != nil {
		return false
	}
	if _, err := os.Stat(p); err != nil {
		return false
	}
	_ = os.Remove(p)
	return true
}

// SessionRemaining reports the time left on the current session (0 if none).
func SessionRemaining() time.Duration {
	p, err := sessionPath()
	if err != nil {
		return 0
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	lines := strings.SplitN(string(data), "\n", 2)
	expiry, err := strconv.ParseInt(strings.TrimSpace(lines[0]), 10, 64)
	if err != nil {
		return 0
	}
	d := time.Until(time.Unix(expiry, 0))
	if d < 0 {
		return 0
	}
	return d
}
