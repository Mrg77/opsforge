package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
)

var (
	listCategory = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	listOK       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	listDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List catalog tools and their installation status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		statuses := detect.All(cat.Tools())
		for _, c := range cat.Categories {
			fmt.Println(listCategory.Render(c.Name))
			for _, t := range c.Tools {
				s := statuses[t.Name]
				mark, note := listDim.Render("·"), t.Description
				if s.Installed {
					mark = listOK.Render("✓")
					if s.Version != "" {
						note = s.Version
					}
				}
				fmt.Printf("  %s %-16s %s\n", mark, t.Name, listDim.Render(note))
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
