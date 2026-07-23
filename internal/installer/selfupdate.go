package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/mod/semver"

	"github.com/Mrg77/opsforge/internal/catalog"
)

// SelfRepo is the GitHub repository opsforge updates itself from.
const SelfRepo = "Mrg77/opsforge"

// selfRelease describes how opsforge's own GoReleaser assets are named:
//
//	opsforge_<version>_<os>_<arch>.tar.gz   (version has no leading v)
//
// with checksums in checksums.txt. Reusing catalog.GitHubRelease lets the
// existing asset/checksum resolution and verification apply unchanged.
func selfRelease() *catalog.GitHubRelease {
	return &catalog.GitHubRelease{
		Repo:          SelfRepo,
		AssetTemplate: "opsforge_{version}_{os}_{arch}.tar.gz",
		// GoReleaser emits raw GOOS/GOARCH, so no os/arch remap is needed.
		ChecksumTemplate: "checksums.txt",
	}
}

// SelfUpdateCheck is the result of comparing the running version to the
// latest published release. It is a pure value so the CLI can render it
// either as human UI or JSON, and so it stays testable.
type SelfUpdateCheck struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Available bool   `json:"available"`
	// Dev is true when the running binary is a local/dev build whose version
	// can't be meaningfully compared against a release tag.
	Dev bool `json:"dev"`
}

// isDevVersion reports whether v is a non-release build. GoReleaser injects
// a real tag (e.g. "v0.4.0"); local `go build` leaves the default "dev",
// and an empty string is treated the same way.
func isDevVersion(v string) bool {
	return v == "" || v == "dev"
}

// compareVersions reports whether latest is strictly newer than current,
// using semantic versioning. Both are canonicalized so bare tags with or
// without a leading "v" compare correctly. A dev/unparseable current
// version is never considered "up to date": there is simply nothing to
// compare, which the caller surfaces separately (see NewerAvailable).
func compareVersions(current, latest string) (newer bool) {
	cv := canonicalVersion(current)
	lv := canonicalVersion(latest)
	if !semver.IsValid(cv) || !semver.IsValid(lv) {
		return false
	}
	return semver.Compare(lv, cv) > 0
}

// canonicalVersion turns a bare or v-prefixed version into a value
// semver.Compare accepts, or "" when it isn't a version at all.
func canonicalVersion(v string) string {
	if v == "" {
		return ""
	}
	if v[0] != 'v' {
		v = "v" + v
	}
	return semver.Canonical(v)
}

// NewerAvailable decides, for a running version and a latest tag, whether an
// update should be offered. It is pure (no I/O) and the unit-tested core of
// self-update's decision:
//
//   - a dev build reports Dev=true and Available=false (nothing to compare);
//   - otherwise Available is true only when latest is strictly newer.
func NewerAvailable(current, latest string) SelfUpdateCheck {
	c := SelfUpdateCheck{Current: current, Latest: latest}
	if isDevVersion(current) {
		c.Dev = true
		return c
	}
	c.Available = compareVersions(current, latest)
	return c
}

// LatestSelfVersion queries GitHub for opsforge's latest release tag. It is
// a thin exported wrapper over latestTag so the CLI layer needn't know the
// repo constant or the unexported helper.
func LatestSelfVersion() (string, error) {
	return latestTag(SelfRepo)
}

// CheckForSelfUpdate fetches the latest release and compares it to current.
// Network-touching, so it is not exercised by the unit tests; the pure
// decision it wraps (NewerAvailable) is.
func CheckForSelfUpdate(current string) (SelfUpdateCheck, error) {
	latest, err := LatestSelfVersion()
	if err != nil {
		return SelfUpdateCheck{Current: current}, err
	}
	return NewerAvailable(current, latest), nil
}

