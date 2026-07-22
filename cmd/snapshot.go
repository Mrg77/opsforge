package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/shellcfg"
	"github.com/Mrg77/opsforge/internal/snapshot"
	"github.com/Mrg77/opsforge/internal/userprofiles"
)

var (
	snapOK  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	snapDim = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

var snapshotOut string

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Export your whole workstation setup to one shareable file",
	Long: `Capture everything opsforge manages — installed tools, your custom
profiles, and the shell environment state — into a single YAML file.

Commit it to your dotfiles repo (or a gist), then rebuild the exact same
workstation anywhere with:  opsforge apply <file-or-url>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		fmt.Println(snapDim.Render("Scanning your workstation…"))
		statuses := detect.All(cat.Tools())
		profiles, _ := userprofiles.Load()

		s := snapshot.Capture(cat, statuses, profiles, shellcfg.InstalledInZshrc(), time.Now())
		data, err := s.Marshal()
		if err != nil {
			return err
		}

		if snapshotOut == "-" {
			fmt.Print(string(data))
			return nil
		}
		if err := os.WriteFile(snapshotOut, data, 0o644); err != nil {
			return err
		}

		fmt.Printf("%s %s\n", snapOK.Render("✓"), snapshotOut)
		fmt.Printf("  %s\n", snapDim.Render(fmt.Sprintf(
			"%d tool(s) · %d profile(s) · shell environment: %v",
			len(s.Tools), len(s.Profiles), s.Shell.Enabled)))
		fmt.Println()
		fmt.Println("Rebuild this workstation anywhere:")
		fmt.Printf("  opsforge apply %s\n", snapshotOut)
		fmt.Println(snapDim.Render("  (commit it to your dotfiles repo and apply from its raw URL)"))
		return nil
	},
}

func init() {
	snapshotCmd.Flags().StringVarP(&snapshotOut, "output", "o", "opsforge-snapshot.yaml",
		"output file ('-' for stdout)")
	rootCmd.AddCommand(snapshotCmd)
}
