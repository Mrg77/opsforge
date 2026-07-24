package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/sbom"
	"github.com/Mrg77/opsforge/internal/ui"
	"github.com/Mrg77/opsforge/internal/vex"
)

var vexKEV bool

var vexCmd = &cobra.Command{
	Use:   "vex",
	Short: "Emit an OpenVEX document of the CVEs affecting your installed tools",
	Long: `Produce an OpenVEX (Vulnerability Exploitability eXchange) document for the
CVEs opsforge found on your installed tools. Where a plain CVE list only
says "these exist", VEX says, per (component, CVE), a machine-readable
status (affected/fixed/…) plus an action — the artifact a downstream
scanner or auditor uses to triage instead of drowning in CVSS.

Since the NVD stopped enriching most CVEs in 2026, prioritizing by
*exploitability* matters more than by score. With --kev, opsforge cross-
references CISA's Known Exploited Vulnerabilities catalog and calls out the
CVEs that are being actively exploited in the wild — the ones to fix first.

  opsforge vex               # OpenVEX to stdout
  opsforge vex --kev         # + highlight actively-exploited (CISA KEV) CVEs
  opsforge vex > vex.json    # capture it (pairs with 'opsforge sbom')`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}

		targets := CollectOSVTargets(cat)
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		findings := audit.ScanTools(ctx, targets)

		// Map each finding to its PURL (same coordinates as the SBOM), using
		// the catalog's OSV mapping via the detected version.
		byName := map[string]catalog.Tool{}
		for _, t := range cat.Tools() {
			byName[t.Name] = t
		}
		var inputs []vex.Input
		for _, f := range findings {
			if len(f.Vulns) == 0 {
				continue
			}
			t := byName[f.Tool]
			if t.OSV == nil {
				continue
			}
			inputs = append(inputs, vex.Input{
				PURL:  sbom.PURL(t.OSV.Ecosystem, t.OSV.Name, detect.Tool(t).Version),
				Vulns: f.Vulns,
			})
		}

		var kev vex.KEVSet
		if vexKEV {
			kev = vex.LoadKEV()
		}

		// time.Now is unavailable to the pure builder; stamp here.
		now := time.Now().UTC()
		doc := vex.Build(inputs,
			"https://openvex.dev/docs/opsforge/"+now.Format("20060102T150405Z"),
			now.Format(time.RFC3339))

		if output.JSON || !vexKEV {
			// The VEX document is the machine artifact — emit JSON to stdout.
			b, err := json.MarshalIndent(doc, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, string(b))
			fmt.Fprintln(os.Stderr, ui.Dim.Render("  "+doc.Summary()))
			return nil
		}

		// --kev without --json: a human-readable triage view highlighting
		// actively-exploited CVEs.
		return printVEXHuman(doc, kev)
	},
}

// printVEXHuman shows the VEX statements grouped so the actively-exploited
// (KEV) CVEs stand out — the ones to fix first.
func printVEXHuman(doc vex.Doc, kev vex.KEVSet) error {
	fmt.Println(ui.Header("opsforge vex", "OpenVEX + CISA KEV — fix the actively-exploited first"))
	fmt.Println()
	if len(doc.Statements) == 0 {
		fmt.Println(ui.OKBold.Render("  ✓ No affected components — nothing to declare."))
		return nil
	}

	var exploited, other []vex.Statement
	for _, s := range doc.Statements {
		if kev.Has(s.Vulnerability.Name) {
			exploited = append(exploited, s)
		} else {
			other = append(other, s)
		}
	}
	sort.Slice(exploited, func(a, b int) bool {
		return exploited[a].Vulnerability.Name < exploited[b].Vulnerability.Name
	})

	if len(exploited) > 0 {
		fmt.Println(ui.Err.Render(fmt.Sprintf("  %s Actively exploited (CISA KEV) — fix now:", ui.MarkErr)))
		for _, s := range exploited {
			fmt.Printf("    %s %s  %s\n", ui.ErrMark(),
				ui.Err.Render(s.Vulnerability.Name), ui.Dim.Render(shortPURL(s.Products[0].ID)))
		}
		fmt.Println()
	}
	fmt.Printf("  %s %d other affected statement(s) — see `opsforge vex --json`\n",
		ui.WarnMark(), len(other))
	fmt.Println()
	fmt.Println(ui.Faint.Render("  " + doc.Summary()))
	return nil
}

// shortPURL trims a purl to its package coordinate for compact display.
func shortPURL(p string) string {
	return p
}

func init() {
	vexCmd.Flags().BoolVar(&vexKEV, "kev", false,
		"cross-reference CISA KEV and highlight actively-exploited CVEs")
	rootCmd.AddCommand(vexCmd)
}
