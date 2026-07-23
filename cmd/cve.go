package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/cve"
	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/ui"
)

// cveCmd groups the cached-CVE plumbing behind the human-facing `status`
// and the shell prompt. `cve refresh` runs the scan and writes the cache;
// `cve check` prints the cached one-line status (used by the prompt).
var cveCmd = &cobra.Command{
	Use:    "cve",
	Short:  "Cached CVE status for installed tools (used by status/prompt)",
	Hidden: true,
}

// cveRefreshCmd scans installed tools against OSV and writes the cache. It
// is meant to run in the background (the prompt/status kick it off when the
// cache is stale) so the interactive path never waits on the network.
var cveRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Scan installed tools and update the CVE cache",
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		targets := CollectOSVTargets(cat)
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		findings := audit.ScanTools(ctx, targets)
		return cve.Save(cve.Summarize(findings, time.Now().UTC()))
	},
}

// cveCheckCmd prints a short cached status. Empty output means "nothing to
// report" (no cache yet, or no vulnerable tools) so the prompt can splice
// it in without noise. With --refresh-stale it triggers a background
// refresh when the cache is old — but still prints only the cached value,
// never blocking.
var cveCheckRefresh bool

var cveCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Print the cached CVE one-liner (used by the prompt)",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, ok := cve.Load()

		if cveCheckRefresh && (!ok || s.Stale(cve.DefaultTTL, time.Now().UTC())) {
			refreshCVECacheInBackground()
		}

		if output.JSON {
			return output.Emit(s)
		}
		if ok && s.Vulnerable > 0 {
			// A compact, prompt-friendly line.
			word := "CVE"
			if s.Vulnerable > 1 {
				word = "CVEs"
			}
			sev := ""
			if s.HighOrCritical > 0 {
				sev = fmt.Sprintf(" (%d high/critical)", s.HighOrCritical)
			}
			fmt.Printf("%s %d tool %s%s — run `opsforge audit`\n",
				ui.WarnMark(), s.Vulnerable, word, sev)
		}
		return nil
	},
}

func init() {
	cveCheckCmd.Flags().BoolVar(&cveCheckRefresh, "refresh-stale", false,
		"trigger a background refresh if the cache is stale")
	cveCmd.AddCommand(cveRefreshCmd, cveCheckCmd)
	rootCmd.AddCommand(cveCmd)
}
