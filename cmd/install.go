package cmd

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/installer"
	"github.com/Mrg77/opsforge/internal/shellcfg"
	"github.com/Mrg77/opsforge/internal/tui"
)

var installProfile string

var installCmd = &cobra.Command{
	Use:   "install [tool...]",
	Short: "Install tools: interactive picker, explicit names, or a profile",
	Example: `  opsforge install                      # interactive picker
  opsforge install kubectl helm k9s     # non-interactive, by name
  opsforge install --profile aws-k8s    # a whole stack at once`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		if len(args) > 0 || installProfile != "" {
			return installNonInteractive(cat, args)
		}
		return runPicker(cat)
	},
}

// runPicker opens the interactive TUI; it is also the default behavior
// when the binary is launched without any subcommand.
func runPicker(cat *catalog.Catalog) error {
	rescan := func() map[string]detect.Status {
		return detect.AllWithOutdated(cat.Tools())
	}
	model := tui.New(cat.Categories, rescan(), rescan)
	final, err := tea.NewProgram(model).Run()
	if err != nil {
		return err
	}
	if final.(tui.Model).DidWork() {
		return postInstall(cat)
	}
	return nil
}

// installNonInteractive resolves the requested tool names (explicit args
// plus profile members), skips what is already present, and installs the
// rest sequentially with plain line output — scriptable and CI-friendly.
func installNonInteractive(cat *catalog.Catalog, names []string) error {
	if installProfile != "" {
		p, ok := cat.Profile(installProfile)
		if !ok {
			var known []string
			for _, pr := range cat.Profiles {
				known = append(known, pr.Name)
			}
			return fmt.Errorf("unknown profile %q (available: %s)",
				installProfile, strings.Join(known, ", "))
		}
		names = append(names, p.Tools...)
	}

	var queue []catalog.Tool
	requested := map[string]bool{}
	for _, name := range names {
		if requested[name] {
			continue
		}
		requested[name] = true
		t, ok := cat.Tool(name)
		if !ok {
			return fmt.Errorf("unknown tool %q (see `opsforge list`)", name)
		}
		queue = append(queue, t)
	}

	installed := 0
	for _, t := range queue {
		if detect.Tool(t).Installed {
			fmt.Printf("✓ %-16s already installed\n", t.Name)
			continue
		}
		fmt.Printf("… installing %s (via %s)\n", t.Name, installer.BackendFor(t))
		if res := installer.Install(t); res.Err != nil {
			fmt.Printf("✗ %-16s %v\n%s\n", t.Name, res.Err, res.OutputTail)
			continue
		}
		fmt.Printf("✓ %-16s installed\n", t.Name)
		installed++
	}
	if installed > 0 {
		return postInstall(cat)
	}
	return nil
}

// postInstall refreshes shell completions after successful installs and
// nudges the user to enable the shell layer if they have not yet.
func postInstall(cat *catalog.Catalog) error {
	synced, err := shellcfg.Sync(cat.Tools())
	if err != nil {
		return err
	}
	fmt.Printf("Generated zsh completions for %d tool(s).\n", len(synced))
	if shellcfg.InstalledInZshrc() {
		fmt.Println("Run `exec zsh` (or open a new terminal) to load the new completions.")
	} else {
		fmt.Println()
		fmt.Println("To get tab-completion for every installed tool, enable the opsforge")
		fmt.Println("shell layer once:")
		fmt.Println("    opsforge shell install && exec zsh")
	}
	return nil
}

func init() {
	installCmd.Flags().StringVarP(&installProfile, "profile", "p", "",
		"install a stack profile (see `opsforge profiles`)")
	rootCmd.AddCommand(installCmd)
}
