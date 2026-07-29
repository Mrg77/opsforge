package envvault

import (
	"os"
	"strings"
	"testing"
)

// withTempHome points $HOME at a scratch dir so tests never touch the real
// vault, and restores it after.
func withTempHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	old := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	t.Cleanup(func() { os.Setenv("HOME", old) })
}

func TestRoundTrip(t *testing.T) {
	withTempHome(t)
	pass := "correct horse battery staple"

	v := New()
	must(t, v.Set("AWS_ACCESS_KEY_ID", "AKIAEXAMPLE"))
	must(t, v.Set("AWS_SECRET_ACCESS_KEY", "s3cr3t/with+slash and space"))
	if err := Save(v, pass); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := Load(pass)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Len() != 2 {
		t.Fatalf("want 2 vars, got %d", got.Len())
	}
	if val, _ := got.Get("AWS_SECRET_ACCESS_KEY"); val != "s3cr3t/with+slash and space" {
		t.Errorf("value round-trip mismatch: %q", val)
	}
}

func TestWrongPassphraseFails(t *testing.T) {
	withTempHome(t)
	v := New()
	must(t, v.Set("TOKEN", "abc"))
	must(t, Save(v, "right-pass"))

	if _, err := Load("wrong-pass"); err == nil {
		t.Fatal("expected an error with the wrong passphrase, got nil")
	}
}

func TestFileIsEncryptedAtRest(t *testing.T) {
	withTempHome(t)
	v := New()
	secret := "AKIAVERYSECRETVALUE12345"
	must(t, v.Set("AWS_SECRET_ACCESS_KEY", secret))
	must(t, Save(v, "pw"))

	p, _ := Path()
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), secret) {
		t.Fatal("secret value appears in cleartext in the vault file — encryption failed")
	}
	// age files start with this header; a good sanity check we actually encrypted.
	if !strings.HasPrefix(string(raw), "age-encryption.org/v1") {
		t.Errorf("vault file is not an age file; header=%q", firstLine(string(raw)))
	}
}

func TestMissingVaultLoadsEmpty(t *testing.T) {
	withTempHome(t)
	v, err := Load("anything")
	if err != nil {
		t.Fatalf("load of missing vault should succeed empty, got %v", err)
	}
	if v.Len() != 0 {
		t.Errorf("want empty vault, got %d vars", v.Len())
	}
}

func TestExportLinesQuotingIsInjectionProof(t *testing.T) {
	v := New()
	// A value engineered to break out of quoting or inject a command. Each
	// embedded ' becomes the '\'' sequence; the whole value stays wrapped in a
	// single-quoted string, so a shell eval sees it as one literal argument.
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

func TestRemove(t *testing.T) {
	v := New()
	must(t, v.Set("A", "1"))
	if !v.Remove("A") {
		t.Error("Remove should report ok for present key")
	}
	if v.Remove("A") {
		t.Error("Remove should report false for absent key")
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
