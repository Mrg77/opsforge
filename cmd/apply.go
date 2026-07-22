package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/installer"
	"github.com/Mrg77/opsforge/internal/shellcfg"
	"github.com/Mrg77/opsforge/internal/snapshot"
	"github.com/Mrg77/opsforge/internal/userprofiles"
)

var (
	applyOK   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	applyNew  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	applyDim  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	applyWarn = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	applyErr  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	applyHead = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
)

var applyYes bool

var applyCmd = &cobra.Command{
	Use:   "apply <file-or-url>",
	Short: "Rebuild a workstation from a snapshot (file or URL)",
	Long: `Read an opsforge snapshot (created with 'opsforge snapshot') from a local
file or an http(s) URL and rebuild that workstation here: install the
missing tools, restore the custom profiles, and enable the shell
environment if it was enabled.

Shows the full plan and asks for confirmation before changing anything
(use --yes to skip in scripts).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		snap, err := snapshot.Load(args[0])
		if err != nil {
			return err
		}
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		fmt.Println(applyDim.Render("Scanning this machine…"))
		statuses := detect.All(cat.Tools())
		plan := snapshot.BuildPlan(snap, cat, statuses)

		// --- show the plan -------------------------------------------------
		fmt.Println()
		fmt.Println(applyHead.Render("Plan"))
		if len(plan.Install) > 0 {
			fmt.Printf("  %s %s\n", applyNew.Render(fmt.Sprintf("%d to install:", len(plan.Install))),
				strings.Join(plan.Install, ", "))
		}
		if len(plan.Present) > 0 {
			fmt.Printf("  %s\n", applyDim.Render(fmt.Sprintf("%d already installed", len(plan.Present))))
		}
		if len(plan.Unknown) > 0 {
			fmt.Printf("  %s %s\n", applyWarn.Render(fmt.Sprintf("%d unknown to this catalog:", len(plan.Unknown))),
				applyWarn.Render(strings.Join(plan.Unknown, ", ")))
		}
		if len(snap.Profiles) > 0 {
			fmt.Printf("  %s\n", applyDim.Render(fmt.Sprintf("%d profile(s) to restore", len(snap.Profiles))))
		}
		if snap.Shell.Enabled && !shellcfg.InstalledInZshrc() {
			fmt.Printf("  %s\n", applyDim.Render("shell environment will be enabled"))
		}
		if len(plan.Install) == 0 && len(snap.Profiles) == 0 &&
			(!snap.Shell.Enabled || shellcfg.InstalledInZshrc()) {
			fmt.Println()
			fmt.Println(applyOK.Render("Nothing to do — this machine already matches the snapshot."))
			return nil
		}

		// --- confirm -------------------------------------------------------
		if !applyYes {
			fmt.Printf("\nProceed? [y/N] ")
			line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			answer := strings.ToLower(strings.TrimSpace(line))
			if answer != "y" && answer != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		// --- execute -------------------------------------------------------
		fmt.Println()
		failed := 0
		for _, name := range plan.Install {
			t, _ := cat.Tool(name)
			fmt.Printf("… installing %s (via %s)\n", name, installer.BackendFor(t))
			if res := installer.Install(t); res.Err != nil {
				fmt.Printf("%s %-16s %v\n%s\n", applyErr.Render("✗"), name, res.Err, res.OutputTail)
				failed++
				continue
			}
			fmt.Printf("%s %-16s installed\n", applyOK.Render("✓"), name)
		}

		for _, p := range snap.Profiles {
			if err := userprofiles.Save(p); err != nil {
				fmt.Printf("%s profile %q: %v\n", applyErr.Render("✗"), p.Name, err)
				failed++
			} else {
				fmt.Printf("%s profile %-10s restored\n", applyOK.Render("✓"), p.Name)
			}
		}

		if snap.Shell.Enabled && !shellcfg.InstalledInZshrc() {
			if _, err := shellcfg.InstallToZshrc(); err != nil {
				fmt.Printf("%s shell environment: %v\n", applyErr.Render("✗"), err)
				failed++
			} else {
				shellcfg.Sync(cat.Tools())
				fmt.Printf("%s shell environment enabled (run `exec zsh`)\n", applyOK.Render("✓"))
			}
		}

		fmt.Println()
		if failed > 0 {
			return fmt.Errorf("workstation restored with %d failure(s)", failed)
		}
		fmt.Println(applyNew.Render("Workstation restored. Welcome home. 🔥"))
		return nil
	},
}

func init() {
	applyCmd.Flags().BoolVarP(&applyYes, "yes", "y", false, "apply without asking for confirmation")
	rootCmd.AddCommand(applyCmd)
}
