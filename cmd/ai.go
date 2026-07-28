package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/ai"
	"github.com/Mrg77/opsforge/internal/ui"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "Show which AI backend opsforge uses — or how to set one up",
	Long: `opsforge's AI features (explain, advise) drive an AI CLI you already have,
so there's no API key to manage. This shows the detected backend and, if none
is found, the quickest way to get a free one.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ui.Header("opsforge ai", "the AI backend opsforge drives"))
		fmt.Println()

		b := ai.Detect()
		if b == nil {
			fmt.Printf("  %s %s\n\n", ui.WarnMark(), ui.Warn.Render("No AI backend detected"))
			fmt.Println(ui.Dim.Render(ai.SetupHint()))
			return nil
		}

		fmt.Printf("  %s %s\n", ui.OKMark(), ui.Heading.Render("Backend: ")+b.Name)
		fmt.Println(ui.Dim.Render("      opsforge feeds it a prompt and streams the answer — nothing is executed."))
		fmt.Println()
		fmt.Println("  " + ui.Faint.Render("Try:"))
		fmt.Println("    " + ui.Dim.Render("opsforge explain \"kubectl drain node-1\""))
		fmt.Println("    " + ui.Dim.Render("opsforge advise    # AI-prioritized take on your CVEs, updates & secrets"))
		fmt.Println()
		fmt.Println("  " + ui.Faint.Render("Override the backend any time with $OPSFORGE_AI_CMD."))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(aiCmd)
}
