package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/installer"
	"github.com/Mrg77/opsforge/internal/shellcfg"
	"github.com/Mrg77/opsforge/internal/tui"
	"github.com/Mrg77/opsforge/internal/ui"
	"github.com/Mrg77/opsforge/internal/userprofiles"
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
	// Saving validates tool names against the catalog before persisting,
	// so a profile can never reference a tool that does not exist.
	saver := func(name string, tools []string) error {
		for _, tn := range tools {
			if _, ok := cat.Tool(tn); !ok {
				return fmt.Errorf("unknown tool %q", tn)
			}
		}
		return userprofiles.Save(catalog.Profile{
			Name:        name,
			Description: "user profile",
			Tools:       tools,
		})
	}
	model := tui.New(cat.Categories, rescan(), rescan).
		WithProfileSaver(saver).
		WithSecurityTargets(CollectOSVTargets(cat))
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
		p, ok := resolveProfile(cat, installProfile)
		if !ok {
			return fmt.Errorf("unknown profile %q (see `opsforge profiles`)", installProfile)
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

	installed, failed := 0, 0
	for _, t := range queue {
		if detect.Tool(t).Installed {
			fmt.Printf("✓ %-16s already installed\n", t.Name)
			continue
		}
		fmt.Printf("… installing %s (via %s)\n", t.Name, installer.BackendFor(t))
		res := installer.Install(t)
		if res.Err != nil {
			fmt.Printf("✗ %-16s %v\n%s\n", t.Name, res.Err, res.OutputTail)
			failed++
			continue
		}
		fmt.Printf("✓ %-16s installed\n", t.Name)
		if res.Warning != "" {
			fmt.Printf("  %s %s\n", ui.WarnMark(), ui.Dim.Render(res.Warning))
		}
		installed++
	}
	if installed > 0 {
		if err := postInstall(cat); err != nil {
			return err
		}
	}
	// Propagate a non-zero exit when any install failed, so scripts/CI can
	// detect it (individual errors were printed above).
	if failed > 0 {
		return fmt.Errorf("%d of %d tool(s) failed to install", failed, failed+installed)
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

// resolveProfile looks up a profile by name, checking the embedded
// catalog first, then the user's saved profiles.
func resolveProfile(cat *catalog.Catalog, name string) (catalog.Profile, bool) {
	if p, ok := cat.Profile(name); ok {
		return p, true
	}
	userps, err := userprofiles.Load()
	if err != nil {
		return catalog.Profile{}, false
	}
	for _, p := range userps {
		if p.Name == name {
			return p, true
		}
	}
	return catalog.Profile{}, false
}

func init() {
	installCmd.Flags().StringVarP(&installProfile, "profile", "p", "",
		"install a stack profile (see `opsforge profiles`)")
	rootCmd.AddCommand(installCmd)
}
