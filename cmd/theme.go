package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/ui"
)

var themeCmd = &cobra.Command{
	Use:   "theme [name]",
	Short: "List color themes, or preview one",
	Long: `opsforge's visual identity is themeable. The simplest way to switch is
'opsforge theme set <name>', which persists your choice (no reload needed). For
a one-off or per-session override, set OPSFORGE_THEME=<name>. Use "auto" to
match your terminal background. Precedence: $OPSFORGE_THEME › saved choice › auto.

  opsforge theme            # list all themes with a color preview
  opsforge theme dracula    # preview a specific theme (doesn't persist)
  opsforge theme set nord   # persist it — every command follows`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			ui.SetTheme(args[0])
			if ui.Active.Name != args[0] && args[0] != "auto" {
				fmt.Printf("%s unknown theme %q — showing the default. Available: %v\n",
					ui.WarnMark(), args[0], ui.ThemeNames())
			}
			previewTheme(ui.Active.Name)
			fmt.Printf("\nKeep it: %s %s\n",
				ui.Accent.Render(fmt.Sprintf("opsforge theme set %s", ui.Active.Name)),
				ui.Dim.Render(fmt.Sprintf("(persists · or OPSFORGE_THEME=%s for one session)", ui.Active.Name)))
			return nil
		}

		fmt.Println(ui.Header("opsforge themes", "your active theme is shown with a ✓"))
		fmt.Println()
		current := ui.Active.Name
		for _, name := range ui.ThemeNames() {
			ui.SetTheme(name)
			marker := "  "
			label := name
			if name == current {
				marker = ui.OK.Render(ui.MarkOK + " ")
				label = name + " (active)"
			}
			// Pad the plain label, then color — keeps the swatches aligned.
			padded := label
			if p := 18 - len([]rune(label)); p > 0 {
				padded += fmt.Sprintf("%*s", p, "")
			}
			fmt.Printf("%s%s%s\n", marker, ui.Title.Render(padded), swatches())
		}
		ui.SetTheme(current) // restore

		fmt.Println()
		fmt.Println(ui.Dim.Render("  Preview:  ") + ui.Accent.Render("opsforge theme dracula"))
		fmt.Println(ui.Dim.Render("  Keep:     ") + ui.Accent.Render("opsforge theme set dracula") +
			ui.Dim.Render("  (persists, no reload) · or OPSFORGE_THEME=dracula for one session"))
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

var themeSetCmd = &cobra.Command{
	Use:   "set <name>",
	Short: "Persist a theme — used by every opsforge command, no reload needed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ui.SaveTheme(args[0]); err != nil {
			return err
		}
		fmt.Printf("%s theme set to %s\n", ui.OKMark(), ui.Accent.Render(args[0]))
		previewTheme(ui.Active.Name)
		fmt.Println(ui.Dim.Render("\n  Applied everywhere. Reload your shell (`exec zsh`) for the prompt to follow."))
		return nil
	},
}

func init() {
	themeCmd.AddCommand(themeSetCmd)
	rootCmd.AddCommand(themeCmd)
}