// SelfUpdateDownload is the staged result of DownloadSelfUpdate: the path to
// the verified-but-not-yet-installed binary, the temp dir the caller must
// clean up, a non-fatal warning, and whether the checksums file's cosign
// keyless signature verified (opsforge signs its own releases).
type SelfUpdateDownload struct {
	BinPath string
	TmpDir  string
	Warning string
	// Signed is true when the published checksums.txt carried a cosign keyless
	// signature that verified against opsforge's release-workflow identity —
	// i.e. verifyChecksumProvenance returned ChecksumSigned. False means the
	// checksum still matched (best-effort: cosign absent or no signature
	// published), never that a signature failed (that path returns an error).
	Signed bool
}

// DownloadSelfUpdate downloads, checksum-verifies and extracts the opsforge
// binary for `tag` and the running platform into a temporary directory,
// returning the staged binary path and provenance details.
//
// It performs NO in-place replacement — that is ApplySelfUpdate's job — so a
// caller can stage the download and only swap the running binary once the
// checksum (and, when available, the signature) has passed. A checksum
// MISMATCH or a published-but-invalid signature (tampered asset) returns an
// error and the staged file is discarded by the caller via os.RemoveAll.
//
// Because opsforge signs its own checksums.txt (cosign keyless, see
// .goreleaser.yaml), the self-update path verifies that signature when cosign
// is installed locally, upgrading integrity to full provenance.
func DownloadSelfUpdate(tag string) (SelfUpdateDownload, error) {
	gh := selfRelease()
	asset := resolveAssetFor(gh, tag, runtime.GOOS, runtime.GOARCH)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", gh.Repo, tag, asset)

	tmpDir, err := os.MkdirTemp("", "opsforge-selfupdate-*")
	if err != nil {
		return SelfUpdateDownload{}, err
	}

	archivePath := filepath.Join(tmpDir, asset)
	if err := download(url, archivePath); err != nil {
		os.RemoveAll(tmpDir)
		return SelfUpdateDownload{}, fmt.Errorf("downloading %s: %w", url, err)
	}

	// Supply-chain gate: verify the SHA-256 against the published
	// checksums.txt BEFORE the binary is ever extracted or run, and — since we
	// sign our own releases — verify the cosign keyless signature over that
	// checksums file too. A mismatch, a bad signature, or an I/O failure that
	// leaves integrity unknown aborts; a release with no checksum installs with
	// a warning; a matched checksum with no verifiable signature stays a
	// (silent) success.
	res := SelfUpdateDownload{TmpDir: tmpDir}
	status, verr := verifyChecksumProvenance(
		gh, gh.Repo, tag, asset, archivePath,
		"checksums.txt", selfCertIdentityRegexp, oidcIssuerGitHubActions)
	switch {
	case status == ChecksumMismatch, verr != nil:
		os.RemoveAll(tmpDir)
		return SelfUpdateDownload{}, verr
	case status == ChecksumUnavailable:
		res.Warning = "no published checksum for this release — integrity not verified"
	case status == ChecksumSigned:
		res.Signed = true
	}

	src, err := extractBinary(archivePath, tmpDir, "opsforge", "")
	if err != nil {
		os.RemoveAll(tmpDir)
		return SelfUpdateDownload{}, fmt.Errorf("extracting opsforge: %w", err)
	}
	res.BinPath = src
	return res, nil
}

// ApplySelfUpdate atomically replaces the executable at dest with the
// verified binary at src. On Unix a running executable can be renamed over,
// so the update takes effect on the next invocation. The rename is atomic
// when src and dest share a filesystem; moveFile falls back to copy+remove
// across filesystems (temp dir vs the install dir).
func ApplySelfUpdate(src, dest string) error {
	if err := os.Chmod(src, 0o755); err != nil {
		return err
	}
	// Stage next to the destination so the final swap is a same-directory
	// rename (atomic) even when src lived on a different filesystem.
	staged := dest + ".opsforge-new"
	if err := moveFile(src, staged); err != nil {
		return err
	}
	if err := os.Rename(staged, dest); err != nil {
		os.Remove(staged)
		return err
	}
	return nil
}
