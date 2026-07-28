package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/ai"
	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/secrets"
	"github.com/Mrg77/opsforge/internal/ui"
)

var adviseCmd = &cobra.Command{
	Use:   "advise",
	Short: "Ask the AI to prioritize your workstation's CVEs, updates & secrets",
	Long: `opsforge already DETECTS what needs attention (CVEs, outdated tools, leaked
credentials). 'advise' asks your AI backend to turn that pile into a plan: what to
fix first and why, in plain language.

The deterministic scanners do the detection; the AI only explains and prioritizes
what they found — it never invents findings. Only tool names, versions and finding
titles are sent (never secret values or file contents).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		backend := ai.Detect()
		if backend == nil {
			fmt.Println(ui.Dim.Render(ai.SetupHint()))
			return fmt.Errorf("no AI backend available")
		}

		fmt.Println(ui.Header("opsforge advise", "AI-prioritized plan for your workstation"))
		fmt.Println()
		fmt.Println(ui.Dim.Render("  Scanning (CVEs · updates · credentials)…"))

		ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
		defer cancel()
		facts, empty := gatherAdviceFacts(ctx)
		if empty {
			fmt.Printf("\n  %s %s\n", ui.OKMark(), ui.OK.Render("Nothing flagged — your workstation looks clean."))
			return nil
		}

		prompt := advicePrompt(facts)
		fmt.Println(ui.Dim.Render("  Asking " + backend.Name + "…\n"))
		return backend.Run(ctx, prompt)
	},
}

// gatherAdviceFacts collects the deterministic findings the AI will prioritize,
// as a compact text block. Returns empty=true when nothing was flagged.
func gatherAdviceFacts(ctx context.Context) (string, bool) {
	var b strings.Builder
	found := false

	cat, err := catalog.Load()
	if err != nil {
		return "", true
	}

	// Outdated tools.
	statuses := detect.AllWithOutdated(cat.Tools())
	var outdated []string
	for _, t := range cat.Tools() {
		if s := statuses[t.Name]; s.Outdated {
			outdated = append(outdated, fmt.Sprintf("%s (installed %s)", t.Name, s.Version))
		}
	}
	if len(outdated) > 0 {
		found = true
		fmt.Fprintf(&b, "OUTDATED TOOLS (%d):\n", len(outdated))
		for _, o := range outdated {
			fmt.Fprintf(&b, "  - %s\n", o)
		}
	}

	// CVEs on installed tools — the highest-severity per tool, named.
	targets := CollectOSVTargets(cat)
	type cveRow struct {
		tool string
		sev  audit.Severity
		n    int
	}
	var rows []cveRow
	for _, f := range audit.ScanTools(ctx, targets) {
		if len(f.Vulns) == 0 {
			continue
		}
		rows = append(rows, cveRow{f.Tool, f.TopSeverity(), len(f.Vulns)})
	}
	if len(rows) > 0 {
		found = true
		sort.Slice(rows, func(i, j int) bool { return rows[i].sev > rows[j].sev })
		fmt.Fprintf(&b, "\nTOOLS WITH KNOWN CVEs (%d):\n", len(rows))
		for _, r := range rows {
			fmt.Fprintf(&b, "  - %s: %d CVE(s), highest severity %s\n", r.tool, r.n, r.sev)
		}
	}

	// Leaked / risky credentials — titles only, never values.
	sec := secrets.ScanWorkstation()
	if len(sec) > 0 {
		found = true
		fmt.Fprintf(&b, "\nCREDENTIAL RISKS (%d):\n", len(sec))
		for _, s := range sec {
			// Rule.Desc + the source file only — never the matched secret value.
			fmt.Fprintf(&b, "  - [%s] %s (in %s)\n", s.Rule.Severity, s.Rule.Desc, s.Source)
		}
	}

	return b.String(), !found
}

func advicePrompt(facts string) string {
	return `You are the advisor inside opsforge, a DevOps workstation manager. A deterministic
scanner produced the findings below about the user's own machine. Turn them into a
short, prioritized remediation plan for a DevOps/SRE.

Rules:
- Lead with the single most urgent thing and why it matters (an unauthenticated RCE
  in an installed tool outranks a cosmetic update).
- Name the specific tool/finding; give the exact opsforge or shell command to fix it
  when there is one (e.g. "opsforge upgrade -u", "opsforge upgrade argocd").
- Group the rest briefly. Don't restate every line — prioritize.
- Be honest: a CVE in a tool you don't run as a server is lower risk than one you do.
- Keep it under ~150 words. No preamble.

FINDINGS:
` + facts
}

func init() {
	rootCmd.AddCommand(adviseCmd)
}
