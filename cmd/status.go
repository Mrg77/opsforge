package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/cve"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/i18n"
	"github.com/Mrg77/opsforge/internal/installer"
	"github.com/Mrg77/opsforge/internal/notices"
	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/shellcfg"
	"github.com/Mrg77/opsforge/internal/ui"
	"github.com/Mrg77/opsforge/internal/userprofiles"
	"github.com/Mrg77/opsforge/internal/versions"
)

// printPostureLine shows the workstation security-posture score + grade from the
// cached notices digest (CVEs + secrets + updates). Instant, no network.
func printPostureLine() {
	d, ok := notices.Load()
	if !ok || d.RefreshedAt.IsZero() {
		return // nothing scanned yet; the Security line already invites a scan
	}
	p := d.PostureScore()
	style := ui.OK
	switch {
	case p.Score < 50:
		style = ui.Err
	case p.Score < 75:
		style = ui.Warn
	}
	fmt.Printf("  %s %s %s\n", ui.Label(i18n.T("status.label.posture"), 10),
		style.Render(fmt.Sprintf("%d/100  %s", p.Score, p.Grade)),
		ui.Dim.Render(i18n.T("status.posture.hint")))
}

// printSecurityLine shows the cached CVE status (instant, no network). If
// the cache is missing or stale it kicks off a detached background refresh
// so the next `status`/prompt is accurate — the current call never waits.
func printSecurityLine() {
	s, ok := cve.Load()
	secLabel := i18n.T("status.label.security")
	switch {
	case !ok || s.ScannedAt.IsZero():
		fmt.Printf("  %s %s\n", ui.Label(secLabel, 10),
			ui.Dim.Render(i18n.T("status.security.pending")))
	case s.HighOrCritical > 0:
		fmt.Printf("  %s %s %s\n", ui.Label(secLabel, 10),
			ui.Err.Render(ui.MarkErr+" "+i18n.T("status.security.highcrit", i18n.V("n", strconv.Itoa(s.HighOrCritical)))),
			ui.Dim.Render(i18n.T("status.security.audithint")))
	case s.Vulnerable > 0:
		fmt.Printf("  %s %s %s\n", ui.Label(secLabel, 10),
			ui.Warn.Render(ui.MarkWarn+" "+i18n.T("status.security.cves", i18n.V("n", strconv.Itoa(s.Vulnerable)))),
			ui.Dim.Render(i18n.T("status.security.audithint")))
	default:
		fmt.Printf("  %s %s\n", ui.Label(secLabel, 10),
			ui.OK.Render(ui.MarkOK+" "+i18n.T("status.security.clean")))
	}
	if !ok || s.Stale(cve.DefaultTTL, time.Now().UTC()) {
		refreshCVECacheInBackground()
	}
}

// refreshCVECacheInBackground spawns `opsforge cve refresh` detached, so a
// stale cache updates without holding up the current command.
func refreshCVECacheInBackground() {
	spawnDetached("cve", "refresh")
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: i18n.T("status.short"),
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
			// Security summary from the cached CVE scan, so the --json output
			// carries the same security info the human view shows (parity).
			type securityJSON struct {
				Scanned        bool `json:"scanned"`
				Vulnerable     int  `json:"vulnerable"`
				HighOrCritical int  `json:"high_or_critical"`
			}
			sec := securityJSON{}
			if s, ok := cve.Load(); ok {
				sec = securityJSON{Scanned: true, Vulnerable: s.Vulnerable, HighOrCritical: s.HighOrCritical}
			}
			return output.Emit(struct {
				ToolsInstalled int          `json:"tools_installed"`
				ToolsTotal     int          `json:"tools_total"`
				UpdatesPending int          `json:"updates_pending"`
				Security       securityJSON `json:"security"`
				ShellLayer     bool         `json:"shell_layer"`
				VersionManager string       `json:"version_manager,omitempty"`
				Backend        string       `json:"backend"`
				Theme          string       `json:"theme"`
				Profiles       []string     `json:"profiles"`
			}{installed, total, outdated, sec, shellOn, vm, backend, ui.Active.Name, names})
		}

		fmt.Println(ui.Header("opsforge", i18n.T("status.header.tag")))
		fmt.Println()

		// Toolbox line with a coverage bar.
		fmt.Printf("  %s %s  %s\n",
			ui.Label(i18n.T("status.label.toolbox"), 10),
			ui.Bar(installed, total, 20),
			ui.Dim.Render(i18n.T("status.toolbox.installed",
				i18n.V("n", strconv.Itoa(installed), "total", strconv.Itoa(total)))))

		// Updates.
		if outdated > 0 {
			fmt.Printf("  %s %s %s\n",
				ui.Label(i18n.T("status.label.updates"), 10),
				ui.Warn.Render(ui.MarkUpdate+" "+i18n.T("status.updates.available", i18n.V("n", strconv.Itoa(outdated)))),
				ui.Dim.Render(i18n.T("status.updates.hint")))
		} else if installed > 0 {
			fmt.Printf("  %s %s\n", ui.Label(i18n.T("status.label.updates"), 10),
				ui.OK.Render(ui.MarkOK+" "+i18n.T("status.updates.uptodate")))
		}

		// Security — from the cached CVE scan, so status never blocks on
		// the network. A stale (or missing) cache triggers a background
		// refresh for next time.
		if installed > 0 {
			printSecurityLine()
			printPostureLine()
		}

		// Shell environment.
		shellVal := ui.Dim.Render(i18n.T("status.shell.off"))
		if shellOn {
			shellVal = ui.OK.Render(ui.MarkOK + " " + i18n.T("status.shell.active"))
		}
		fmt.Printf("  %s %s\n", ui.Label(i18n.T("status.label.shell"), 10), shellVal)

		// Version manager.
		vmLine := ui.Dim.Render(i18n.T("status.versions.none"))
		if vm != "" {
			vmLine = ui.OK.Render(ui.MarkOK + " " + vm)
		}
		fmt.Printf("  %s %s\n", ui.Label(i18n.T("status.label.versions"), 10), vmLine)

		// Backend + theme footer.
		backendLine := i18n.T("status.backend.github")
		if installer.BrewAvailable() {
			backendLine = i18n.T("status.backend.brew")
		}
		fmt.Printf("  %s %s\n", ui.Label(i18n.T("status.label.backend"), 10), ui.Dim.Render(backendLine))
		theme := ui.Accent.Render(ui.Active.Name)
		switch {
		case os.Getenv("OPSFORGE_THEME") != "":
			theme += ui.Dim.Render(i18n.T("status.theme.fromenv"))
		case !ui.ThemePersisted():
			theme += ui.Dim.Render(i18n.T("status.theme.auto"))
		}
		fmt.Printf("  %s %s\n", ui.Label(i18n.T("status.label.theme"), 10), theme)

		if len(userps) > 0 {
			names := make([]string, 0, len(userps))
			for _, p := range userps {
				names = append(names, p.Name)
			}
			fmt.Printf("  %s %s\n", ui.Label(i18n.T("status.label.profiles"), 10),
				ui.Dim.Render(strings.Join(names, ", ")))
		}

		fmt.Println()
		fmt.Println(ui.Dim.Render(i18n.T("status.footer")))
		// One discreet pointer to a non-obvious feature, so it gets found.
		fmt.Println(ui.Faint.Render(i18n.T("status.tip")))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
