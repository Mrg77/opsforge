package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/ui"
)

// listItem is the machine-readable shape of one tool in `list --json`.
type listItem struct {
	Name      string `json:"name"`
	Category  string `json:"category"`
	Installed bool   `json:"installed"`
	Outdated  bool   `json:"outdated"`
	Version   string `json:"version,omitempty"`
}

// listFilter decides which tools a list invocation shows.
type listFilter int

const (
	filterInstalled listFilter = iota // default: only what you have
	filterAll                         // the whole catalog
	filterOutdated                    // only tools with an update
)

var (
	listOutdatedOnly bool
	listSearch       string
)

// matchesSearch reports whether a tool matches the search term (empty term
// matches everything). It looks in the tool name, its description and its
// category — case-insensitively — so `list all -s dns` finds doggo, dnsx…
func matchesSearch(t catalog.Tool, category, term string) bool {
	if term == "" {
		return true
	}
	term = strings.ToLower(term)
	return strings.Contains(strings.ToLower(t.Name), term) ||
		strings.Contains(strings.ToLower(t.Description), term) ||
		strings.Contains(strings.ToLower(category), term)
}

func runList(filter listFilter) error {
	cat, err := catalog.Load()
	if err != nil {
		return err
	}
	statuses := detect.AllWithOutdated(cat.Tools())

	if output.JSON {
		return listJSON(cat, statuses, filter)
	}

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
			if !matchesSearch(t, c.Name, listSearch) {
				continue
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

// listJSON emits the (filtered) catalog as a JSON array for scripting/CI.
func listJSON(cat *catalog.Catalog, statuses map[string]detect.Status, filter listFilter) error {
	items := []listItem{}
	for _, c := range cat.Categories {
		for _, t := range c.Tools {
			s := statuses[t.Name]
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
			if !matchesSearch(t, c.Name, listSearch) {
				continue
			}
			items = append(items, listItem{
				Name:      t.Name,
				Category:  c.Name,
				Installed: s.Installed,
				Outdated:  s.Outdated,
				Version:   s.Version,
			})
		}
	}
	return output.Emit(items)
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
	if listSearch != "" {
		if shown == 0 {
			fmt.Printf("%s\n", ui.Dim.Render(fmt.Sprintf("No tool matches %q.", listSearch)))
			return
		}
		fmt.Printf("%s   %s\n",
			ui.Dim.Render(fmt.Sprintf("%d match(es) for %q", shown, listSearch)),
			ui.Dim.Render("of "+fmt.Sprintf("%d tools", total)))
		return
	}
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
	Use:   "list [search]",
	Short: "List installed tools — or search the whole catalog (list <term>)",
	Long: `With no arguments, lists the catalog tools you have installed.

  opsforge list              # what you have
  opsforge list all          # the entire catalog (249 tools)
  opsforge list -u           # only installed tools with an update
  opsforge list dns          # search the whole catalog by name/description
  opsforge list all -s kube  # same as 'list kube' (searches everything)

A search term (positional or -s) matches the tool name, description or
category, case-insensitively, and searches the FULL catalog.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			listSearch = args[0]
		}
		if listOutdatedOnly {
			return runList(filterOutdated)
		}
		// A search implies the whole catalog — otherwise you'd only find
		// among what you've already installed, which is rarely the intent.
		if listSearch != "" {
			return runList(filterAll)
		}
		return runList(filterInstalled)
	},
}

var listAllCmd = &cobra.Command{
	Use:   "all [search]",
	Short: "List the entire catalog with installation status",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			listSearch = args[0]
		}
		return runList(filterAll)
	},
}

func init() {
	listCmd.Flags().BoolVarP(&listOutdatedOnly, "outdated", "u", false,
		"show only installed tools with an update available")
	listCmd.PersistentFlags().StringVarP(&listSearch, "search", "s", "",
		"filter by name, description or category (searches the full catalog)")
	listCmd.AddCommand(listAllCmd)
	rootCmd.AddCommand(listCmd)
}
