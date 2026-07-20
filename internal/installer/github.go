package installer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Mrg77/opsforge/internal/catalog"
)

// BinDir is where the GitHub backend installs binaries. It mirrors the
// install.sh default so a self-installed opsforge and the tools it
// installs share one directory on PATH.
func BinDir() string {
	if d := os.Getenv("OPSFORGE_BIN_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".local/bin"
	}
	return filepath.Join(home, ".local", "bin")
}

var httpClient = &http.Client{Timeout: 60 * time.Second}

// InstallFromGitHub downloads the tool's release asset for the current
// OS/arch, extracts the binary and installs it into BinDir.
func InstallFromGitHub(t catalog.Tool) Result {
	if t.GitHub == nil {
		return Result{Err: fmt.Errorf("%s has no github release configured", t.Name)}
	}
	gh := t.GitHub

	tag, err := latestTag(gh.Repo)
	if err != nil {
		return Result{Err: err}
	}

	asset := resolveAsset(gh, tag)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", gh.Repo, tag, asset)

	tmp, err := os.MkdirTemp("", "opsforge-*")
	if err != nil {
		return Result{Err: err}
	}
	defer os.RemoveAll(tmp)

	archivePath := filepath.Join(tmp, asset)
	if err := download(url, archivePath); err != nil {
		return Result{Err: fmt.Errorf("downloading %s: %w", url, err)}
	}

	binName := t.Bin
	src, err := extractBinary(archivePath, tmp, binName, gh.BinInArchive)
	if err != nil {
		return Result{Err: fmt.Errorf("extracting %s: %w", t.Name, err)}
	}

	dir := BinDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{Err: err}
	}
	dest := filepath.Join(dir, binName)
	if err := moveFile(src, dest); err != nil {
		return Result{Err: err}
	}
	if err := os.Chmod(dest, 0o755); err != nil {
		return Result{Err: err}
	}
	return Result{}
}

// resolveAsset fills the asset template for the running platform.
func resolveAsset(gh *catalog.GitHubRelease, tag string) string {
	return resolveAssetFor(gh, tag, runtime.GOOS, runtime.GOARCH)
}

// resolveAssetFor fills the asset template for a given os/arch, applying
// the tool's per-platform name overrides. Split out for testing.
func resolveAssetFor(gh *catalog.GitHubRelease, tag, goos, goarch string) string {
	osName := goos
	if v, ok := gh.OSMap[goos]; ok {
		osName = v
	}
	arch := goarch
	if v, ok := gh.ArchMap[goarch]; ok {
		arch = v
	}
	r := strings.NewReplacer(
		"{os}", osName,
		"{arch}", arch,
		"{version}", strings.TrimPrefix(tag, "v"),
		"{tag}", tag,
	)
	return r.Replace(gh.AssetTemplate)
}

type ghRelease struct {
	TagName string `json:"tag_name"`
}

// latestTag queries the GitHub API for a repo's latest release tag. It
// honors GITHUB_TOKEN to avoid the low unauthenticated rate limit.
func latestTag(repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api %s: %s", repo, resp.Status)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("no release found for %s", repo)
	}
	return rel.TagName, nil
}

func download(url, dest string) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %s", resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// extractBinary pulls the wanted executable out of an archive (tar.gz,
// tgz or zip) or returns the asset itself when it is a raw binary. It
// returns the path to the extracted binary on disk.
func extractBinary(archivePath, dir, binName, binInArchive string) (string, error) {
	want := binName
	if binInArchive != "" {
		want = binInArchive
	}
	switch {
	case strings.HasSuffix(archivePath, ".tar.gz"), strings.HasSuffix(archivePath, ".tgz"):
		return extractTarGz(archivePath, dir, want)
	case strings.HasSuffix(archivePath, ".zip"):
		return extractZip(archivePath, dir, want)
	default:
		// Raw binary asset — nothing to unpack.
		return archivePath, nil
	}
}

// matchesBinary reports whether an archive entry is the binary we want,
// matching either the full in-archive path or just the basename.
func matchesBinary(entryPath, want string) bool {
	return entryPath == want || path.Base(entryPath) == path.Base(want)
}

func extractTarGz(archivePath, dir, want string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.Typeflag != tar.TypeReg || !matchesBinary(hdr.Name, want) {
			continue
		}
		out := filepath.Join(dir, path.Base(want))
		if err := writeFile(out, tr); err != nil {
			return "", err
		}
		return out, nil
	}
	return "", fmt.Errorf("binary %q not found in archive", want)
}

func extractZip(archivePath, dir, want string) (string, error) {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer zr.Close()
	for _, zf := range zr.File {
		if zf.FileInfo().IsDir() || !matchesBinary(zf.Name, want) {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			return "", err
		}
		out := filepath.Join(dir, path.Base(want))
		err = writeFile(out, rc)
		rc.Close()
		if err != nil {
			return "", err
		}
		return out, nil
	}
	return "", fmt.Errorf("binary %q not found in archive", want)
}

func writeFile(dest string, r io.Reader) error {
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

// moveFile renames src to dest, falling back to copy+remove when they
// live on different filesystems (os.TempDir vs $HOME).
func moveFile(src, dest string) error {
	if err := os.Rename(src, dest); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := writeFile(dest, in); err != nil {
		return err
	}
	return os.Remove(src)
}
