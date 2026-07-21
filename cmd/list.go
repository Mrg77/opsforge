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
	listOK       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // green: installed
	listUpdate   = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // orange: update available
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
		statuses := detect.AllWithOutdated(cat.Tools())
		for _, c := range cat.Categories {
			fmt.Println(listCategory.Render(c.Name))
			for _, t := range c.Tools {
				s := statuses[t.Name]
				mark, note := listDim.Render("·"), listDim.Render(t.Description)
				switch {
				case s.Outdated:
					mark = listUpdate.Render("✓")
					label := "update available"
					if s.Version != "" {
						label = s.Version + "  · update available"
					}
					note = listUpdate.Render(label)
				case s.Installed:
					mark = listOK.Render("✓")
					if s.Version != "" {
						note = listDim.Render(s.Version)
					}
				}
				fmt.Printf("  %s %-16s %s\n", mark, t.Name, note)
			}
		}
		fmt.Printf("\n%s   %s   %s\n",
			listOK.Render("✓ installed"),
			listUpdate.Render("✓ update available"),
			listDim.Render("· not installed"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
