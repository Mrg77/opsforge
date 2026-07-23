package installer

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/Mrg77/opsforge/internal/catalog"
)

// ChecksumStatus reports how a downloaded asset was integrity-checked.
// Each value means exactly one thing — in particular a MISMATCH is never
// conflated with "no checksum published", so a caller can never treat a
// tampered asset as merely unverified.
type ChecksumStatus int

const (
	// ChecksumVerified — a published checksum was found and matched.
	ChecksumVerified ChecksumStatus = iota
	// ChecksumUnavailable — no checksum was published for this release, so
	// integrity could not be verified (a warning, not a failure).
	ChecksumUnavailable
	// ChecksumMismatch — a checksum WAS published and did NOT match. The
	// asset must never be installed. Always paired with a non-nil error.
	ChecksumMismatch
	// ChecksumSigned — the strongest state: the published checksum was found
	// AND matched, AND the checksums file itself carries a cosign keyless
	// signature that verified against the expected OIDC identity. This is a
	// strict refinement of ChecksumVerified — the asset is not just intact but
	// provably built by the trusted release workflow. Only reached when cosign
	// is installed locally, a signature (+certificate) was published, and it
	// verified; otherwise verification degrades to ChecksumVerified (see
	// verifyProvenance). A signature that IS present but FAILS to verify is a
	// hard failure surfaced as ChecksumMismatch, never as ChecksumSigned.
	ChecksumSigned
)

// verifyChecksum verifies the SHA-256 of the downloaded asset against a
// checksum published alongside the GitHub release.
//
// Contract (status and error move together, no ambiguity):
//   - (ChecksumVerified, nil)     — a checksum was found and matched.
//   - (ChecksumUnavailable, nil)  — no checksum was published (benign).
//   - (ChecksumMismatch, err)     — a checksum was found and did NOT match;
//     the caller MUST abort the install.
//   - (ChecksumUnavailable, err)  — the asset couldn't be hashed (I/O);
//     the caller MUST abort (err != nil), integrity is unknown.
//
// This mirrors the 2026 supply-chain baseline: never run a downloaded
// binary whose published checksum disagrees, while not blocking the ~85%
// of releases that publish no checksum at all.
func verifyChecksum(gh *catalog.GitHubRelease, repo, tag, asset, archivePath string) (ChecksumStatus, error) {
	sum, err := sha256File(archivePath)
	if err != nil {
		return ChecksumUnavailable, fmt.Errorf("hashing %s: %w", asset, err)
	}
	// fetch resolves a candidate checksum-file name to its body (or ok=false
	// when absent). Split out so the decision logic is testable without a
	// network — see verifyChecksumSum.
	fetch := func(name string) (string, bool) {
		return fetchText(fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, name))
	}
	return verifyChecksumSum(gh, tag, asset, sum, fetch)
}

// verifyChecksumSum is the pure decision core of verifyChecksum: given the
// asset's computed hash and a function that returns the body of a named
// checksum file, it decides verified / unavailable / mismatch. No I/O of
// its own, so it is unit-testable.
func verifyChecksumSum(gh *catalog.GitHubRelease, tag, asset, sum string, fetch func(name string) (string, bool)) (ChecksumStatus, error) {
	for _, name := range checksumCandidates(gh, tag, asset) {
		body, ok := fetch(name)
		if !ok {
			continue
		}
		want, found := checksumFor(body, asset)
		if !found {
			continue
		}
		if strings.EqualFold(want, sum) {
			return ChecksumVerified, nil
		}
		return ChecksumMismatch, fmt.Errorf(
			"checksum mismatch for %s: published %s, got %s (refusing to install)",
			asset, want, sum)
	}
	return ChecksumUnavailable, nil
}

// checksumCandidates lists the release-asset names that might hold the
// checksum, most specific first: an explicit catalog template, then the
// common conventions.
func checksumCandidates(gh *catalog.GitHubRelease, tag, asset string) []string {
	var names []string
	if gh.ChecksumTemplate != "" {
		names = append(names, resolveAssetFor(&catalog.GitHubRelease{
			AssetTemplate: gh.ChecksumTemplate, ArchMap: gh.ArchMap, OSMap: gh.OSMap,
		}, tag, runtime.GOOS, runtime.GOARCH))
	}
	ver := strings.TrimPrefix(tag, "v")
	names = append(names,
		asset+".sha256",
		asset+".sha256sum",
		"checksums.txt",
		"checksums_sha256.txt",
		"SHA256SUMS",
		fmt.Sprintf("%s_%s_checksums.txt", repoName(gh), ver),
	)
	return dedupe(names)
}

// checksumFor finds the hex SHA-256 for `asset` inside a checksum file.
// It handles both the "<hash>  <file>" list format (checksums.txt) and a
// bare "<hash>" single-asset file (foo.tar.gz.sha256).
func checksumFor(body, asset string) (string, bool) {
	base := path.Base(asset)
	sc := bufio.NewScanner(strings.NewReader(body))
	var single string
	lines := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		lines++
		fields := strings.Fields(line)
		if len(fields) == 1 && isHexSHA256(fields[0]) {
			single = fields[0]
			continue
		}
		if len(fields) >= 2 && isHexSHA256(fields[0]) {
			// The filename may carry a leading "*" (binary mode) or a path.
			name := strings.TrimPrefix(fields[len(fields)-1], "*")
			if path.Base(name) == base {
				return strings.ToLower(fields[0]), true
			}
		}
	}
	// A file that is just one bare hash applies to the single asset.
	if single != "" && lines == 1 {
		return strings.ToLower(single), true
	}
	return "", false
}

func isHexSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// fetchText GETs a URL and returns its body as text, or ok=false on any
// non-200 (a missing checksum file is expected, not an error).
func fetchText(url string) (string, bool) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		return "", false
	}
	return string(b), true
}

// fetchBytes GETs a URL and returns its raw body, or ok=false on any non-200
// (a missing signature/certificate is expected, not an error). Mirrors
// fetchText but preserves bytes, which cosign artifacts (.sig/.pem) need.
func fetchBytes(url string) ([]byte, bool) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		return nil, false
	}
	return b, true
}

func repoName(gh *catalog.GitHubRelease) string {
	if i := strings.LastIndex(gh.Repo, "/"); i >= 0 {
		return gh.Repo[i+1:]
	}
	return gh.Repo
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
