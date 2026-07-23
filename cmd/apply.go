package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/installer"
	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/shellcfg"
	"github.com/Mrg77/opsforge/internal/snapshot"
	"github.com/Mrg77/opsforge/internal/ui"
	"github.com/Mrg77/opsforge/internal/userprofiles"
	"github.com/Mrg77/opsforge/internal/versions"
)

// apply* aliases keep the command body readable while using the shared ui.
var (
	applyOK   = ui.OK
	applyNew  = ui.OKBold
	applyDim  = ui.Dim
	applyWarn = ui.Warn
	applyErr  = ui.Err
	applyHead = ui.Heading
)

var (
	applyYes   bool
	applyCheck bool
)

var applyCmd = &cobra.Command{
	Use:   "apply <file-or-url>",
	Short: "Rebuild — or verify (--check) — a workstation from a snapshot",
	Long: `Read an opsforge snapshot (created with 'opsforge snapshot') from a local
file or an http(s) URL and rebuild that workstation here: install the
missing tools, restore the custom profiles, pin the theme, restore the
guard policy, and enable the shell environment if it was enabled.

Shows the full plan and asks for confirmation before changing anything
(use --yes to skip in scripts).

With --check, compare this machine to the snapshot WITHOUT changing
anything and exit non-zero if it has drifted — a CI-native way to assert
that your workstation or build image still matches a snapshot you froze.
Combine with --json for a structured drift report.`,
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
		if applyCheck {
			return runCheck(snap, cat)
		}
		return runApply(snap, cat)
	},
}

// runCheck implements `apply --check`: read-only drift detection against the
// baseline. Exits non-zero (via a returned error) when the machine drifts,
// so a CI job fails loudly.
func runCheck(snap snapshot.Snapshot, cat *catalog.Catalog) error {
	statuses := detect.All(cat.Tools())
	cur := currentState(cat, statuses)
	report := snapshot.CheckDrift(snap, cat, cur)

	if output.JSON {
		if err := output.Emit(report); err != nil {
			return err
		}
		if !report.Compliant {
			return errDrift
		}
		return nil
	}

	fmt.Println(ui.Header("opsforge apply --check", "verify this machine against your snapshot"))
	fmt.Println()

	if report.Compliant {
		fmt.Println(applyNew.Render(ui.MarkOK + " Compliant — this machine matches the baseline."))
		if len(report.UnknownTools) > 0 {
			fmt.Printf("  %s\n", applyWarn.Render(fmt.Sprintf(
				"note: %d tool(s) unknown to this catalog, not checked: %s",
				len(report.UnknownTools), strings.Join(report.UnknownTools, ", "))))
		}
		return nil
	}

	fmt.Println(applyErr.Render(ui.MarkErr + " Drift detected — this machine does not match the baseline."))
	fmt.Println()

	if len(report.MissingTools) > 0 {
		fmt.Println(applyHead.Render("Missing tools"))
		for _, t := range report.MissingTools {
			fmt.Printf("  %s %s\n", applyErr.Render(ui.MarkErr), t)
		}
		fmt.Println()
	}

	if len(report.Drift) > 0 {
		fmt.Println(applyHead.Render("Configuration drift"))
		for _, d := range report.Drift {
			fmt.Printf("  %s %s\n", applyErr.Render(ui.MarkErr), d.Detail)
			fmt.Printf("      %s\n", applyDim.Render(fmt.Sprintf(
				"expected %q · got %q", d.Expected, d.Actual)))
		}
		fmt.Println()
	}

	if len(report.UnknownTools) > 0 {
		fmt.Printf("%s\n", applyWarn.Render(fmt.Sprintf(
			"%d tool(s) unknown to this catalog, not checked: %s",
			len(report.UnknownTools), strings.Join(report.UnknownTools, ", "))))
	}

	fmt.Println(applyDim.Render("Bring this machine into line by re-running without --check."))
	return errDrift
}

// errDrift is returned by --check on drift so Execute exits non-zero without
// printing a scary stack — the report above already explained everything.
var errDrift = fmt.Errorf("machine has drifted from the baseline")

