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
	args := []string{"install"}
	if cask {
		args = append(args, "--cask")
	}
	args = append(args, formula)
	out, err := exec.Command("brew", args...).CombinedOutput()
	if err != nil {
		return Result{
			Err:        fmt.Errorf("brew install %s: %w", formula, err),
			OutputTail: tail(string(out), 6),
		}
	}
	return Result{}
}

func brewUpgrade(formula string, cask bool) Result {
	args := []string{"upgrade"}
	if cask {
		args = append(args, "--cask")
	}
	args = append(args, formula)
	out, err := exec.Command("brew", args...).CombinedOutput()
	if err != nil {
		text := string(out)
		if strings.Contains(text, "No such keg") ||
			strings.Contains(text, "is not installed") {
			return Result{NotBrewManaged: true, Err: err}
		}
		return Result{
			Err:        fmt.Errorf("brew upgrade %s: %w", formula, err),
			OutputTail: tail(text, 6),
		}
	}
	return Result{}
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
