package cmd

import (
	"github.com/Mrg77/opsforge/internal/output"
)

// The --json flag is registered as a persistent flag on the root command
// so every subcommand can opt into machine-readable output. Commands that
// support it read output.JSON and branch to a JSON path; commands that
// don't simply ignore it. Kept in its own file so the flag wiring is easy
// to find and doesn't crowd root.go.
func init() {
	rootCmd.PersistentFlags().BoolVar(&output.JSON, "json", false,
		"output machine-readable JSON instead of the human UI (where supported)")
}
