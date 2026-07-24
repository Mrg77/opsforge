package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSeverityString(t *testing.T) {
	cases := map[Severity]string{
		SevCritical:  "CRITICAL",
		SevWarning:   "WARNING",
		SevInfo:      "INFO",
		Severity(99): "INFO", // unknown falls back to INFO
	}
	for sev, want := range cases {
		if got := sev.String(); got != want {
			t.Errorf("Severity(%d).String() = %q, want %q", sev, got, want)
		}
	}
}

func TestScanFileFindsLeaksWithLineNumbers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "HARMLESS=1\n" +
		"export GITHUB_TOKEN=ghp_" + "abcdefghijklmnopqrstuvwxyz0123456789AB\n" +
		"NOTHING=here\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	findings := ScanFile(path)
	if len(findings) == 0 {
		t.Fatal("expected at least one finding in the .env file")
	}
	f := findings[0]
	if f.Source != path {
		t.Errorf("finding source = %q, want %q", f.Source, path)
	}
	if f.Line != 2 {
		t.Errorf("finding line = %d, want 2 (the export line)", f.Line)
	}
	// The excerpt must be masked — never the raw token.
	if len(f.Excerpt) == 0 || containsRawToken(f.Excerpt) {
		t.Errorf("excerpt should be masked, got %q", f.Excerpt)
	}
}

func containsRawToken(s string) bool {
	// A masked excerpt ends in "…(N chars)"; a raw 44-char token wouldn't.
	return len(s) > 8 && s[len(s)-1] != ')'
}

func TestScanFileMissingIsSafe(t *testing.T) {
	if f := ScanFile(filepath.Join(t.TempDir(), "does-not-exist")); f != nil {
		t.Errorf("scanning a missing file should return nil, got %v", f)
	}
}

func TestDefaultTargetsIncludesHistoryAndEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// A .env in the working directory should be picked up.
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, ".env.local"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	targets := DefaultTargets()

	var hasZshHist, hasEnv bool
	for _, tg := range targets {
		if filepath.Base(tg) == ".zsh_history" {
			hasZshHist = true
		}
		if filepath.Base(tg) == ".env.local" {
			hasEnv = true
		}
	}
	if !hasZshHist {
		t.Error("DefaultTargets should include ~/.zsh_history")
	}
	if !hasEnv {
		t.Error("DefaultTargets should include .env* files in the cwd")
	}
}
