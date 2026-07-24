package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/installer"
	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/project"
	"github.com/Mrg77/opsforge/internal/ui"
)

var (
	syncCheck bool
	syncInit  bool
	syncYes   bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Install the tools a project's opsforge.yaml declares (reproducible env)",
	Long: `Read the nearest opsforge.yaml (walking up from the current directory) and
install the tools the project needs — so a repo carries its toolchain and
anyone reproduces it with one command.

  opsforge sync           # install what the project declares
  opsforge sync --check    # report drift, exit non-zero if a tool is missing (CI)
  opsforge sync --init     # write a starter opsforge.yaml here

Unlike mise/devbox, an opsforge.yaml can also gate on CVEs: set
fail_on: high|critical and sync (or --check) fails when a required tool
carries a vulnerability at that level.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if syncInit {
			return syncInitManifest()
		}

		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		path, ok := project.Find(wd)
		if !ok {
			return fmt.Errorf("no %s found here or in any parent directory (run `opsforge sync --init`)", project.FileName)
		}
		proj, err := project.Load(path)
		if err != nil {
			return err
		}
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		statuses := detect.All(cat.Tools())
		plan := project.BuildPlan(proj, cat, statuses)

		if syncCheck {
			return syncReport(proj, cat, plan, path, statuses)
		}
		return syncApply(proj, cat, plan, path)
	},
}

// syncReport is the CI path: it reports drift and CVE gate status without
// installing anything, exiting non-zero on any drift or gate failure. When
// an opsforge.lock exists it also flags VERSION drift (a tool installed but
// at a different version than the lock pinned).
func syncReport(proj *project.Project, cat *catalog.Catalog, plan project.Plan, manifestPath string, statuses map[string]detect.Status) error {
	gate := cveGate(proj, cat)

	// Version drift against the lockfile (if one was committed).
	var lockDrift []project.LockDrift
	if l, ok, err := project.ReadLock(project.LockPath(manifestPath)); err != nil {
		return err
	} else if ok {
		lockDrift = project.CheckLock(l, statuses)
	}

	compliant := len(plan.Install) == 0 && len(gate) == 0 && len(lockDrift) == 0

	if output.JSON {
		if err := output.Emit(struct {
			Compliant    bool                `json:"compliant"`
			Missing      []string            `json:"missing"`
			Present      []string            `json:"present"`
			Unknown      []string            `json:"unknown"`
			CVEBlocked   []string            `json:"cve_blocked"`
			VersionDrift []project.LockDrift `json:"version_drift"`
			FailOn       string              `json:"fail_on,omitempty"`
		}{
			Compliant:    compliant,
			Missing:      plan.Install,
			Present:      plan.Present,
			Unknown:      plan.Unknown,
			CVEBlocked:   gate,
			VersionDrift: lockDrift,
			FailOn:       proj.FailOn,
		}); err != nil {
			return err
		}
	} else {
		fmt.Println(ui.Header("opsforge sync --check", "does this machine match the project's opsforge.yaml?"))
		fmt.Println()
		if compliant {
			fmt.Println(ui.OKBold.Render("Compliant — every required tool is installed at the locked version."))
		} else {
			for _, t := range plan.Install {
				fmt.Printf("  %s %s\n", ui.ErrMark(), ui.Err.Render(t+" — missing"))
			}
			for _, d := range lockDrift {
				if d.Got == "" {
					continue // already reported as missing above
				}
				fmt.Printf("  %s %s\n", ui.ErrMark(),
					ui.Err.Render(fmt.Sprintf("%s — locked %s, have %s", d.Name, d.Expected, d.Got)))
			}
			for _, g := range gate {
				fmt.Printf("  %s %s\n", ui.ErrMark(), ui.Err.Render(g))
			}
		}
		for _, u := range plan.Unknown {
			fmt.Printf("  %s %s\n", ui.WarnMark(), ui.Dim.Render(u+" — not in the catalog"))
		}
	}

	if !compliant {
		return fmt.Errorf("project is not in sync (%d missing, %d version drift, %d CVE-blocked)",
			len(plan.Install), len(lockDrift), len(gate))
	}
	return nil
}

// syncApply installs the missing tools (with confirmation unless --yes),
// then enforces the CVE gate if the manifest sets one and writes an
// opsforge.lock pinning the resolved versions.
func syncApply(proj *project.Project, cat *catalog.Catalog, plan project.Plan, manifestPath string) error {
	fmt.Println(ui.Header("opsforge sync", "reproducing this project's toolchain"))
	fmt.Println()
	for _, u := range plan.Unknown {
		fmt.Printf("  %s %s\n", ui.WarnMark(), ui.Dim.Render(u+" — not in the catalog, skipped"))
	}
	if len(plan.Install) == 0 {
		fmt.Println(ui.OKBold.Render("Nothing to do — every required tool is already installed."))
	} else {
		if !syncYes {
			fmt.Printf("Will install %s:\n", plural(len(plan.Install), "tool"))
			for _, t := range plan.Install {
				fmt.Printf("  %s %s\n", ui.MissMark(), t)
			}
			if !confirm("Proceed?") {
				fmt.Println(ui.Dim.Render("Aborted."))
				return nil
			}
		}
		failed := 0
		for _, name := range plan.Install {
			t, _ := cat.Tool(name)
			fmt.Printf("… installing %s (via %s)\n", name, installer.BackendFor(t))
			if res := installer.Install(t); res.Err != nil {
				fmt.Printf("%s %-16s %v\n", ui.ErrMark(), name, res.Err)
				failed++
				continue
			}
			fmt.Printf("%s %s installed\n", ui.OKMark(), name)
		}
		if failed > 0 {
			return fmt.Errorf("%d tool(s) failed to install", failed)
		}
	}

	// Enforce the project's CVE gate after installing.
	if gate := cveGate(proj, cat); len(gate) > 0 {
		fmt.Println()
		for _, g := range gate {
			fmt.Printf("  %s %s\n", ui.ErrMark(), ui.Err.Render(g))
		}
		return fmt.Errorf("%d required tool(s) blocked by the fail_on: %s gate", len(gate), proj.FailOn)
	}

	// Pin the resolved versions so a reviewer reproduces the exact toolchain.
	// Re-detect after installing so freshly-installed versions land in the
	// lock (the plan's statuses predate the installs).
	required := append(append([]string{}, plan.Present...), plan.Install...)
	statuses := detect.All(cat.Tools())
	lock := project.BuildLock(required, statuses)
	if err := project.WriteLock(project.LockPath(manifestPath), lock); err != nil {
		return fmt.Errorf("writing %s: %w", project.LockFileName, err)
	}
	fmt.Println()
	fmt.Printf("%s %s\n", ui.OKMark(),
		ui.Dim.Render(fmt.Sprintf("Pinned %d tool(s) to %s", len(lock.Tools), project.LockFileName)))
	return nil
}

// cveGate returns human messages for required tools whose worst CVE meets
// or exceeds the manifest's fail_on threshold. Empty when no gate is set
// or nothing crosses it.
func cveGate(proj *project.Project, cat *catalog.Catalog) []string {
	if proj.FailOn == "" {
		return nil
	}
	threshold := audit.SevHigh
	if proj.FailOn == "critical" {
		threshold = audit.SevCritical
	}
	// Only audit the tools this project requires.
	want := map[string]bool{}
	tools, _ := proj.ResolveTools(cat)
	for _, t := range tools {
		want[t] = true
	}
	var targets []audit.ToolTarget
	for _, tg := range CollectOSVTargets(cat) {
		if want[tg.Name] {
			targets = append(targets, tg)
		}
	}
	if len(targets) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	var blocked []string
	for _, f := range audit.ScanTools(ctx, targets) {
		if f.TopSeverity() >= threshold {
			blocked = append(blocked, fmt.Sprintf("%s has a %s CVE (fail_on: %s)",
				f.Tool, f.TopSeverity(), proj.FailOn))
		}
	}
	return blocked
}

func syncInitManifest() error {
	if _, err := os.Stat(project.FileName); err == nil {
		return fmt.Errorf("%s already exists here", project.FileName)
	}
	if err := os.WriteFile(project.FileName, []byte(project.ExampleManifest), 0o644); err != nil {
		return err
	}
	fmt.Printf("%s wrote %s\n", ui.OKMark(), project.FileName)
	fmt.Println(ui.Dim.Render("  Edit it, commit it, then anyone can run `opsforge sync`."))
	return nil
}

func init() {
	syncCmd.Flags().BoolVar(&syncCheck, "check", false,
		"report drift and exit non-zero if a required tool is missing (CI)")
	syncCmd.Flags().BoolVar(&syncInit, "init", false,
		"write a starter opsforge.yaml in the current directory")
	syncCmd.Flags().BoolVar(&syncYes, "yes", false,
		"install without asking for confirmation")
	rootCmd.AddCommand(syncCmd)
}
