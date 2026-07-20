// Package installer installs catalog tools. It picks a backend at
// runtime: Homebrew when available (macOS, Linuxbrew), otherwise a
// GitHub-releases binary download for hosts without brew (bare Linux
// servers, CI images). The backend can be forced with OPSFORGE_BACKEND.
package installer

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Mrg77/opsforge/internal/catalog"
)

// Backend identifies how a tool is installed.
type Backend string

const (
	BackendBrew   Backend = "brew"
	BackendGitHub Backend = "github"
	BackendNone   Backend = "none"
)

// Result reports the outcome of one installation or upgrade.
type Result struct {
	Err error
	// OutputTail holds the last lines of backend output, kept only on
	// failure so the UI can show why.
	OutputTail string
	// NotBrewManaged is set when an upgrade failed only because the
	// tool was installed by other means (manual download, cloud SDK...).
	NotBrewManaged bool
	// Backend records which backend handled the operation.
	Backend Backend
}

// brewAvailable reports whether Homebrew is on PATH.
func brewAvailable() bool {
	_, err := exec.LookPath("brew")
	return err == nil
}

// Available reports whether any backend can install tools at all. The
// GitHub backend needs nothing beyond network access, so this is only
// false when explicitly pinned to a missing brew.
func Available() bool {
	return backendFor(catalog.Tool{Brew: "x"}) != BackendNone
}

// BrewAvailable is exported for the shell/doctor layer to report state.
func BrewAvailable() bool { return brewAvailable() }

// backendFor decides which backend to use for a tool, honoring the
// OPSFORGE_BACKEND override.
func backendFor(t catalog.Tool) Backend {
	forced := Backend(os.Getenv("OPSFORGE_BACKEND"))
	switch forced {
	case BackendBrew:
		if brewAvailable() {
			return BackendBrew
		}
		return BackendNone
	case BackendGitHub:
		if t.GitHub != nil {
			return BackendGitHub
		}
		return BackendNone
	}
	// Auto: prefer brew, fall back to a GitHub release when available.
	if brewAvailable() && t.Brew != "" {
		return BackendBrew
	}
	if t.GitHub != nil {
		return BackendGitHub
	}
	return BackendNone
}

// BackendFor exposes the resolved backend for a tool (for doctor/UI).
func BackendFor(t catalog.Tool) Backend { return backendFor(t) }

// Install installs a tool via its resolved backend.
func Install(t catalog.Tool) Result {
	switch backendFor(t) {
	case BackendBrew:
		res := brewInstall(t.Brew, t.Cask)
		res.Backend = BackendBrew
		return res
	case BackendGitHub:
		res := InstallFromGitHub(t)
		res.Backend = BackendGitHub
		return res
	default:
		return Result{
			Backend: BackendNone,
			Err: fmt.Errorf(
				"no backend for %s: install Homebrew (https://brew.sh) or add a github release to the catalog",
				t.Name),
		}
	}
}

// Upgrade upgrades a tool. Brew upgrades in place; the GitHub backend
// re-downloads the latest release, which is itself an upgrade.
func Upgrade(t catalog.Tool) Result {
	switch backendFor(t) {
	case BackendBrew:
		res := brewUpgrade(t.Brew, t.Cask)
		res.Backend = BackendBrew
		return res
	case BackendGitHub:
		res := InstallFromGitHub(t)
		res.Backend = BackendGitHub
		return res
	default:
		return Result{Backend: BackendNone, NotBrewManaged: true,
			Err: fmt.Errorf("no backend for %s", t.Name)}
	}
}

func brewInstall(formula string, cask bool) Result {
	if err := ensureTapped(formula); err != nil {
		return Result{Err: err}
	}
	args := []string{"install"}
	if cask {
		args = append(args, "--cask")
	}
	args = append(args, formula)
	out, err := exec.Command("brew", args...).CombinedOutput()
	if err != nil {
		text := string(out)
		if isLinkConflict(text) {
			if lerr := brewLinkOverwrite(leafFormula(formula), cask); lerr == nil {
				return Result{}
			}
		}
		return Result{
			Err:        fmt.Errorf("brew install %s: %w", formula, err),
			OutputTail: tail(text, 6),
		}
	}
	return Result{}
}

// ensureTapped taps a formula's third-party tap when the reference is of
// the form "owner/tap/formula". Homebrew does not auto-tap on install, so
// installs of tapped formulas (hashicorp/tap, fluxcd/tap, xo/xo...) fail
// without this. Core formulas (no slashes) are left untouched.
func ensureTapped(formula string) error {
	parts := strings.Split(formula, "/")
	if len(parts) != 3 {
		return nil
	}
	tap := parts[0] + "/" + parts[1]
	out, err := exec.Command("brew", "tap", tap).CombinedOutput()
	if err != nil {
		return fmt.Errorf("brew tap %s: %w\n%s", tap, err, tail(string(out), 4))
	}
	return nil
}

func brewUpgrade(formula string, cask bool) Result {
	if err := ensureTapped(formula); err != nil {
		return Result{Err: err}
	}
	args := []string{"upgrade"}
	if cask {
		args = append(args, "--cask")
	}
	args = append(args, formula)
	out, err := exec.Command("brew", args...).CombinedOutput()
	if err == nil {
		return Result{}
	}
	text := string(out)
	switch {
	case strings.Contains(text, "No such keg"), strings.Contains(text, "is not installed"):
		return Result{NotBrewManaged: true, Err: err}
	case isLinkConflict(text):
		// The new version was installed but brew could not symlink it
		// because another formula (e.g. docker-completion) owns some of
		// the same files. Force the link — the upgrade itself succeeded.
		if lerr := brewLinkOverwrite(leafFormula(formula), cask); lerr != nil {
			return Result{
				Err:        fmt.Errorf("brew link --overwrite %s: %w", formula, lerr),
				OutputTail: tail(text, 6),
			}
		}
		return Result{}
	default:
		return Result{
			Err:        fmt.Errorf("brew upgrade %s: %w", formula, err),
			OutputTail: tail(text, 6),
		}
	}
}

// isLinkConflict recognizes brew's "conflicting files" / "already linked"
// failure that leaves the new version installed but unlinked.
func isLinkConflict(brewOutput string) bool {
	return strings.Contains(brewOutput, "conflicting files") ||
		strings.Contains(brewOutput, "link --overwrite") ||
		strings.Contains(brewOutput, "already linked")
}

// brewLinkOverwrite force-links a formula, resolving file conflicts left
// by an upgrade. Casks are not linked, so this is a no-op for them.
func brewLinkOverwrite(formula string, cask bool) error {
	if cask {
		return nil
	}
	return exec.Command("brew", "link", "--overwrite", formula).Run()
}

// leafFormula strips a tap prefix: "hashicorp/tap/vault" -> "vault".
func leafFormula(formula string) string {
	if i := strings.LastIndex(formula, "/"); i >= 0 {
		return formula[i+1:]
	}
	return formula
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
