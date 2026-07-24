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
	Short: "Print the shell snippet to load from your rc file",
	Long: `Meant to be used from your shell rc:
  zsh:  eval "$(opsforge shell env)"
  fish: opsforge shell env --shell fish | source`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sh, err := resolveShell()
		if err != nil {
			return err
		}
		out, err := sh.EnvFor()
		if err != nil {
			return err
		}
		fmt.Print(out)
		return nil
	},
}

var shellSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Refresh the shell modules and cached completions (run after upgrading opsforge)",
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
	Short: "Install the opsforge shell environment into your rc file (idempotent)",
	RunE: func(cmd *cobra.Command, args []string) error {
		sh, err := resolveShell()
		if err != nil {
			return err
		}
		path, err := sh.InstallTo()
		if err != nil {
			return err
		}

		if sh == shellcfg.Zsh {
			if cat, err := catalog.Load(); err == nil {
				shellcfg.Sync(cat.Tools())
			}
			// zsh needs plugins for inline suggestions / menu / highlighting;
			// fish ships all of that natively, so this step is zsh-only.
			if !shellNoPlugins {
				fmt.Println("Setting up the interactive experience (this may take a minute)…")
				installed, failed := shellcfg.EnsureInteractivePlugins()
				for _, p := range installed {
					fmt.Printf("  %s installed %s\n", docOKGreen.Render("✓"), p)
				}
				if len(failed) > 0 {
					fmt.Printf("  %s could not install: %v (features degrade gracefully)\n",
						docDimGrey.Render("·"), failed)
				}
			}
		}

		fmt.Printf("\nopsforge shell environment installed in %s\n", path)
		if sh == shellcfg.Fish {
			fmt.Println("Run `exec fish` (or open a new terminal) to activate it.")
			fmt.Println("fish already gives you inline autosuggestions, syntax highlighting")
			fmt.Println("and prefix history search — opsforge adds the guards, the prod-aware")
			fmt.Println("prompt, the `?` help and the DevOps aliases on top.")
		} else {
			fmt.Println("Run `exec zsh` (or open a new terminal) to activate it — then just")
			fmt.Println("start typing: a gray suggestion appears inline (→ to accept), ↑ walks")
			fmt.Println("your history by what you've typed, Tab completes, and the line is")
			fmt.Println("colored as you go. (Prefer an always-on menu? OPSFORGE_AUTOMENU=1.)")
		}
		return nil
	},
}

var shellNoPlugins bool

var shellUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the opsforge shell environment from your rc file",
	RunE: func(cmd *cobra.Command, args []string) error {
		sh, err := resolveShell()
		if err != nil {
			return err
		}
		if !sh.InstalledIn() {
			fmt.Println("opsforge shell environment is not installed.")
			return nil
		}
		path, err := sh.UninstallFrom()
		if err != nil {
			return err
		}
		fmt.Printf("Removed the opsforge block from %s and deleted its modules.\n", path)
		fmt.Printf("Run `exec %s` to apply.\n", sh)
		return nil
	},
}

// shellFlag holds the --shell value; empty means auto-detect from $SHELL.
var shellFlag string

// resolveShell turns the --shell flag (or $SHELL auto-detection) into a Shell.
func resolveShell() (shellcfg.Shell, error) {
	if shellFlag == "" {
		return shellcfg.DetectShell(), nil
	}
	return shellcfg.ParseShell(shellFlag)
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

		sh, err := resolveShell()
		if err != nil {
			return err
		}
		rc, _ := sh.RcPath()
		fmt.Printf("shell: %s\n", sh)
		fmt.Printf("%s installed in %s\n", mark(sh.InstalledIn()), rc)

		cfgDir, _ := sh.ModuleDir()
		for _, name := range sh.ModuleNames() {
			_, err := os.Stat(filepath.Join(cfgDir, name+sh.Ext()))
			fmt.Printf("  %s module %s\n", mark(err == nil), name)
		}

		// Completions cache and the plugin layer are zsh-specific (fish ships
		// autosuggestions/highlighting/completions natively).
		if sh == shellcfg.Zsh {
			complDir, _ := shellcfg.CompletionsDir()
			entries, _ := os.ReadDir(complDir)
			fmt.Printf("%s %d cached tool completion(s)\n", mark(len(entries) > 0), len(entries))

			fmt.Println("\ninteractive experience (inline suggestions, auto menu, highlighting):")
			for _, p := range shellcfg.InteractivePluginStatus() {
				fmt.Printf("  %s %s\n", mark(p.Installed), p.Name)
			}
		} else {
			fmt.Println("\ninteractive experience: provided natively by fish")
		}

		fmt.Println("\nintegrations detected on PATH:")
		for _, tool := range []string{"fzf", "zoxide", "atuin", "eza", "bat", "kubectl"} {
			_, err := lookPath(tool)
			fmt.Printf("  %s %s\n", mark(err == nil), tool)
		}
		return nil
	},
}

func init() {
	shellInstallCmd.Flags().BoolVar(&shellNoPlugins, "no-plugins", false,
		"skip installing the interactive plugins (autosuggestions, menu, highlighting)")
	// --shell selects the target shell (zsh|fish); empty auto-detects $SHELL.
	for _, c := range []*cobra.Command{shellEnvCmd, shellInstallCmd, shellUninstallCmd, shellDoctorCmd} {
		c.Flags().StringVar(&shellFlag, "shell", "", "target shell: zsh or fish (default: auto-detect from $SHELL)")
	}
	shellCmd.AddCommand(shellEnvCmd, shellSyncCmd, shellInstallCmd, shellUninstallCmd, shellDoctorCmd)
	rootCmd.AddCommand(shellCmd)
}
