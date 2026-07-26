package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/installer"
)

// version, commit and date are injected at build time by GoReleaser via
// ldflags. They default to a "dev" marker for source builds so the version
// command can honestly say the binary wasn't cut from a release.
var (
	version = "dev"
	commit  = ""
	date    = ""
)

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
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		return runPicker(cat)
	},
}

// Execute runs the root command. It is the single entry point used by main.
func Execute() {
	ensureBinDirOnPath()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ensureBinDirOnPath prepends opsforge's install dir (~/.local/bin) to the
// process PATH so detection sees tools opsforge itself installed there, even
// when the user's shell hasn't been reloaded to pick up the new PATH. Without
// this, `sync`/`install` could install a tool via a GitHub release and then
// fail to detect it (e.g. an empty opsforge.lock right after install).
func ensureBinDirOnPath() {
	dir := installer.BinDir()
	if dir == "" {
		return
	}
	path := os.Getenv("PATH")
	for _, p := range strings.Split(path, string(os.PathListSeparator)) {
		if p == dir {
			return // already present
		}
	}
	os.Setenv("PATH", dir+string(os.PathListSeparator)+path)
}
