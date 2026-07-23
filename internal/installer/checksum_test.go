package installer

import (
	"testing"

	"github.com/Mrg77/opsforge/internal/catalog"
)

func TestChecksumFor(t *testing.T) {
	const h1 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	const h2 = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

	t.Run("checksums.txt list format", func(t *testing.T) {
		body := h1 + "  k9s_Linux_amd64.tar.gz\n" +
			h2 + "  k9s_Darwin_arm64.tar.gz\n"
		got, ok := checksumFor(body, "k9s_Darwin_arm64.tar.gz")
		if !ok || got != h2 {
			t.Fatalf("got (%q,%v), want (%q,true)", got, ok, h2)
		}
	})

	t.Run("binary-mode asterisk prefix", func(t *testing.T) {
		body := h1 + " *tool_linux_amd64.tar.gz\n"
		got, ok := checksumFor(body, "tool_linux_amd64.tar.gz")
		if !ok || got != h1 {
			t.Fatalf("got (%q,%v), want (%q,true)", got, ok, h1)
		}
	})

	t.Run("bare single hash file", func(t *testing.T) {
		got, ok := checksumFor(h1+"\n", "anything.tar.gz")
		if !ok || got != h1 {
			t.Fatalf("got (%q,%v), want (%q,true)", got, ok, h1)
		}
	})

	t.Run("path-qualified filename matches on basename", func(t *testing.T) {
		body := h1 + "  ./dist/tool_linux_amd64.tar.gz\n"
		got, ok := checksumFor(body, "tool_linux_amd64.tar.gz")
		if !ok || got != h1 {
			t.Fatalf("got (%q,%v), want match on basename", got, ok)
		}
	})

	t.Run("asset absent from list", func(t *testing.T) {
		body := h1 + "  other.tar.gz\n"
		if _, ok := checksumFor(body, "missing.tar.gz"); ok {
			t.Fatal("expected no match for an asset not listed")
		}
	})
}

func TestIsHexSHA256(t *testing.T) {
	valid := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if !isHexSHA256(valid) {
		t.Error("valid sha256 rejected")
	}
	for _, bad := range []string{"", "abc", valid + "00", "zz" + valid[2:]} {
		if isHexSHA256(bad) {
			t.Errorf("bad hex accepted: %q", bad)
		}
	}
}

func TestVerifyChecksumSum(t *testing.T) {
	const good = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	const bad = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
	gh := &catalog.GitHubRelease{Repo: "acme/tool"}
	asset := "tool_linux_amd64.tar.gz"

	t.Run("verified when published checksum matches", func(t *testing.T) {
		fetch := func(name string) (string, bool) {
			if name == "checksums.txt" {
				return good + "  " + asset + "\n", true
			}
			return "", false
		}
		st, err := verifyChecksumSum(gh, "v1", asset, good, fetch)
		if st != ChecksumVerified || err != nil {
			t.Fatalf("got (%v,%v), want (Verified,nil)", st, err)
		}
	})

	t.Run("mismatch aborts with error", func(t *testing.T) {
		fetch := func(name string) (string, bool) {
			if name == "checksums.txt" {
				return bad + "  " + asset + "\n", true // published != our sum
			}
			return "", false
		}
		st, err := verifyChecksumSum(gh, "v1", asset, good, fetch)
		if st != ChecksumMismatch || err == nil {
			t.Fatalf("got (%v,%v), want (Mismatch, error)", st, err)
		}
	})

	t.Run("unavailable when nothing published", func(t *testing.T) {
		fetch := func(string) (string, bool) { return "", false }
		st, err := verifyChecksumSum(gh, "v1", asset, good, fetch)
		if st != ChecksumUnavailable || err != nil {
			t.Fatalf("got (%v,%v), want (Unavailable,nil)", st, err)
		}
	})
}
