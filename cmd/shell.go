package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/shellcfg"
)

// lookPath is a thin alias so doctor reads cleanly.
var lookPath = exec.LookPath

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Manage the opsforge DevOps zsh environment",
	Long: `The opsforge shell layer turns your zsh into a DevOps-aware environment:

  - context prompt: kube cluster:namespace (red on prod), cloud account,
    terraform workspace — each shown only when relevant
  - guards: a confirmation before destructive commands on a prod cluster
  - aliases & helpers: k, tf, dc, kx/kn (context/namespace switch)
  - integrations: fzf, zoxide and atuin wired up when installed
  - completions: cached zsh completions for every installed tool

Enable it once with 'opsforge shell install', then 'exec zsh'.`,
}

var shellEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Print the zsh snippet to eval from ~/.zshrc",
	Long:  "Meant to be used as: eval \"$(opsforge shell env)\"",
	RunE: func(cmd *cobra.Command, args []string) error {
		out, err := shellcfg.Env()
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
	Short: "Install the opsforge shell environment into ~/.zshrc (idempotent)",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := shellcfg.InstallToZshrc()
		if err != nil {
			return err
		}
		if cat, err := catalog.Load(); err == nil {
			shellcfg.Sync(cat.Tools())
		}
		fmt.Printf("opsforge shell environment installed in %s\n", path)
		fmt.Println("Run `exec zsh` (or open a new terminal) to activate it.")
		return nil
	},
}

var shellUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the opsforge shell environment from ~/.zshrc",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !shellcfg.InstalledInZshrc() {
			fmt.Println("opsforge shell environment is not installed.")
			return nil
		}
		path, err := shellcfg.UninstallFromZshrc()
		if err != nil {
			return err
		}
		fmt.Printf("Removed the opsforge block from %s and deleted its modules.\n", path)
		fmt.Println("Run `exec zsh` to apply.")
		return nil
	},
}

var (
	docOKGreen = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	docDimGrey = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

var shellDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Show what the opsforge shell environment provides and its state",
	RunE: func(cmd *cobra.Command, args []string) error {
		mark := func(ok bool) string {
			if ok {
				return docOKGreen.Render("✓")
			}
			return docDimGrey.Render("·")
		}

		fmt.Printf("%s installed in ~/.zshrc\n", mark(shellcfg.InstalledInZshrc()))

		cfgDir, _ := shellcfg.ConfigDir()
		mods, _ := shellcfg.Modules()
		for _, m := range mods {
			_, err := os.Stat(filepath.Join(cfgDir, m.Name+".zsh"))
			fmt.Printf("  %s module %s\n", mark(err == nil), m.Name)
		}

		complDir, _ := shellcfg.CompletionsDir()
		entries, _ := os.ReadDir(complDir)
		fmt.Printf("%s %d cached tool completion(s)\n", mark(len(entries) > 0), len(entries))

		fmt.Println("\nintegrations detected on PATH:")
		for _, tool := range []string{"fzf", "zoxide", "atuin", "eza", "bat", "kubectl"} {
			_, err := lookPath(tool)
			fmt.Printf("  %s %s\n", mark(err == nil), tool)
		}
		return nil
	},
}

func init() {
	shellCmd.AddCommand(shellEnvCmd, shellSyncCmd, shellInstallCmd, shellUninstallCmd, shellDoctorCmd)
	rootCmd.AddCommand(shellCmd)
}
