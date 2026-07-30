package envvault

import (
	"os"
	"strings"
	"testing"
	"time"
)

// withTempEnv isolates $HOME (vault location) and a session dir so tests never
// touch the real vault or a real RAM session.
func withTempEnv(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	run := t.TempDir()
	for k, v := range map[string]string{"HOME": home, "XDG_RUNTIME_DIR": run} {
		old := os.Getenv(k)
		os.Setenv(k, v)
		t.Cleanup(func() { os.Setenv(k, old) })
	}
}

func TestCreateAndRoundTrip(t *testing.T) {
	withTempEnv(t)
	pass := "correct horse battery staple"

	id, err := Create(pass)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !Exists() {
		t.Fatal("Exists() should be true after Create")
	}

	v, err := LoadWith(id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	must(t, v.Set("AWS_SECRET_ACCESS_KEY", "s3cr3t/with+slash and space"))
	must(t, SaveWith(v, id))

	// Re-unwrap from passphrase and read back.
	id2, err := UnwrapIdentity(pass)
	if err != nil {
		t.Fatalf("unwrap: %v", err)
	}
	got, err := LoadWith(id2)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if val, _ := got.Get("AWS_SECRET_ACCESS_KEY"); val != "s3cr3t/with+slash and space" {
		t.Errorf("round-trip mismatch: %q", val)
	}
}

func TestWrongPassphraseFails(t *testing.T) {
	withTempEnv(t)
	if _, err := Create("right-pass"); err != nil {
		t.Fatal(err)
	}
	if _, err := UnwrapIdentity("wrong-pass"); err == nil {
		t.Fatal("expected error with wrong passphrase")
	}
}

func TestFilesEncryptedAtRest(t *testing.T) {
	withTempEnv(t)
	id, err := Create("pw")
	if err != nil {
		t.Fatal(err)
	}
	v, _ := LoadWith(id)
	secret := "AKIAVERYSECRETVALUE12345"
	must(t, v.Set("AWS_SECRET_ACCESS_KEY", secret))
	must(t, SaveWith(v, id))

	vp, _ := VaultPath()
	raw, _ := os.ReadFile(vp)
	if strings.Contains(string(raw), secret) {
		t.Fatal("secret in cleartext in vault file")
	}
	if !strings.HasPrefix(string(raw), "age-encryption.org/v1") {
		t.Error("vault is not an age file")
	}
	// The identity file must also be an encrypted age file (passphrase-wrapped).
	ip, _ := identityPath()
	rawID, _ := os.ReadFile(ip)
	if !strings.HasPrefix(string(rawID), "age-encryption.org/v1") {
		t.Error("identity file is not an age file")
	}
	if strings.Contains(string(rawID), "AGE-SECRET-KEY-") {
		t.Fatal("identity secret key is in cleartext")
	}
}

func TestSessionUnlockAvoidsPassphrase(t *testing.T) {
	withTempEnv(t)
	pass := "pw"
	if _, err := Create(pass); err != nil {
		t.Fatal(err)
	}

	// No session yet.
	if _, ok := Session(); ok {
		t.Fatal("no session should exist before unlock")
	}

	// Unlock: caches the identity in RAM.
	if err := OpenSession(pass, DefaultTTL); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	id, ok := Session()
	if !ok {
		t.Fatal("session should be active after unlock")
	}

	// The cached identity actually works to save/load without the passphrase.
	v, _ := LoadWith(id)
	must(t, v.Set("KUBECONFIG", "/tmp/kc"))
	must(t, SaveWith(v, id))
	got, _ := LoadWith(id)
	if val, _ := got.Get("KUBECONFIG"); val != "/tmp/kc" {
		t.Errorf("session identity round-trip failed: %q", val)
	}

	// Lock clears it.
	if !CloseSession() {
		t.Error("CloseSession should report a session was present")
	}
	if _, ok := Session(); ok {
		t.Fatal("session should be gone after lock")
	}
}

func TestSessionExpires(t *testing.T) {
	withTempEnv(t)
	pass := "pw"
	if _, err := Create(pass); err != nil {
		t.Fatal(err)
	}
	// Open with a negative TTL → already expired.
	if err := OpenSession(pass, -time.Second); err != nil {
		t.Fatal(err)
	}
	if _, ok := Session(); ok {
		t.Fatal("an expired session must not be returned")
	}
}

func TestSessionFileIs0600(t *testing.T) {
	withTempEnv(t)
	if _, err := Create("pw"); err != nil {
		t.Fatal(err)
	}
	if err := OpenSession("pw", DefaultTTL); err != nil {
		t.Fatal(err)
	}
	p, _ := sessionPath()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("session file perms = %o, want 600", perm)
	}
}

func TestExportLinesQuotingIsInjectionProof(t *testing.T) {
	v := New()
	must(t, v.Set("EVIL", `'; rm -rf /; echo '`))
	line := strings.TrimSpace(v.ExportLines())
	want := `export EVIL='` + `'\''` + `; rm -rf /; echo ` + `'\''` + `'`
	if line != want {
		t.Errorf("quoting not injection-proof\n got: %s\nwant: %s", line, want)
	}
}

func TestSetValidatesNames(t *testing.T) {
	v := New()
	for _, bad := range []string{"", "1ABC", "has space", "has-dash", "a=b"} {
		if err := v.Set(bad, "x"); err == nil {
			t.Errorf("Set(%q) should have failed", bad)
		}
	}
	for _, good := range []string{"A", "_x", "AWS_REGION", "TF_VAR_foo", "x9"} {
		if err := v.Set(good, "x"); err != nil {
			t.Errorf("Set(%q) should have succeeded: %v", good, err)
		}
	}
}

func TestSetRejectsNewline(t *testing.T) {
	v := New()
	if err := v.Set("MULTI", "line1\nline2"); err == nil {
		t.Error("Set with newline value should fail")
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
