package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
layer with auto-generated completions, aliases and a kube-aware prompt.`,
}

// Execute runs the root command. It is the single entry point used by main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
