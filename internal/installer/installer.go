// Package installer installs catalog tools through Homebrew, the only
// backend supported for now (macOS and Linuxbrew). A GitHub-releases
// binary backend is the planned fallback for brew-less hosts.
package installer

import (
	"fmt"
	"os/exec"
	"strings"
)

// Result reports the outcome of one installation or upgrade.
type Result struct {
	Err error
	// OutputTail holds the last lines of brew output, kept only on
	// failure so the UI can show why.
	OutputTail string
	// NotBrewManaged is set when an upgrade failed only because the
	// tool was installed by other means (manual download, cloud SDK...).
	NotBrewManaged bool
}

// Available reports whether the Homebrew backend can run at all.
func Available() bool {
	_, err := exec.LookPath("brew")
	return err == nil
}

// Install runs `brew install [--cask] <formula>` and blocks until done.
func Install(formula string, cask bool) Result {
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

// Upgrade runs `brew upgrade [--cask] <formula>`. Homebrew exits 0 when
// the formula is already current, so success means "up to date or newer".
func Upgrade(formula string, cask bool) Result {
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
