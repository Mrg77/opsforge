package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/installer"
	"github.com/Mrg77/opsforge/internal/shellcfg"
	"github.com/Mrg77/opsforge/internal/ui"
	"github.com/Mrg77/opsforge/internal/userprofiles"
	"github.com/Mrg77/opsforge/internal/versions"
)

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

		// Shell environment.
		shellVal := ui.Dim.Render("off — `opsforge shell install`")
		if shellOn {
			shellVal = ui.OK.Render(ui.MarkOK + " active")
		}
		fmt.Printf("  %s %s\n", ui.Label("Shell", 10), shellVal)

		// Version manager.
		vm := ui.Dim.Render("none — install mise for `opsforge use`")
		if mgr := versions.Detect(); mgr != versions.None {
			vm = ui.OK.Render(ui.MarkOK + " " + string(mgr))
		}
		fmt.Printf("  %s %s\n", ui.Label("Versions", 10), vm)

		// Backend + theme footer.
		backend := "GitHub releases"
		if installer.BrewAvailable() {
			backend = "Homebrew + GitHub"
		}
		fmt.Printf("  %s %s\n", ui.Label("Backend", 10), ui.Dim.Render(backend))
		theme := ui.Active.Name
		if os.Getenv("OPSFORGE_THEME") == "" {
			theme += ui.Dim.Render(" (auto)")
		}
		fmt.Printf("  %s %s\n", ui.Label("Theme", 10), ui.Accent.Render(theme))

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
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
