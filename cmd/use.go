package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/versions"
)

var useCmd = &cobra.Command{
	Use:   "use <tool>[@<version>]",
	Short: "Pin a specific tool version in this directory (via mise/asdf)",
	Long: `Pin a specific version of a tool for the current project — useful for
reproducing or debugging version-specific behavior (e.g. terraform@1.5).

opsforge does not manage versions itself: it delegates to mise (preferred)
or asdf, writing the project's .mise.toml / .tool-versions with the mature
tool you already trust. Install one first with 'opsforge install mise'.`,
	Example: `  opsforge use terraform@1.5    # pin terraform 1.5 here
  opsforge use node@20          # pin node 20 here`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := versions.Detect()
		if mgr == versions.None {
			return fmt.Errorf("no version manager found — run `opsforge install mise` (or asdf) first")
		}

		tool, version := versions.ParseSpec(args[0])

		// A soft catalog check: warn (don't block) if the tool is unknown
		// to opsforge, since mise/asdf support many runtimes we don't list.
		if cat, err := catalog.Load(); err == nil {
			if _, known := cat.Tool(tool); !known {
				fmt.Printf("note: %q is not in the opsforge catalog; delegating to %s anyway.\n", tool, mgr)
			}
		}

		fmt.Printf("Pinning %s via %s…\n", args[0], mgr)
		ran, err := versions.Use(mgr, tool, version)
		for _, c := range ran {
			fmt.Printf("  $ %s\n", c)
		}
		if err != nil {
			return err
		}
		fmt.Printf("\nPinned. %s will use this version in this directory.\n", tool)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}
