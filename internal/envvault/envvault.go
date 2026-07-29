// Package envvault is opsforge's encrypted environment store: a passphrase-
// locked vault of KEY=value pairs that persists across shells WITHOUT ever
// writing a secret in cleartext to disk.
//
// A DevOps re-exports the same AWS/Vault/registry credentials in every new
// terminal. Putting them in ~/.zshrc persists them — but in cleartext, where a
// dotfiles backup or `opsforge audit --secrets` finds them leaking. This vault
// keeps the convenience (set once, load anywhere) while the file at rest is
// encrypted: `cat`-ing it reveals nothing, and the secret scanner ignores it.
//
// Encryption is delegated to age (filippo.io/age) with a scrypt passphrase
// recipient — the reference implementation, not home-rolled crypto. The
// plaintext is a tiny dotenv-style body (one KEY=value per line); age handles
// the KDF, authentication and format.
//
// Honesty about the threat model: once `load` decrypts, the values become
// ordinary environment variables in that session — visible to child processes
// and in the process environment. That's inherent to using them (the AWS CLI
// must read the key). What the vault buys is: nothing in cleartext on disk,
// no retyping, and dotfiles you can back up or commit without leaking.
package envvault

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"filippo.io/age"
)

// Path is ~/.config/opsforge/env.age — the encrypted vault file.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "opsforge", "env.age"), nil
}

// Exists reports whether a vault file is present.
func Exists() bool {
	p, err := Path()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// Vault is the decrypted set of variables, kept ordered by key for stable
// output. It exists only in memory; it is never written unencrypted.
type Vault struct {
	// vars holds KEY -> value. Unexported so callers go through the methods.
	vars map[string]string
}

// New returns an empty vault.
func New() *Vault { return &Vault{vars: map[string]string{}} }

// Set adds or replaces a variable. The key is validated to be a shell-safe
// environment variable name so a malformed entry can't corrupt the dotenv body
// or inject extra lines.
func (v *Vault) Set(key, value string) error {
	if !ValidName(key) {
		return fmt.Errorf("invalid variable name %q (use letters, digits, underscore; not starting with a digit)", key)
	}
	// The on-disk body is one KEY=value per line, so a newline in a value would
	// split into a bogus extra entry on reload. Env-var values are single-line
	// in practice; refuse the rare exception rather than silently corrupt.
	if strings.ContainsAny(value, "\n\r") {
		return fmt.Errorf("value for %q contains a newline, which the vault can't store", key)
	}
	v.vars[key] = value
	return nil
}

// Remove deletes a variable; ok is false if it wasn't present.
func (v *Vault) Remove(key string) (ok bool) {
	if _, ok = v.vars[key]; ok {
		delete(v.vars, key)
	}
	return ok
}

// Names returns the variable names, sorted. Values are never exposed in bulk.
func (v *Vault) Names() []string {
	names := make([]string, 0, len(v.vars))
	for k := range v.vars {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// Len is the number of stored variables.
func (v *Vault) Len() int { return len(v.vars) }

// Get returns a single value.
func (v *Vault) Get(key string) (string, bool) {
	val, ok := v.vars[key]
	return val, ok
}

// ExportLines renders the vault as shell `export KEY='value'` lines, ready to
// eval. Single-quoting with the standard '\” escape makes any value safe
// (spaces, $, backticks, newlines all stay literal).
func (v *Vault) ExportLines() string {
	var b strings.Builder
	for _, k := range v.Names() {
		fmt.Fprintf(&b, "export %s=%s\n", k, shellQuote(v.vars[k]))
	}
	return b.String()
}

// ValidName reports whether s is a POSIX-ish environment variable name.
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
				return false // can't start with a digit
			}
		default:
			return false
		}
	}
	return true
}

// shellQuote wraps a value in single quotes, escaping embedded single quotes
// as '\” — the portable, injection-proof idiom for sh/zsh/bash.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// --- persistence: encrypt to / decrypt from the age vault file -------------

// Load decrypts the vault file with the passphrase. A missing file yields an
// empty vault (so `set` on a fresh machine just works). A wrong passphrase
// returns an error whose message stays generic (no oracle).
func Load(passphrase string) (*Vault, error) {
	p, err := Path()
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

	id, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return nil, err
	}
	r, err := age.Decrypt(bytes.NewReader(data), id)
	if err != nil {
		// age returns a distinct "no identity matched" on a bad passphrase;
		// present it plainly without leaking which check failed.
		return nil, fmt.Errorf("could not unlock the vault — wrong passphrase?")
	}
	plain, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return parse(plain)
}

// Save encrypts the vault to disk with the passphrase, atomically (write to a
// temp file then rename) and 0600, so a crash can't leave a half-written or
// world-readable vault.
func Save(v *Vault, passphrase string) error {
	p, err := Path()
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
	if _, err := w.Write([]byte(v.body())); err != nil {
		return err
	}
	if err := w.Close(); err != nil { // flushes age's footer/MAC
		return err
	}

	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// body renders the plaintext dotenv form (KEY=value per line, sorted). This
// string is only ever handed to age.Encrypt — never written to disk as-is.
func (v *Vault) body() string {
	var b strings.Builder
	for _, k := range v.Names() {
		// Safe: Set() forbids newlines in values, so one line == one variable.
		fmt.Fprintf(&b, "%s=%s\n", k, v.vars[k])
	}
	return b.String()
}

// parse reads the dotenv body back into a vault.
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
