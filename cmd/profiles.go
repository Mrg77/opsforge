package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
)

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "List stack profiles usable with `opsforge install --profile`",
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		statuses := detect.All(cat.Tools())
		bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		for _, p := range cat.Profiles {
			installed := 0
			for _, name := range p.Tools {
				if statuses[name].Installed {
					installed++
				}
			}
			fmt.Printf("%s %s (%d/%d installed)\n", bold.Render(p.Name),
				dim.Render("— "+p.Description), installed, len(p.Tools))
			fmt.Printf("  %s\n", strings.Join(p.Tools, ", "))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(profilesCmd)
}
