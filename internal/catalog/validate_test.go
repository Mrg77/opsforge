package catalog

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestCatalogBrewFormulasExist checks that every brew reference in the
// catalog resolves to a real formula or cask. It shells out to
// `brew info`, tapping third-party taps first (as the installer does at
// runtime), so a typo in a formula name is caught here instead of by a
// user mid-install.
//
// Skipped when brew is unavailable (e.g. Linux CI); the macOS CI job
// enforces it. Set OPSFORGE_SKIP_BREW_VALIDATION=1 to skip locally.
func TestCatalogBrewFormulasExist(t *testing.T) {
	if os.Getenv("OPSFORGE_SKIP_BREW_VALIDATION") != "" {
		t.Skip("OPSFORGE_SKIP_BREW_VALIDATION set; skipping catalog formula validation")
	}
	if _, err := exec.LookPath("brew"); err != nil {
		t.Skip("brew not installed; skipping catalog formula validation")
	}

	cat, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	// Tap every third-party tap up front (serially) so the parallel
	// `brew info` checks below can resolve tapped formulas.
	taps := map[string]bool{}
	for _, tool := range cat.Tools() {
		if parts := strings.Split(tool.Brew, "/"); len(parts) == 3 {
			taps[parts[0]+"/"+parts[1]] = true
		}
	}
	for tap := range taps {
		if err := exec.Command("brew", "tap", tap).Run(); err != nil {
			t.Logf("could not tap %s (formula check may fail): %v", tap, err)
		}
	}

	for _, tool := range cat.Tools() {
		tool := tool
		t.Run(tool.Name, func(t *testing.T) {
			t.Parallel()
			if out, err := exec.Command("brew", "info", tool.Brew).CombinedOutput(); err != nil {
				t.Errorf("brew formula %q for tool %q does not resolve:\n%s",
					tool.Brew, tool.Name, out)
			}
		})
	}
}
