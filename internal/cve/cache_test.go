package cve

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	want := Summary{
		ScannedAt:      time.Now().UTC().Truncate(time.Second),
		Vulnerable:     2,
		HighOrCritical: 1,
		Tools: []Affected{
			{Name: "argocd", TopSeverity: "CRITICAL"},
			{Name: "helm", TopSeverity: "MEDIUM"},
		},
	}
	if err := Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, ok := Load()
	if !ok {
		t.Fatal("Load ok=false after Save")
	}
	if got.Vulnerable != want.Vulnerable || got.HighOrCritical != want.HighOrCritical {
		t.Errorf("counts mismatch: got %+v want %+v", got, want)
	}
	if len(got.Tools) != 2 || got.Tools[0].Name != "argocd" || got.Tools[0].TopSeverity != "CRITICAL" {
		t.Errorf("tools mismatch: %+v", got.Tools)
	}
	if !got.ScannedAt.Equal(want.ScannedAt) {
		t.Errorf("ScannedAt mismatch: got %v want %v", got.ScannedAt, want.ScannedAt)
	}
}

func TestLoadMissingCacheIsNotOK(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if _, ok := Load(); ok {
		t.Error("Load should report ok=false when no cache exists")
	}
}

func TestLoadCorruptCacheIsNotOK(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	p := filepath.Join(home, ".cache", "opsforge", "cve-cache.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := Load(); ok {
		t.Error("Load should report ok=false on corrupt JSON (treated as unknown, not clean)")
	}
}
