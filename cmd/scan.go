package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/imagescan"
	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/ui"
)

var scanDiff bool

var scanCmd = &cobra.Command{
	Use:   "scan <image>",
	Short: "Scan a container image for CVEs and correlate it with your workstation",
	Long: `Scan a local container image for known CVEs, then — the part other scanners
don't do — correlate the result with your own workstation.

opsforge doesn't re-implement image SBOM extraction: it drives syft or trivy
(whichever is installed) to produce the image's SBOM, then runs those
components through opsforge's OWN OSV engine — the same one 'opsforge audit'
uses on your machine. With --diff, it also compares the image against the tools
installed on your workstation and reports VERSION drift: a tool you run locally
that ships at a different version in the image.

  opsforge scan nginx:latest              # CVEs in the image
  opsforge scan my-ci-image --diff        # + how it drifts from your machine
  opsforge scan my-image --json           # machine-readable report`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image := args[0]

		backend, ok := imagescan.DetectBackend()
		if !ok {
			return fmt.Errorf("no image SBOM backend found — install one of: %v "+
				"(e.g. `opsforge install syft`)", imagescan.BackendNames())
		}

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		if !output.JSON {
			fmt.Fprintln(os.Stderr, ui.Dim.Render(
				fmt.Sprintf("  generating SBOM for %s via %s…", image, backend.Bin)))
		}
		comps, err := imagescan.GenerateSBOM(ctx, backend, image)
		if err != nil {
			return err
		}

		findings := imagescan.ScanComponents(ctx, comps)

		var drift []imagescan.Drift
		if scanDiff {
			cat, err := catalog.Load()
			if err != nil {
				return err
			}
			drift = imagescan.Correlate(comps, workstationTools(cat))
		}

		if output.JSON {
			if err := emitScanJSON(image, backend.Bin, comps, findings, drift); err != nil {
				return err
			}
		} else {
			printScanHuman(image, backend.Bin, comps, findings, drift, scanDiff)
		}

		// Non-zero exit on a HIGH/CRITICAL CVE (both output modes), so `scan`
		// gates CI like `audit`.
		highOrCritical := 0
		for _, f := range findings {
			if f.TopSeverity == audit.SevHigh.String() || f.TopSeverity == audit.SevCritical.String() {
				highOrCritical++
			}
		}
		if highOrCritical > 0 {
			return fmt.Errorf("%d image component(s) with HIGH or CRITICAL vulnerabilities", highOrCritical)
		}
		return nil
	},
}

// workstationTools lists installed, OSV-mapped tools as (name, normalized
// version) for the --diff correlation.
func workstationTools(cat *catalog.Catalog) []imagescan.WorkstationTool {
	var out []imagescan.WorkstationTool
	for _, t := range cat.Tools() {
		s := detect.Tool(t)
		if !s.Installed {
			continue
		}
		out = append(out, imagescan.WorkstationTool{
			Name: t.Name, Version: audit.NormalizeVersion(s.Version),
		})
	}
	return out
}

func emitScanJSON(image, backend string, comps []imagescan.Component, findings []imagescan.ImageFinding, drift []imagescan.Drift) error {
	highOrCritical := 0
	for _, f := range findings {
		if f.TopSeverity == audit.SevHigh.String() || f.TopSeverity == audit.SevCritical.String() {
			highOrCritical++
		}
	}
	return output.Emit(struct {
		Image           string                   `json:"image"`
		Backend         string                   `json:"backend"`
		ComponentsTotal int                      `json:"components_scanned"`
		HighOrCritical  int                      `json:"high_or_critical"`
		Findings        []imagescan.ImageFinding `json:"findings"`
		Drift           []imagescan.Drift        `json:"workstation_drift,omitempty"`
	}{
		Image:           image,
		Backend:         backend,
		ComponentsTotal: len(comps),
		HighOrCritical:  highOrCritical,
		Findings:        findings,
		Drift:           drift,
	})
}

func printScanHuman(image, backend string, comps []imagescan.Component, findings []imagescan.ImageFinding, drift []imagescan.Drift, showDrift bool) {
	fmt.Println(ui.Header("opsforge scan", image+"  ·  via "+backend))
	fmt.Println()
	fmt.Printf("  %s\n", ui.Dim.Render(fmt.Sprintf("%d OSV-mapped component(s) scanned", len(comps))))
	fmt.Println()

	if len(findings) == 0 {
		fmt.Println(ui.OKBold.Render("  ✓ No known CVEs in the image's mapped components."))
	} else {
		// Worst first.
		sort.Slice(findings, func(a, b int) bool {
			return sevRank(findings[a].TopSeverity) > sevRank(findings[b].TopSeverity)
		})
		for _, f := range findings {
			fmt.Printf("  %s %s %s\n", ui.WarnMark(),
				ui.Warn.Render(f.Name+" "+f.Version),
				ui.Dim.Render("["+f.TopSeverity+"]"))
			for _, v := range f.Vulns {
				fix := ""
				if v.FixedIn != "" {
					fix = ui.Dim.Render("  → fixed in " + v.FixedIn)
				}
				fmt.Printf("      %s %s%s\n", ui.Dim.Render("["+v.Severity.String()+"]"), v.ID, fix)
			}
		}
	}

	if showDrift {
		fmt.Println()
		fmt.Println(ui.OKBold.Render("  Workstation correlation"))
		if len(drift) == 0 {
			fmt.Printf("    %s\n", ui.Dim.Render("none of your installed tools appear in this image"))
		} else {
			differ := false
			for _, d := range drift {
				if d.VersionDiffer {
					differ = true
					fmt.Printf("    %s %s\n", ui.WarnMark(), ui.Warn.Render(
						fmt.Sprintf("%s — you run %s, image ships %s", d.Name, d.WorkstationV, d.ImageV)))
				}
			}
			if !differ {
				fmt.Printf("    %s\n", ui.Dim.Render(
					fmt.Sprintf("%d shared tool(s), no version drift", len(drift))))
			}
		}
	}
}

func sevRank(s string) int {
	switch s {
	case audit.SevCritical.String():
		return 4
	case audit.SevHigh.String():
		return 3
	case audit.SevMedium.String():
		return 2
	case audit.SevLow.String():
		return 1
	}
	return 0
}

func init() {
	scanCmd.Flags().BoolVar(&scanDiff, "diff", false,
		"correlate the image against your workstation and report version drift")
	rootCmd.AddCommand(scanCmd)
}