// runApply implements the normal (mutating) apply: plan, confirm, execute.
func runApply(snap snapshot.Snapshot, cat *catalog.Catalog) error {
	fmt.Println(ui.Header("opsforge apply", "rebuild this workstation from a snapshot"))
	fmt.Println()
	fmt.Println(applyDim.Render("Scanning this machine…"))
	statuses := detect.All(cat.Tools())
	plan := snapshot.BuildPlan(snap, cat, statuses)

	// Non-tool restorations gated on whether they'd actually change anything.
	restoreTheme := snap.Theme.Persisted && snap.Theme.Name != "" &&
		(!ui.ThemePersisted() || ui.Active.Name != snap.Theme.Name)
	restoreGuards := strings.TrimSpace(snap.Guards.YAML) != "" && guardsNeedRestore(snap.Guards.YAML)
	enableShell := snap.Shell.Enabled && !shellcfg.InstalledInZshrc()

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
	if restoreTheme {
		fmt.Printf("  %s\n", applyDim.Render("theme will be set to "+snap.Theme.Name))
	}
	if restoreGuards {
		fmt.Printf("  %s\n", applyDim.Render("guard policy (guards.yaml) will be restored"))
	}
	if enableShell {
		fmt.Printf("  %s\n", applyDim.Render("shell environment will be enabled"))
	}
	if snap.Versions.Manager != "" && string(versions.Detect()) != snap.Versions.Manager {
		fmt.Printf("  %s\n", applyWarn.Render(fmt.Sprintf(
			"version manager %q recorded in snapshot but not installed here (install it separately)",
			snap.Versions.Manager)))
	}
	if len(plan.Install) == 0 && len(snap.Profiles) == 0 &&
		!restoreTheme && !restoreGuards && !enableShell {
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

	if restoreTheme {
		if err := ui.SaveTheme(snap.Theme.Name); err != nil {
			fmt.Printf("%s theme %q: %v\n", applyErr.Render("✗"), snap.Theme.Name, err)
			failed++
		} else {
			fmt.Printf("%s theme %-10s set\n", applyOK.Render("✓"), snap.Theme.Name)
		}
	}

	if restoreGuards {
		if err := restoreGuardPolicy(snap.Guards.YAML); err != nil {
			fmt.Printf("%s guard policy: %v\n", applyErr.Render("✗"), err)
			failed++
		} else {
			fmt.Printf("%s guard policy restored\n", applyOK.Render("✓"))
		}
	}

	if enableShell {
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
}

// currentState reads the live machine state that --check diffs the snapshot
// against. All environment access lives here so snapshot.CheckDrift stays
// pure. Reads are passive — never invokes kubectl/gcloud.
func currentState(cat *catalog.Catalog, statuses map[string]detect.Status) snapshot.CurrentState {
	installed := make(map[string]bool, len(statuses))
	for _, t := range cat.Tools() {
		installed[t.Name] = statuses[t.Name].Installed
	}
	return snapshot.CurrentState{
		Installed:      installed,
		ShellEnabled:   shellcfg.InstalledInZshrc(),
		ThemeName:      ui.Active.Name,
		ThemePersisted: ui.ThemePersisted(),
		GuardsYAML:     readGuardsFile(),
		VersionManager: string(versions.Detect()),
	}
}

// readGuardsFile reads the machine's guards.yaml verbatim, or "" when absent.
func readGuardsFile() string {
	path, err := shellcfg.PolicyPath()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// guardsNeedRestore reports whether writing the snapshot's guards.yaml would
// actually change the machine (content differs from what's already there).
func guardsNeedRestore(want string) bool {
	have := readGuardsFile()
	return normalizeForCompare(have) != normalizeForCompare(want)
}

func normalizeForCompare(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}

// restoreGuardPolicy validates the snapshot's guard YAML and writes it to
// the user's guards.yaml. Validating first means we never install a policy
// this build can't parse (which would disable all guards at runtime).
func restoreGuardPolicy(content string) error {
	if _, err := shellcfg.ParsePolicy([]byte(content)); err != nil {
		return fmt.Errorf("refusing to restore an invalid guard policy: %w", err)
	}
	path, err := shellcfg.PolicyPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func init() {
	applyCmd.Flags().BoolVarP(&applyYes, "yes", "y", false, "apply without asking for confirmation")
	applyCmd.Flags().BoolVar(&applyCheck, "check", false,
		"verify this machine against the snapshot without changing anything (CI baseline); exits non-zero on drift")
	rootCmd.AddCommand(applyCmd)
}
