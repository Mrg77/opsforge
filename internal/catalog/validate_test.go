package catalog

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestCatalogBrewFormulasExist checks that every brew reference in the
// catalog resolves to a real formula or cask of the declared kind. It
// queries the public Homebrew API (formulae.brew.sh) rather than the
// local `brew` cache, so results do not depend on which formulas/taps
// happen to be installed on the machine running the test — that machine
// state is exactly what let a renamed cask (google-cloud-sdk ->
// gcloud-cli) and a moved formula (tflint) pass locally yet fail on a
// clean CI runner.
//
// Third-party tapped references ("owner/tap/name") are not on the public
// core/cask API, so their existence is asserted structurally and left to
// the runtime `brew tap`. Set OPSFORGE_SKIP_BREW_VALIDATION=1 to skip.
func TestCatalogBrewFormulasExist(t *testing.T) {
	if os.Getenv("OPSFORGE_SKIP_BREW_VALIDATION") != "" {
		t.Skip("OPSFORGE_SKIP_BREW_VALIDATION set; skipping catalog formula validation")
	}

	cat, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	for _, tool := range cat.Tools() {
		tool := tool
		t.Run(tool.Name, func(t *testing.T) {
			t.Parallel()
			// Tapped refs (owner/tap/name) are handled by `brew tap` at
			// install time and are not on the public core API.
			if strings.Count(tool.Brew, "/") == 2 {
				return
			}
			kind := "formula"
			if tool.Cask {
				kind = "cask"
			}
			switch existsInAPI(client, kind, tool.Brew) {
			case apiFound:
				// good
			case apiWrongKind:
				t.Errorf("brew %q for tool %q exists but not as a %s — check the `cask` flag",
					tool.Brew, tool.Name, kind)
			default:
				t.Errorf("brew %s %q for tool %q does not exist on the Homebrew API "+
					"(renamed or removed?)", kind, tool.Brew, tool.Name)
			}
		})
	}
}

type apiResult int

const (
	apiMissing apiResult = iota
	apiFound
	apiWrongKind
)

// existsInAPI reports whether name resolves on the Homebrew API as the
// given kind, distinguishing "wrong kind" (exists as the other kind) so
// a mislabeled cask/formula gives an actionable error.
func existsInAPI(c *http.Client, kind, name string) apiResult {
	if headOK(c, kind, name) {
		return apiFound
	}
	other := "cask"
	if kind == "cask" {
		other = "formula"
	}
	if headOK(c, other, name) {
		return apiWrongKind
	}
	return apiMissing
}

func headOK(c *http.Client, kind, name string) bool {
	url := "https://formulae.brew.sh/api/" + kind + "/" + name + ".json"
	resp, err := c.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
