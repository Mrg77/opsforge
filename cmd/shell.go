package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/shellcfg"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Manage the opsforge zsh layer (completions, aliases, prompt)",
}

var shellEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Print the zsh snippet to eval from ~/.zshrc",
	Long:  "Meant to be used as: eval \"$(opsforge shell env)\"",
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		out, err := shellcfg.Env(cat.Tools())
		if err != nil {
			return err
		}
		fmt.Print(out)
		return nil
	},
}

var shellSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Regenerate cached zsh completions for installed tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		synced, err := shellcfg.Sync(cat.Tools())
		if err != nil {
			return err
		}
		if len(synced) == 0 {
			fmt.Println("No completions to sync (no installed tool exposes zsh completions).")
			return nil
		}
		fmt.Printf("Synced completions: %s\n", strings.Join(synced, ", "))
		fmt.Println("Restart your shell (or `exec zsh`) to pick them up.")
		return nil
	},
}

var shellInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Add the opsforge layer to ~/.zshrc (idempotent)",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := shellcfg.InstallToZshrc()
		if err != nil {
			return err
		}
		fmt.Printf("opsforge layer installed in %s\n", path)
		fmt.Println("Open a new terminal (or `exec zsh`) to activate it.")
		return nil
	},
}

func init() {
	shellCmd.AddCommand(shellEnvCmd, shellSyncCmd, shellInstallCmd)
	rootCmd.AddCommand(shellCmd)
}
