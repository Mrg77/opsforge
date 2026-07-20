package catalog

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestCatalogGitHubAssetsResolve verifies that every tool carrying a
// `github:` block resolves, for both darwin and linux on amd64/arm64, to
// an asset that actually exists in the repo's latest release. A wrong
// template (bad arch token, renamed asset) would otherwise fail only at
// install time on a brew-less host — exactly where it is hardest to
// diagnose. Queries the GitHub API, so it is machine-independent.
//
// Set OPSFORGE_SKIP_BREW_VALIDATION=1 to skip (shares the flag with the
// brew validation; both hit the network). Honors GITHUB_TOKEN.
func TestCatalogGitHubAssetsResolve(t *testing.T) {
	if os.Getenv("OPSFORGE_SKIP_BREW_VALIDATION") != "" {
		t.Skip("network validation skipped via OPSFORGE_SKIP_BREW_VALIDATION")
	}
	cat, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 20 * time.Second}

	platforms := []struct{ goos, goarch string }{
		{"darwin", "amd64"}, {"darwin", "arm64"},
		{"linux", "amd64"}, {"linux", "arm64"},
	}

	for _, tool := range cat.Tools() {
		if tool.GitHub == nil {
			continue
		}
		tool := tool
		t.Run(tool.Name, func(t *testing.T) {
			t.Parallel()
			assets, tag, err := latestReleaseAssets(client, tool.GitHub.Repo)
			if err != nil {
				t.Fatalf("fetching latest release for %s: %v", tool.GitHub.Repo, err)
			}
			for _, p := range platforms {
				want := renderAsset(tool.GitHub, tag, p.goos, p.goarch)
				if !assets[want] {
					t.Errorf("%s: asset %q (for %s/%s) not found in %s release %s",
						tool.Name, want, p.goos, p.goarch, tool.GitHub.Repo, tag)
				}
			}
		})
	}
}

// renderAsset mirrors installer.resolveAssetFor so the catalog can be
// validated without importing the installer package (which would create
// an import cycle: installer already imports catalog).
func renderAsset(gh *GitHubRelease, tag, goos, goarch string) string {
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

func latestReleaseAssets(c *http.Client, repo string) (map[string]bool, string, error) {
	req, _ := http.NewRequest(http.MethodGet,
		"https://api.github.com/repos/"+repo+"/releases/latest", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", &httpError{resp.StatusCode}
	}
	var rel struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, "", err
	}
	set := make(map[string]bool, len(rel.Assets))
	for _, a := range rel.Assets {
		set[a.Name] = true
	}
	return set, rel.TagName, nil
}

type httpError struct{ code int }

func (e *httpError) Error() string { return "github api status " + http.StatusText(e.code) }
