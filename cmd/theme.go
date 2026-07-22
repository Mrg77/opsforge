package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/ui"
)

var themeCmd = &cobra.Command{
	Use:   "theme [name]",
	Short: "List color themes, or preview one",
	Long: `opsforge's visual identity is themeable. Set OPSFORGE_THEME (in your
~/.zshrc) to any of the listed names, or "auto" to match your terminal
background.

  opsforge theme            # list all themes with a color preview
  opsforge theme dracula    # preview a specific theme`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			ui.SetTheme(args[0])
			if ui.Active.Name != args[0] && args[0] != "auto" {
				fmt.Printf("%s unknown theme %q — showing the default. Available: %v\n",
					ui.WarnMark(), args[0], ui.ThemeNames())
			}
			previewTheme(ui.Active.Name)
			fmt.Printf("\nUse it: %s\n", ui.Accent.Render(fmt.Sprintf("export OPSFORGE_THEME=%s", ui.Active.Name)))
			return nil
		}

		fmt.Println(ui.Header("opsforge themes", "set OPSFORGE_THEME=<name> in your ~/.zshrc (or 'auto')"))
		fmt.Println()
		current := ui.Active.Name
		for _, name := range ui.ThemeNames() {
			ui.SetTheme(name)
			marker := "  "
			if name == current {
				marker = ui.OK.Render(ui.MarkOK + " ")
			}
			fmt.Printf("%s%s  %s\n", marker, ui.Title.Render(fmt.Sprintf("%-9s", name)), swatches())
		}
		ui.SetTheme(current) // restore
		return nil
	},
}

// swatches renders one colored block per palette role, so a theme's look
// is visible at a glance.
func swatches() string {
	blocks := []string{
		ui.Title.Render("██"),
		ui.Heading.Render("██"),
		ui.OK.Render("██"),
		ui.Warn.Render("██"),
		ui.Err.Render("██"),
		ui.Selected.Render("██"),
		ui.Accent.Render("██"),
		ui.Dim.Render("██"),
	}
	out := ""
	for _, b := range blocks {
		out += b
	}
	return out
}

// previewTheme shows a small mock of real opsforge output in the theme.
func previewTheme(name string) {
	fmt.Println(ui.Header("opsforge — "+name, "preview of this theme"))
	fmt.Println()
	fmt.Println(ui.Section("Toolbox"))
	fmt.Printf("  %s %-14s %s\n", ui.OKMark(), "kubectl", ui.Dim.Render("v1.33.0"))
	fmt.Printf("  %s %-14s %s\n", ui.Warn.Render(ui.MarkOK), "terraform",
		ui.Warn.Render("v1.5.7  · update available"))
	fmt.Printf("  %s %-14s %s\n", ui.MissMark(), "k9s", ui.Dim.Render("Terminal UI for Kubernetes"))
	fmt.Printf("  %s %-14s %s\n", ui.Selected.Render(ui.MarkSel), "helm", ui.Selected.Render("selected"))
	fmt.Println("  " + ui.Bar(3, 5, 20))
}

func init() {
	rootCmd.AddCommand(themeCmd)
}
