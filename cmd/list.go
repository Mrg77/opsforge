package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/ui"
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
			continue
		}
		fmt.Println(ui.Section(c.Name))
		for _, r := range rows {
			fmt.Println(r)
		}
	}

	printListFooter(filter, shown, installed, outdated, len(cat.Tools()))
	return nil
}

func formatRow(t catalog.Tool, s detect.Status) string {
	mark, note := ui.MissMark(), ui.Dim.Render(t.Description)
	switch {
	case s.Outdated:
		mark = ui.Warn.Render(ui.MarkOK)
		label := "update available"
		if s.Version != "" {
			label = s.Version + "  " + ui.MarkMissing + " update available"
		}
		note = ui.Warn.Render(label)
	case s.Installed:
		mark = ui.OKMark()
		if s.Version != "" {
			note = ui.Dim.Render(s.Version)
		}
	}
	return fmt.Sprintf("  %s %-16s %s", mark, t.Name, note)
}

func printListFooter(filter listFilter, shown, installed, outdated, total int) {
	fmt.Println()
	switch filter {
	case filterInstalled:
		if shown == 0 {
			fmt.Println(ui.Dim.Render("No catalog tools installed yet. Run `opsforge` to pick some, or `opsforge list all`."))
			return
		}
		fmt.Printf("%s   %s\n",
			ui.Dim.Render(fmt.Sprintf("%d installed", installed)),
			ui.Dim.Render("`opsforge list all` shows the full catalog"))
	case filterOutdated:
		if shown == 0 {
			fmt.Println(ui.OK.Render("Everything installed is up to date."))
			return
		}
		fmt.Printf("%s   %s\n",
			ui.Warn.Render(fmt.Sprintf("%d update(s) available", outdated)),
			ui.Dim.Render("run `opsforge upgrade` or select them in the picker"))
	default:
		fmt.Printf("%s   %s   %s\n",
			ui.OK.Render(fmt.Sprintf("%s %d installed", ui.MarkOK, installed)),
			ui.Warn.Render(fmt.Sprintf("%s %d to update", ui.MarkOK, outdated)),
			ui.Dim.Render(fmt.Sprintf("%s %d total", ui.MarkMissing, total)))
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
