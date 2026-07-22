package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/installer"
	"github.com/Mrg77/opsforge/internal/ui"
)

var (
	upOK  = ui.OK
	upNew = ui.OKBold // green: new version
	upOld = ui.Warn   // orange: old version
	upDim = ui.Dim
	upErr = ui.Err
)

var upgradeOutdatedOnly bool

var upgradeCmd = &cobra.Command{
	Use:     "upgrade [tool...]",
	Aliases: []string{"update"},
	Short:   "Upgrade installed tools — all, only outdated (-u), or named ones",
	Example: `  opsforge upgrade              # upgrade every installed tool
  opsforge upgrade -u           # upgrade only tools with an update available
  opsforge upgrade jq yq gh     # upgrade just these`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !installer.Available() {
			return fmt.Errorf("homebrew is required (https://brew.sh)")
		}
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		statuses := detect.AllWithOutdated(cat.Tools())

		// Resolve the target set from args / flags.
		targets, err := upgradeTargets(cat, statuses, args)
		if err != nil {
			return err
		}
		if len(targets) == 0 {
			if upgradeOutdatedOnly {
				fmt.Println("Everything installed is already up to date.")
			} else {
				fmt.Println("No installed catalog tools to upgrade.")
			}
			return nil
		}

		upgraded, unchanged, skipped, failed := 0, 0, 0, 0
		for _, t := range targets {
			before := audit.NormalizeVersion(statuses[t.Name].Version)
			switch res := installer.Upgrade(t); {
			case res.Err == nil:
				after := audit.NormalizeVersion(detect.Tool(t).Version)
				switch {
				case before != "" && after != "" && before != after:
					fmt.Printf("%s %-16s %s → %s\n", upOK.Render("✓"), t.Name,
						upOld.Render("v"+before), upNew.Render("v"+after))
					upgraded++
				default:
					fmt.Printf("%s %-16s %s\n", upOK.Render("✓"), t.Name,
						upDim.Render("already up to date"))
					unchanged++
				}
			case res.NotBrewManaged:
				fmt.Printf("%s %-16s %s\n", upDim.Render("·"), t.Name,
					upDim.Render("skipped (not installed via Homebrew)"))
				skipped++
			default:
				fmt.Printf("%s %-16s %v\n%s\n", upErr.Render("✗"), t.Name, res.Err, res.OutputTail)
				failed++
			}
		}
		fmt.Printf("\n%s upgraded, %d already current, %d skipped, %d failed\n",
			upNew.Render(fmt.Sprintf("%d", upgraded)), unchanged, skipped, failed)
		if failed > 0 {
			return fmt.Errorf("%d upgrade(s) failed", failed)
		}
		return nil
	},
}

// upgradeTargets builds the list of tools to upgrade from the requested
// names (or all installed), honoring the --outdated filter.
func upgradeTargets(cat *catalog.Catalog, statuses map[string]detect.Status, names []string) ([]catalog.Tool, error) {
	var targets []catalog.Tool

	if len(names) > 0 {
		// Explicit names: validate, require installed, warn if up to date.
		for _, name := range names {
			t, ok := cat.Tool(name)
			if !ok {
				return nil, fmt.Errorf("unknown tool %q (see `opsforge list all`)", name)
			}
			s := statuses[name]
			if !s.Installed {
				fmt.Printf("· %-16s not installed — skipping (use `opsforge install %s`)\n", name, name)
				continue
			}
			if upgradeOutdatedOnly && !s.Outdated {
				fmt.Printf("· %-16s already up to date\n", name)
				continue
			}
			targets = append(targets, t)
		}
		return targets, nil
	}

	// No names: all installed tools, filtered by --outdated when set.
	for _, t := range cat.Tools() {
		s := statuses[t.Name]
		if !s.Installed {
			continue
		}
		if upgradeOutdatedOnly && !s.Outdated {
			continue
		}
		targets = append(targets, t)
	}
	return targets, nil
}

func init() {
	upgradeCmd.Flags().BoolVarP(&upgradeOutdatedOnly, "outdated", "u", false,
		"upgrade only tools that have an update available")
	rootCmd.AddCommand(upgradeCmd)
}
