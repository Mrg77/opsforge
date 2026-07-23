package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/cve"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/installer"
	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/shellcfg"
	"github.com/Mrg77/opsforge/internal/ui"
	"github.com/Mrg77/opsforge/internal/userprofiles"
	"github.com/Mrg77/opsforge/internal/versions"
)

// printSecurityLine shows the cached CVE status (instant, no network). If
// the cache is missing or stale it kicks off a detached background refresh
// so the next `status`/prompt is accurate — the current call never waits.
func printSecurityLine() {
	s, ok := cve.Load()
	switch {
	case !ok || s.ScannedAt.IsZero():
		fmt.Printf("  %s %s\n", ui.Label("Security", 10),
			ui.Dim.Render("scan pending — `opsforge audit` for a full report"))
	case s.HighOrCritical > 0:
		fmt.Printf("  %s %s %s\n", ui.Label("Security", 10),
			ui.Err.Render(fmt.Sprintf("%s %d tool(s) with HIGH/CRITICAL CVEs", ui.MarkErr, s.HighOrCritical)),
			ui.Dim.Render("— `opsforge audit`"))
	case s.Vulnerable > 0:
		fmt.Printf("  %s %s %s\n", ui.Label("Security", 10),
			ui.Warn.Render(fmt.Sprintf("%s %d tool(s) with CVEs", ui.MarkWarn, s.Vulnerable)),
			ui.Dim.Render("— `opsforge audit`"))
	default:
		fmt.Printf("  %s %s\n", ui.Label("Security", 10),
			ui.OK.Render(ui.MarkOK+" no known CVEs"))
	}
	if !ok || s.Stale(cve.DefaultTTL, time.Now().UTC()) {
		refreshCVECacheInBackground()
	}
}

// refreshCVECacheInBackground spawns `opsforge cve refresh` detached, so a
// stale cache updates without holding up the current command.
func refreshCVECacheInBackground() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exe, "cve", "refresh")
	cmd.Stdout, cmd.Stderr = nil, nil
	// Detach: we don't wait, and we don't want it killed when we exit.
	_ = cmd.Start()
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "A one-glance cockpit of your DevOps workstation",
	Long: `A compact dashboard: how many tools are installed, how many have updates,
whether the shell environment is on, and your active theme — everything at
a glance. Run 'opsforge' (no args) for the interactive picker.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		statuses := detect.AllWithOutdated(cat.Tools())

		installed, outdated := 0, 0
		for _, t := range cat.Tools() {
			s := statuses[t.Name]
			if s.Installed {
				installed++
			}
			if s.Outdated {
				outdated++
			}
		}
		total := len(cat.Tools())
		userps, _ := userprofiles.Load()
		shellOn := shellcfg.InstalledInZshrc()

		vm := ""
		if mgr := versions.Detect(); mgr != versions.None {
			vm = string(mgr)
		}
		backend := "github"
		if installer.BrewAvailable() {
			backend = "homebrew+github"
		}

		if output.JSON {
			names := make([]string, 0, len(userps))
			for _, p := range userps {
				names = append(names, p.Name)
			}
			return output.Emit(struct {
				ToolsInstalled int      `json:"tools_installed"`
				ToolsTotal     int      `json:"tools_total"`
				UpdatesPending int      `json:"updates_pending"`
				ShellLayer     bool     `json:"shell_layer"`
				VersionManager string   `json:"version_manager,omitempty"`
				Backend        string   `json:"backend"`
				Theme          string   `json:"theme"`
				Profiles       []string `json:"profiles"`
			}{installed, total, outdated, shellOn, vm, backend, ui.Active.Name, names})
		}

		fmt.Println(ui.Header("opsforge", "your DevOps workstation at a glance"))
		fmt.Println()

		// Toolbox line with a coverage bar.
		fmt.Printf("  %s %s  %s\n",
			ui.Label("Toolbox", 10),
			ui.Bar(installed, total, 20),
			ui.Dim.Render(fmt.Sprintf("%d/%d installed", installed, total)))

		// Updates.
		if outdated > 0 {
			fmt.Printf("  %s %s %s\n",
				ui.Label("Updates", 10),
				ui.Warn.Render(fmt.Sprintf("%s %d available", ui.MarkUpdate, outdated)),
				ui.Dim.Render("— run `opsforge upgrade -u`"))
		} else if installed > 0 {
			fmt.Printf("  %s %s\n", ui.Label("Updates", 10),
				ui.OK.Render(ui.MarkOK+" everything up to date"))
		}

		// Security — from the cached CVE scan, so status never blocks on
		// the network. A stale (or missing) cache triggers a background
		// refresh for next time.
		if installed > 0 {
			printSecurityLine()
		}

		// Shell environment.
		shellVal := ui.Dim.Render("off — `opsforge shell install`")
		if shellOn {
			shellVal = ui.OK.Render(ui.MarkOK + " active")
		}
		fmt.Printf("  %s %s\n", ui.Label("Shell", 10), shellVal)

		// Version manager.
		vmLine := ui.Dim.Render("none — install mise for `opsforge use`")
		if vm != "" {
			vmLine = ui.OK.Render(ui.MarkOK + " " + vm)
		}
		fmt.Printf("  %s %s\n", ui.Label("Versions", 10), vmLine)

		// Backend + theme footer.
		backendLine := "GitHub releases"
		if installer.BrewAvailable() {
			backendLine = "Homebrew + GitHub"
		}
		fmt.Printf("  %s %s\n", ui.Label("Backend", 10), ui.Dim.Render(backendLine))
		theme := ui.Accent.Render(ui.Active.Name)
		switch {
		case os.Getenv("OPSFORGE_THEME") != "":
			theme += ui.Dim.Render(" (from $OPSFORGE_THEME)")
		case !ui.ThemePersisted():
			theme += ui.Dim.Render(" (auto — `opsforge theme set <name>` to change)")
		}
		fmt.Printf("  %s %s\n", ui.Label("Theme", 10), theme)

		if len(userps) > 0 {
			names := make([]string, 0, len(userps))
			for _, p := range userps {
				names = append(names, p.Name)
			}
			fmt.Printf("  %s %s\n", ui.Label("Profiles", 10),
				ui.Dim.Render(strings.Join(names, ", ")))
		}

		fmt.Println()
		fmt.Println(ui.Dim.Render("  Run `opsforge` to open the picker · `opsforge doctor` for a full check"))
		// One discreet pointer to a non-obvious feature, so it gets found.
		fmt.Println(ui.Faint.Render("  Tip: `opsforge audit --secrets` scans your tools for CVEs and your shell for leaked credentials"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
