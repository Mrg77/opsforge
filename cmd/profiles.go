package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/ui"
	"github.com/Mrg77/opsforge/internal/userprofiles"
)

// toolCol is the fixed column width for a tool cell (marker + name),
// picked to fit the longest catalog tool name with breathing room.
const toolCol = 18

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "List stack profiles usable with `opsforge install --profile`",
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		statuses := detect.AllWithOutdated(cat.Tools())
		userps, _ := userprofiles.Load()

		fmt.Println(ui.Header("opsforge profiles", "install a whole stack with `opsforge install --profile <name>`"))
		fmt.Println()

		fmt.Println(ui.Section("Built-in"))
		for _, p := range cat.Profiles {
			printProfile(p, statuses, false)
		}

		if len(userps) > 0 {
			fmt.Println(ui.Section("Yours"))
			for _, p := range userps {
				printProfile(p, statuses, true)
			}
		} else {
			fmt.Println(ui.Dim.Render("  Tip: in the picker, select tools and press s to save your own profile.\n"))
		}

		fmt.Printf("  %s   %s   %s\n\n",
			ui.OK.Render("● installed"),
			ui.Warn.Render("● update available"),
			ui.Dim.Render("● not installed"))
		return nil
	},
}

// printProfile renders one profile: header with progress bar, then its
// tools in an aligned 4-column grid. `own` marks user profiles.
func printProfile(p catalog.Profile, statuses map[string]detect.Status, own bool) {
	installed := 0
	for _, name := range p.Tools {
		if statuses[name].Installed {
			installed++
		}
	}
	desc := p.Description
	if own {
		desc = "your saved stack"
	}
	fmt.Printf("  %s  %s  %s\n",
		ui.Heading.Render(fmt.Sprintf("%-14s", p.Name)),
		ui.Bar(installed, len(p.Tools), 12),
		ui.Dim.Render(fmt.Sprintf("%d/%d", installed, len(p.Tools))))
	fmt.Printf("  %s\n", ui.Dim.Render(desc))

	const cols = 4
	for i, name := range p.Tools {
		fmt.Print("  " + renderCell(name, statuses[name]))
		if (i+1)%cols == 0 || i == len(p.Tools)-1 {
			fmt.Println()
		}
	}
	fmt.Println()
}

// renderCell formats one tool as a fixed-width, colored, state-marked cell.
func renderCell(name string, s detect.Status) string {
	marker, style := "○", ui.Dim
	switch {
	case s.Outdated:
		marker, style = "↑", ui.Warn
	case s.Installed:
		marker, style = "●", ui.OK
	}
	cell := fmt.Sprintf("%s %s", marker, name)
	// Pad to a fixed visible width before applying color (ANSI codes
	// don't count toward the width, so pad the plain text first).
	if pad := toolCol - len(cell); pad > 0 {
		cell += strings.Repeat(" ", pad)
	}
	return style.Render(cell)
}

func init() {
	rootCmd.AddCommand(profilesCmd)
}
