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

// listFilter decides which tools a list invocation shows.
type listFilter int

const (
	filterInstalled listFilter = iota // default: only what you have
	filterAll                         // the whole catalog
	filterOutdated                    // only tools with an update
)

var listOutdatedOnly bool

func runList(filter listFilter) error {
	cat, err := catalog.Load()
	if err != nil {
		return err
	}
	statuses := detect.AllWithOutdated(cat.Tools())

	shown, installed, outdated := 0, 0, 0
	for _, c := range cat.Categories {
		var rows []string
		for _, t := range c.Tools {
			s := statuses[t.Name]
			if s.Installed {
				installed++
			}
			if s.Outdated {
				outdated++
			}
			// Apply the filter.
			switch filter {
			case filterInstalled:
				if !s.Installed {
					continue
				}
			case filterOutdated:
				if !s.Outdated {
					continue
				}
			}
			rows = append(rows, formatRow(t, s))
			shown++
		}
		if len(rows) == 0 {
			continue // hide categories with nothing to show under this filter
		}
		fmt.Println(listCategory.Render(c.Name))
		for _, r := range rows {
			fmt.Println(r)
		}
	}

	printListFooter(filter, shown, installed, outdated, len(cat.Tools()))
	return nil
}

func formatRow(t catalog.Tool, s detect.Status) string {
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
	return fmt.Sprintf("  %s %-16s %s", mark, t.Name, note)
}

func printListFooter(filter listFilter, shown, installed, outdated, total int) {
	switch filter {
	case filterInstalled:
		if shown == 0 {
			fmt.Println(listDim.Render("No catalog tools installed yet. Run `opsforge` to pick some, or `opsforge list all`."))
			return
		}
		fmt.Printf("\n%s   %s\n",
			listDim.Render(fmt.Sprintf("%d installed", installed)),
			listDim.Render("`opsforge list all` shows the full catalog"))
	case filterOutdated:
		if shown == 0 {
			fmt.Println(listOK.Render("Everything installed is up to date."))
			return
		}
		fmt.Printf("\n%s   %s\n",
			listUpdate.Render(fmt.Sprintf("%d update(s) available", outdated)),
			listDim.Render("run `opsforge upgrade` or select them in the picker"))
	default: // filterAll
		fmt.Printf("\n%s   %s   %s\n",
			listOK.Render(fmt.Sprintf("✓ %d installed", installed)),
			listUpdate.Render(fmt.Sprintf("✓ %d to update", outdated)),
			listDim.Render(fmt.Sprintf("· %d total", total)))
	}
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List your installed catalog tools (use `list all` for the full catalog)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if listOutdatedOnly {
			return runList(filterOutdated)
		}
		return runList(filterInstalled)
	},
}

var listAllCmd = &cobra.Command{
	Use:   "all",
	Short: "List the entire catalog with installation status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(filterAll)
	},
}

func init() {
	listCmd.Flags().BoolVarP(&listOutdatedOnly, "outdated", "u", false,
		"show only installed tools with an update available")
	listCmd.AddCommand(listAllCmd)
	rootCmd.AddCommand(listCmd)
}
