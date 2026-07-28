package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/ai"
	"github.com/Mrg77/opsforge/internal/i18n"
	"github.com/Mrg77/opsforge/internal/ui"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: i18n.T("ai.short"),
	Long: `opsforge's AI features (explain, advise) drive an AI CLI you already have,
so there's no API key to manage. This shows the detected backend and, if none
is found, the quickest way to get a free one.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ui.Header("opsforge ai", i18n.T("ai.header.tag")))
		fmt.Println()

		b := ai.Detect()
		if b == nil {
			fmt.Printf("  %s %s\n\n", ui.WarnMark(), ui.Warn.Render(i18n.T("ai.none")))
			fmt.Println(ui.Dim.Render(ai.SetupHint()))
			return nil
		}

		fmt.Printf("  %s %s\n", ui.OKMark(), ui.Heading.Render(i18n.T("ai.backend"))+b.Name)
		fmt.Println(ui.Dim.Render(i18n.T("ai.backend.note")))
		fmt.Println()
		fmt.Println("  " + ui.Faint.Render(i18n.T("ai.try")))
		fmt.Println("    " + ui.Dim.Render("opsforge explain \"kubectl drain node-1\""))
		fmt.Println("    " + ui.Dim.Render(i18n.T("ai.try.advise")))
		fmt.Println()
		fmt.Println("  " + ui.Faint.Render(i18n.T("ai.override")))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(aiCmd)
}
