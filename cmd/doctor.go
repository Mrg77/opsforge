package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/installer"
	"github.com/Mrg77/opsforge/internal/shellcfg"
)

var (
	docOK   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	docWarn = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	docErr  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the health of your DevOps environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		tools := cat.Tools()
		fail := 0

		check := func(ok bool, label, fix string) {
			if ok {
				fmt.Printf("%s %s\n", docOK.Render("✓"), label)
				return
			}
			fail++
			fmt.Printf("%s %s\n", docErr.Render("✗"), label)
			if fix != "" {
				fmt.Printf("    → %s\n", fix)
			}
		}

		check(installer.Available(), "Homebrew available",
			"install it from https://brew.sh")
		check(strings.Contains(os.Getenv("PATH"), "/opt/homebrew/bin") ||
			strings.Contains(os.Getenv("PATH"), "/usr/local/bin"),
			"Homebrew bin directory in PATH",
			"add `eval \"$(/opt/homebrew/bin/brew shellenv)\"` to your ~/.zprofile")
		check(shellcfg.InstalledInZshrc(), "opsforge shell layer in ~/.zshrc",
			"run `opsforge shell install`")

		dir, err := shellcfg.CompletionsDir()
		if err == nil {
			entries, _ := os.ReadDir(dir)
			check(len(entries) > 0, fmt.Sprintf("cached zsh completions (%d)", len(entries)),
				"run `opsforge shell sync`")
		}

		installed, broken := 0, []string{}
		for _, t := range tools {
			s := detect.Tool(t)
			if !s.Installed {
				continue
			}
			installed++
			if s.Version == "" {
				broken = append(broken, t.Name)
			}
		}
		fmt.Printf("%s %d/%d catalog tools installed\n", docOK.Render("✓"), installed, len(tools))
		if len(broken) > 0 {
			fmt.Printf("%s version check failed for: %s\n",
				docWarn.Render("⚠"), strings.Join(broken, ", "))
		}

		if fail > 0 {
			return fmt.Errorf("%d check(s) failed", fail)
		}
		fmt.Println("\nAll good. Happy shipping!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
