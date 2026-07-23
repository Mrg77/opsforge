package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/installer"
	"github.com/Mrg77/opsforge/internal/notices"
	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/secrets"
	"github.com/Mrg77/opsforge/internal/ui"
)

var (
	notifyRefresh bool
	notifyQuiet   bool
)

var notifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Everything on your machine that needs attention — CVEs, updates, secrets, self-update",
	Long: `A single digest of what opsforge thinks you should know about your own
workstation, read from a cache so it's instant and never blocks:

  - installed tools with a known CVE (HIGH/CRITICAL called out),
  - tools that can be updated,
  - credentials leaking in your shell history / rc / .env (when scanned),
  - a newer opsforge release than the one you're running.

  opsforge notify            # the full digest
  opsforge notify --json     # machine-readable
  opsforge notify --refresh  # recompute the cache now (else it self-refreshes when stale)

The shell prints a compact one-liner on startup when there's something to
report; this command shows the detail on demand.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if notifyRefresh {
			return refreshNotices()
		}

		d, ok := notices.Load()
		// Kick a background refresh when the cache is stale/absent so the
		// next call is accurate — never wait on it here.
		if !ok || d.Stale(notices.DefaultTTL, time.Now().UTC()) {
			refreshNoticesInBackground()
		}

		if output.JSON {
			return output.Emit(d)
		}

		items := d.Items()
		if notifyQuiet {
			if line := d.OneLine(); line != "" {
				fmt.Println(line)
			}
			return nil
		}

		fmt.Println(ui.Header("opsforge notify", "what needs your attention"))
		fmt.Println()
		if !ok || d.RefreshedAt.IsZero() {
			fmt.Println(ui.Dim.Render("  No digest yet — computing in the background. Try again in a moment."))
			return nil
		}
		if len(items) == 0 {
			fmt.Println(ui.OKBold.Render("  ✓ All clear — no CVEs, updates, leaks or new versions pending."))
			return nil
		}
		for _, it := range items {
			mark := ui.WarnMark()
			style := ui.Warn
			switch it.Severity {
			case notices.SevCritical:
				mark, style = ui.ErrMark(), ui.Err
			case notices.SevInfo:
				mark, style = ui.OKMark(), ui.OK
			}
			fmt.Printf("  %s %s\n", mark, style.Render(it.Text))
			fmt.Printf("      %s %s\n", ui.Dim.Render(ui.MarkArrow), ui.Dim.Render(it.Fix))
		}
		fmt.Println()
		fmt.Println(ui.Faint.Render(fmt.Sprintf("  as of %s · refresh with `opsforge notify --refresh`",
			d.RefreshedAt.Local().Format("15:04"))))
		return nil
	},
}

// refreshNotices recomputes the whole digest from live sources and caches
// it. It's the one place that touches the network (OSV, brew-outdated,
// the GitHub latest-release), so callers run it in the background.
func refreshNotices() error {
	cat, err := catalog.Load()
	if err != nil {
		return err
	}
	statuses := detect.AllWithOutdated(cat.Tools())

	var d notices.Digest
	d.RefreshedAt = time.Now().UTC()

	// Updates.
	for _, t := range cat.Tools() {
		if statuses[t.Name].Outdated {
			d.Updates++
		}
	}

	// CVEs on installed tools.
	targets := CollectOSVTargets(cat)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	for _, f := range audit.ScanTools(ctx, targets) {
		if len(f.Vulns) == 0 {
			continue
		}
		d.CVETools++
		if f.TopSeverity() >= audit.SevHigh {
			d.CVEHighOrCritical++
		}
	}

	// Leaked secrets (local, instant).
	d.SecretsScanned = true
	for _, s := range secrets.ScanWorkstation() {
		d.Secrets++
		if s.Rule.Severity == secrets.SevCritical {
			d.SecretsCritical++
		}
	}

	// A newer opsforge? Skipped on a dev build (nothing to compare).
	if latest, err := installer.LatestSelfVersion(); err == nil {
		chk := installer.NewerAvailable(version, latest)
		if chk.Available {
			d.SelfUpdate = true
			d.SelfLatestVersion = chk.Latest
		}
	}

	return notices.Save(d)
}

// refreshNoticesInBackground spawns `opsforge notify --refresh` detached,
// so a stale cache updates without holding up the current command.
func refreshNoticesInBackground() {
	spawnDetached("notify", "--refresh")
}

// spawnDetached runs opsforge again with the given args, fully detached:
// we don't wait for it and it survives our exit. Used to refresh caches
// in the background so interactive commands stay instant.
func spawnDetached(args ...string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	c := exec.Command(exe, args...)
	c.Stdout, c.Stderr = nil, nil
	if err := c.Start(); err == nil && c.Process != nil {
		_ = c.Process.Release()
	}
}

func init() {
	notifyCmd.Flags().BoolVar(&notifyRefresh, "refresh", false,
		"recompute the digest from live sources and update the cache")
	notifyCmd.Flags().BoolVar(&notifyQuiet, "quiet", false,
		"print only the compact one-liner (used by the shell)")
	rootCmd.AddCommand(notifyCmd)
}
