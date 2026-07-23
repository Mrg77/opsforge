package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/shellcfg"
	"github.com/Mrg77/opsforge/internal/snapshot"
	"github.com/Mrg77/opsforge/internal/ui"
	"github.com/Mrg77/opsforge/internal/userprofiles"
	"github.com/Mrg77/opsforge/internal/versions"
)

var snapshotOut string

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Export your whole workstation setup to one shareable file",
	Long: `Capture everything opsforge manages — installed tools, your custom
profiles, the shell environment state, the active theme, your guard
policy and the detected version manager — into a single YAML file.

Commit it to your dotfiles repo (or a gist), then rebuild the exact same
workstation anywhere with:  opsforge apply <file-or-url>
or verify a machine against it in CI with: opsforge apply --check <file-or-url>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		if snapshotOut != "-" {
			fmt.Println(ui.Header("opsforge snapshot", "export your whole workstation to one shareable file"))
			fmt.Println()
			fmt.Println(ui.Dim.Render("Scanning your workstation…"))
		}
		statuses := detect.All(cat.Tools())
		profiles, _ := userprofiles.Load()

		s := snapshot.Capture(cat, statuses, profiles, shellcfg.InstalledInZshrc(),
			captureTheme(), captureGuards(), string(versions.Detect()), time.Now())
		data, err := s.Marshal()
		if err != nil {
			return err
		}

		if snapshotOut == "-" {
			fmt.Print(string(data))
			return nil
		}
		if err := os.WriteFile(snapshotOut, data, 0o644); err != nil {
			return err
		}

		fmt.Printf("%s %s\n", ui.OK.Render("✓"), snapshotOut)
		fmt.Printf("  %s\n", ui.Dim.Render(fmt.Sprintf(
			"%d tool(s) · %d profile(s) · shell: %v · theme: %s · guards: %s · versions: %s",
			len(s.Tools), len(s.Profiles), s.Shell.Enabled,
			themeSummary(s.Theme), guardsSummary(s.Guards), managerSummary(s.Versions))))
		fmt.Println()
		fmt.Println("Rebuild this workstation anywhere:")
		fmt.Printf("  opsforge apply %s\n", snapshotOut)
		fmt.Println(ui.Dim.Render("  (commit it to your dotfiles repo and apply from its raw URL)"))
		return nil
	},
}

// captureTheme reads the persisted UI theme. ui.Active.Name is the theme in
// use; ui.ThemePersisted() distinguishes a user choice from auto-resolution
// so the snapshot only pins a theme the user explicitly set.
func captureTheme() snapshot.ThemeState {
	return snapshot.ThemeState{
		Name:      ui.Active.Name,
		Persisted: ui.ThemePersisted(),
	}
}

// captureGuards reads the user's guards.yaml verbatim, or "" when the user
// relies on the built-in default policy. Read passively — never evaluated,
// so it can't trigger a kube/cloud probe.
func captureGuards() string {
	path, err := shellcfg.PolicyPath()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "" // absent → user is on default policy
	}
	return string(data)
}

func themeSummary(t snapshot.ThemeState) string {
	if t.Persisted && t.Name != "" {
		return t.Name
	}
	return "auto"
}

func guardsSummary(g snapshot.GuardState) string {
	if g.YAML != "" {
		return "custom"
	}
	return "default"
}

func managerSummary(v snapshot.VersionState) string {
	if v.Manager != "" {
		return v.Manager
	}
	return "none"
}

func init() {
	snapshotCmd.Flags().StringVarP(&snapshotOut, "output", "o", "opsforge-snapshot.yaml",
		"output file ('-' for stdout)")
	rootCmd.AddCommand(snapshotCmd)
}
