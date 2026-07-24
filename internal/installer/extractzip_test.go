package installer

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mrg77/opsforge/internal/catalog"
)

func TestExtractZipFindsBinaryByBasename(t *testing.T) {
	// Build a .zip holding nested/mytool plus a decoy.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	write := func(name, body string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body))
	}
	write("README.md", "docs")
	write("nested/mytool", "#!/bin/sh\necho zip-hi\n")
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	archive := filepath.Join(dir, "mytool.zip")
	if err := os.WriteFile(archive, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := extractZip(archive, dir, "mytool")
	if err != nil {
		t.Fatalf("extractZip failed: %v", err)
	}
	data, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("echo zip-hi")) {
		t.Errorf("extracted wrong file: %q", data)
	}
}

func TestExtractZipMissingBinary(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("only-docs.txt")
	w.Write([]byte("nothing here"))
	zw.Close()

	dir := t.TempDir()
	archive := filepath.Join(dir, "a.zip")
	os.WriteFile(archive, buf.Bytes(), 0o644)

	if _, err := extractZip(archive, dir, "mytool"); err == nil {
		t.Error("extractZip should error when the wanted binary is absent")
	}
}

func TestBackendForRespectsForcedGitHub(t *testing.T) {
	t.Setenv("OPSFORGE_BACKEND", "github")

	withGH := catalog.Tool{Name: "x", GitHub: &catalog.GitHubRelease{Repo: "acme/x"}}
	if got := backendFor(withGH); got != BackendGitHub {
		t.Errorf("forced github + a github block should pick github, got %q", got)
	}

	// Forced github but no github block → none (can't satisfy the request).
	noGH := catalog.Tool{Name: "y", Brew: "y"}
	if got := backendFor(noGH); got != BackendNone {
		t.Errorf("forced github without a github block should be none, got %q", got)
	}
}

func TestBackendForAutoPrefersGitHubWhenNoBrew(t *testing.T) {
	// Force nothing; simulate "no brew" by asking a tool that only has a
	// github block — auto mode must fall back to github regardless of brew.
	t.Setenv("OPSFORGE_BACKEND", "")
	ghOnly := catalog.Tool{Name: "z", GitHub: &catalog.GitHubRelease{Repo: "acme/z"}}
	if got := backendFor(ghOnly); got != BackendGitHub {
		t.Errorf("a github-only tool should resolve to github in auto mode, got %q", got)
	}
}
