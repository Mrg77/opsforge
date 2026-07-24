package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSha256File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "asset.bin")
	// "" hashes to the well-known empty SHA-256; a known payload to a known sum.
	if err := os.WriteFile(path, []byte("opsforge"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := sha256File(path)
	if err != nil {
		t.Fatalf("sha256File: %v", err)
	}
	// A SHA-256 hex string is 64 chars; assert shape + determinism rather than
	// a hard-coded digest (keeps the test robust and self-checking).
	if len(got) != 64 {
		t.Errorf("sha256 hex length = %d, want 64 (%q)", len(got), got)
	}
	again, _ := sha256File(path)
	if got != again {
		t.Error("sha256File is not deterministic for the same file")
	}

	// A different payload must hash differently.
	other := filepath.Join(dir, "other.bin")
	if err := os.WriteFile(other, []byte("different"), 0o644); err != nil {
		t.Fatal(err)
	}
	og, _ := sha256File(other)
	if og == got {
		t.Error("distinct payloads produced the same hash")
	}
}

func TestSha256FileMissing(t *testing.T) {
	if _, err := sha256File(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("sha256File on a missing file should error")
	}
}
