package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
)

var (
	profileName = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	profileDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	profileMeta = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	toolOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // green: installed
	toolUpdate  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // orange: update available
	toolMissing = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // grey: not installed
	barFilled   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	barEmpty    = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)

// toolCol is the fixed column width for a tool cell (marker + name),
// picked to fit the longest catalog tool name with breathing room.
const toolCol = 18

// progressBar renders a filled/empty bar, e.g. ███████░░░ for 7/10.
func progressBar(done, total, width int) string {
	if total == 0 {
		return barEmpty.Render(strings.Repeat("░", width))
	}
	filled := done * width / total
	return barFilled.Render(strings.Repeat("█", filled)) +
		barEmpty.Render(strings.Repeat("░", width-filled))
}

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "List stack profiles usable with `opsforge install --profile`",
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		statuses := detect.AllWithOutdated(cat.Tools())

		fmt.Println()
		for _, p := range cat.Profiles {
			installed := 0
			for _, name := range p.Tools {
				if statuses[name].Installed {
					installed++
				}
			}

			// Header: name, progress bar, count, description.
			fmt.Printf("  %s  %s  %s\n",
				profileName.Render(fmt.Sprintf("%-14s", p.Name)),
				progressBar(installed, len(p.Tools), 12),
				profileMeta.Render(fmt.Sprintf("%d/%d", installed, len(p.Tools))))
			fmt.Printf("  %s\n", profileDesc.Render(p.Description))

			// Tools in an aligned grid: 4 fixed-width columns.
			const cols = 4
			for i, name := range p.Tools {
				fmt.Print("  " + renderCell(name, statuses[name]))
				if (i+1)%cols == 0 || i == len(p.Tools)-1 {
					fmt.Println()
				}
			}
			fmt.Println()
		}

		fmt.Printf("  %s   %s   %s\n\n",
			toolOK.Render("● installed"),
			toolUpdate.Render("● update available"),
			toolMissing.Render("● not installed"))
		return nil
	},
}

// renderCell formats one tool as a fixed-width, colored, state-marked cell.
func renderCell(name string, s detect.Status) string {
	var marker, style = "○", toolMissing
	switch {
	case s.Outdated:
		marker, style = "↑", toolUpdate
	case s.Installed:
		marker, style = "●", toolOK
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
