package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mrg77/opsforge/internal/catalog"
)

func TestResolveAssetFor(t *testing.T) {
	cases := []struct {
		name       string
		gh         catalog.GitHubRelease
		goos, arch string
		tag        string
		want       string
	}{
		{
			name: "os and arch maps applied",
			gh: catalog.GitHubRelease{
				AssetTemplate: "k9s_{os}_{arch}.tar.gz",
				OSMap:         map[string]string{"darwin": "Darwin", "linux": "Linux"},
			},
			goos: "darwin", arch: "arm64", tag: "v0.32.5",
			want: "k9s_Darwin_arm64.tar.gz",
		},
		{
			name: "version placeholder stripped of v",
			gh: catalog.GitHubRelease{
				AssetTemplate: "tool-{version}-{os}-{arch}",
			},
			goos: "linux", arch: "amd64", tag: "v1.2.3",
			want: "tool-1.2.3-linux-amd64",
		},
		{
			name: "arch remap x86_64",
			gh: catalog.GitHubRelease{
				AssetTemplate: "bin_{os}_{arch}",
				ArchMap:       map[string]string{"amd64": "x86_64"},
			},
			goos: "linux", arch: "amd64", tag: "v2",
			want: "bin_linux_x86_64",
		},
		{
			name: "raw binary name with tag",
			gh: catalog.GitHubRelease{
				AssetTemplate: "kind-{os}-{arch}",
			},
			goos: "linux", arch: "arm64", tag: "v0.23.0",
			want: "kind-linux-arm64",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := resolveAssetFor(&c.gh, c.tag, c.goos, c.arch)
			if got != c.want {
				t.Errorf("resolveAssetFor() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestExtractTarGzFindsBinaryByBasename(t *testing.T) {
	// Build a tar.gz holding ./nested/mytool plus a decoy file.
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	writeEntry := func(name, body string) {
		tw.WriteHeader(&tar.Header{
			Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg,
		})
		tw.Write([]byte(body))
	}
	writeEntry("README.md", "docs")
	writeEntry("nested/mytool", "#!/bin/sh\necho hi\n")
	tw.Close()
	gz.Close()

	dir := t.TempDir()
	archive := filepath.Join(dir, "mytool.tar.gz")
	if err := os.WriteFile(archive, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	// Ask for "mytool" — extraction should match by basename inside nested/.
	got, err := extractBinary(archive, dir, "mytool", "")
	if err != nil {
		t.Fatalf("extractBinary failed: %v", err)
	}
	data, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("echo hi")) {
		t.Errorf("extracted wrong file, content = %q", data)
	}
}

func TestExtractBinaryMissingBinary(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "other", Mode: 0o755, Size: 1, Typeflag: tar.TypeReg})
	tw.Write([]byte("x"))
	tw.Close()
	gz.Close()

	dir := t.TempDir()
	archive := filepath.Join(dir, "a.tar.gz")
	os.WriteFile(archive, buf.Bytes(), 0o644)

	if _, err := extractBinary(archive, dir, "wanted", ""); err == nil {
		t.Error("expected error when binary is absent from archive")
	}
}
