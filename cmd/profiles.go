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
	profileDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	toolOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // green: installed
	toolUpdate  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // orange: update available
	toolMissing = lipgloss.NewStyle().Foreground(lipgloss.Color("241")) // grey: not installed
)

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "List stack profiles usable with `opsforge install --profile`",
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		statuses := detect.AllWithOutdated(cat.Tools())
		for _, p := range cat.Profiles {
			installed := 0
			names := make([]string, 0, len(p.Tools))
			for _, name := range p.Tools {
				s := statuses[name]
				switch {
				case s.Outdated:
					installed++
					names = append(names, toolUpdate.Render(name+" ↑"))
				case s.Installed:
					installed++
					names = append(names, toolOK.Render(name))
				default:
					names = append(names, toolMissing.Render(name))
				}
			}
			fmt.Printf("%s %s (%d/%d installed)\n", profileName.Render(p.Name),
				profileDesc.Render("— "+p.Description), installed, len(p.Tools))
			fmt.Printf("  %s\n", strings.Join(names, "  "))
		}
		fmt.Printf("\n%s  %s  %s\n",
			toolOK.Render("● installed"),
			toolUpdate.Render("● update available"),
			toolMissing.Render("● not installed"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(profilesCmd)
}
