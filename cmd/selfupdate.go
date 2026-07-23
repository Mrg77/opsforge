package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/installer"
	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/ui"
)

// self groups commands that operate on the opsforge binary itself:
// reporting its version and updating it in place from GitHub releases. The
// update path is supply-chain-safe — it never runs a downloaded binary whose
// published SHA-256 disagrees (see installer.DownloadSelfUpdate).
var selfCmd = &cobra.Command{
	Use:   "self",
	Short: "Manage the opsforge binary itself (version, update)",
}

var selfVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the running opsforge version",
	RunE: func(cmd *cobra.Command, args []string) error {
		if output.JSON {
			return output.Emit(struct {
				Version string `json:"version"`
				Dev     bool   `json:"dev"`
			}{version, version == "dev" || version == ""})
		}
		fmt.Println(version)
		return nil
	},
}

var (
	selfUpdateCheck bool
	selfUpdateYes   bool
)

var selfUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update opsforge to the latest release (checksum-verified)",
	Long: `Query the latest opsforge release, and — when it is newer than the
running version — download the binary for this platform, verify its
published SHA-256 checksum, and replace the current binary atomically.

  --check   report whether an update is available without installing it;
            exits non-zero when one is (handy in CI/cron)
  --yes     don't ask for confirmation before installing
  --json    machine-readable output (pairs well with --check)

A dev build (local 'go build') can't be compared against a release tag, so
update is a no-op there. A checksum mismatch always aborts the install.`,
	RunE: runSelfUpdate,
}

func runSelfUpdate(cmd *cobra.Command, args []string) error {
	check, err := installer.CheckForSelfUpdate(version)
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	// --check: report only, never install. Exit non-zero when an update is
	// available so a CI/cron job can act on it.
	if selfUpdateCheck {
		if output.JSON {
			if err := output.Emit(check); err != nil {
				return err
			}
		} else {
			printCheck(check)
		}
		if check.Available {
			os.Exit(1)
		}
		return nil
	}

	if check.Dev {
		fmt.Println(ui.Header("opsforge self update", "development build"))
		fmt.Println()
		fmt.Printf("  %s running a dev build (%s) — can't compare against a release.\n",
			ui.WarnMark(), ui.Accent.Render(version))
		fmt.Println(ui.Dim.Render("  Install a released build to enable self-update."))
		return nil
	}

	if !check.Available {
		fmt.Println(ui.Header("opsforge self update", "already up to date"))
		fmt.Println()
		fmt.Printf("  %s opsforge %s is the latest release.\n",
			ui.OKMark(), ui.Accent.Render(check.Current))
		return nil
	}

	fmt.Println(ui.Header("opsforge self update", ""))
	fmt.Println()
	fmt.Printf("  %s update available: %s %s %s\n\n",
		ui.MarkUpdate, ui.Dim.Render(check.Current), ui.MarkArrow, ui.Accent.Render(check.Latest))

	if !selfUpdateYes && !confirm("Download and install this update?") {
		fmt.Println(ui.Dim.Render("  aborted."))
		return nil
	}

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating the running binary: %w", err)
	}

	dl, err := installer.DownloadSelfUpdate(check.Latest)
	if err != nil {
		// A checksum mismatch or an invalid signature surfaces here as a hard
		// error — the tampered asset is never installed.
		return err
	}
	defer os.RemoveAll(dl.TmpDir)

	if dl.Warning != "" {
		fmt.Printf("  %s %s\n", ui.WarnMark(), ui.Warn.Render(dl.Warning))
	}
	if dl.Signed {
		fmt.Printf("  %s %s\n", ui.OKMark(),
			ui.Dim.Render("signature verified (cosign, keyless)"))
	}

	if err := installer.ApplySelfUpdate(dl.BinPath, self); err != nil {
		return fmt.Errorf("replacing %s: %w", self, err)
	}

	fmt.Printf("  %s updated to %s\n", ui.OKMark(), ui.Accent.Render(check.Latest))
	fmt.Println(ui.Dim.Render("  Run `opsforge self version` to confirm."))
	return nil
}

// printCheck renders the --check result for humans (the JSON path uses
// output.Emit directly).
func printCheck(c installer.SelfUpdateCheck) {
	switch {
	case c.Dev:
		fmt.Printf("%s dev build (%s) — no release to compare against\n",
			ui.WarnMark(), c.Current)
	case c.Available:
		fmt.Printf("%s update available: %s %s %s\n",
			ui.MarkUpdate, c.Current, ui.MarkArrow, ui.Accent.Render(c.Latest))
	default:
		fmt.Printf("%s up to date (%s)\n", ui.OKMark(), c.Current)
	}
}

// confirm asks a yes/no question on stdin, defaulting to no. Kept simple and
// local — self-update is the only command that prompts this way.
func confirm(prompt string) bool {
	fmt.Printf("  %s %s [y/N] ", ui.Prompt, prompt)
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(sc.Text()))
	return answer == "y" || answer == "yes"
}

func init() {
	selfUpdateCmd.Flags().BoolVar(&selfUpdateCheck, "check", false,
		"only report whether an update is available (exit non-zero if so)")
	selfUpdateCmd.Flags().BoolVar(&selfUpdateYes, "yes", false,
		"skip the confirmation prompt")
	selfCmd.AddCommand(selfVersionCmd, selfUpdateCmd)
	rootCmd.AddCommand(selfCmd)
}
