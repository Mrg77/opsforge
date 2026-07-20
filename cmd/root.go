package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/installer"
)

// version is injected at build time by GoReleaser via ldflags.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:           "opsforge",
	Short:         "Forge your DevOps workstation: pick your CLIs, get a fully wired shell",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `opsforge sets up a DevOps workstation in minutes.

Pick the CLIs you need (Kubernetes, IaC, cloud providers, containers...)
from an interactive terminal UI, install them in one go, and get a zsh
layer with auto-generated completions, aliases and a kube-aware prompt.

Run with no arguments to open the interactive picker.`,
	// Launching the bare binary in a terminal opens the picker — the
	// primary UX. Pipes and scripts still get the help text.
	RunE: func(cmd *cobra.Command, args []string) error {
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			return cmd.Help()
		}
		if !installer.Available() {
			return fmt.Errorf("homebrew is required (https://brew.sh) — binary downloads are on the roadmap")
		}
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		return runPicker(cat)
	},
}

// Execute runs the root command. It is the single entry point used by main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
