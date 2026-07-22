package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/secrets"
	"github.com/Mrg77/opsforge/internal/ui"
)

var auditSecrets bool

// CollectOSVTargets gathers installed tools with an OSV mapping and a
// parsable version — shared by the audit command and the TUI.
func CollectOSVTargets(cat *catalog.Catalog) []audit.ToolTarget {
	var targets []audit.ToolTarget
	for _, t := range cat.Tools() {
		if t.OSV == nil {
			continue
		}
		s := detect.Tool(t)
		if !s.Installed {
			continue
		}
		ver := audit.NormalizeVersion(s.Version)
		if ver == "" {
			continue
		}
		targets = append(targets, audit.ToolTarget{
			Name: t.Name, Ecosystem: t.OSV.Ecosystem, Package: t.OSV.Name, Version: ver,
		})
	}
	return targets
}

// auditJSON runs the same scans as the human path but emits a structured
// report and sets a non-zero exit on HIGH/CRITICAL CVEs or critical
// secret leaks — the shape a CI gate consumes.
func auditJSON(cat *catalog.Catalog) error {
	type vulnJSON struct {
		ID       string `json:"id"`
		Severity string `json:"severity"`
		Summary  string `json:"summary"`
		FixedIn  string `json:"fixed_in,omitempty"`
	}
	type toolJSON struct {
		Tool            string     `json:"tool"`
		Version         string     `json:"version"`
		TopSeverity     string     `json:"top_severity"`
		Vulnerable      bool       `json:"vulnerable"`
		Vulnerabilities []vulnJSON `json:"vulnerabilities"`
	}
	type secretJSON struct {
		Source   string `json:"source"`
		Line     int    `json:"line"`
		Rule     string `json:"rule"`
		Severity string `json:"severity"`
		Excerpt  string `json:"excerpt"`
	}

	targets := CollectOSVTargets(cat)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	findings := audit.ScanTools(ctx, targets)
	sort.Slice(findings, func(a, b int) bool {
		if findings[a].TopSeverity() != findings[b].TopSeverity() {
			return findings[a].TopSeverity() > findings[b].TopSeverity()
		}
		return findings[a].Tool < findings[b].Tool
	})

	tools := make([]toolJSON, 0, len(findings))
	highOrWorse := 0
	for _, f := range findings {
		vulns := make([]vulnJSON, 0, len(f.Vulns))
		for _, v := range f.Vulns {
			vulns = append(vulns, vulnJSON{v.ID, v.Severity.String(), v.Summary, v.FixedIn})
		}
		if f.TopSeverity() >= audit.SevHigh {
			highOrWorse++
		}
		tools = append(tools, toolJSON{
			Tool: f.Tool, Version: f.Version, TopSeverity: f.TopSeverity().String(),
			Vulnerable: len(f.Vulns) > 0, Vulnerabilities: vulns,
		})
	}

	var leaks []secretJSON
	criticalSecrets := 0
	if auditSecrets {
		for _, s := range secrets.ScanWorkstation() {
			if s.Rule.Severity == secrets.SevCritical {
				criticalSecrets++
			}
			leaks = append(leaks, secretJSON{
				Source: s.Source, Line: s.Line, Rule: s.Rule.Desc,
				Severity: s.Rule.Severity.String(), Excerpt: s.Excerpt,
			})
		}
	}

	if err := output.Emit(struct {
		ToolsScanned    int          `json:"tools_scanned"`
		HighOrCritical  int          `json:"high_or_critical"`
		Tools           []toolJSON   `json:"tools"`
		SecretsScanned  bool         `json:"secrets_scanned"`
		CriticalSecrets int          `json:"critical_secrets"`
		Secrets         []secretJSON `json:"secrets"`
	}{len(targets), highOrWorse, tools, auditSecrets, criticalSecrets, leaks}); err != nil {
		return err
	}
	if highOrWorse > 0 || criticalSecrets > 0 {
		return fmt.Errorf("audit found %d HIGH/CRITICAL CVE tool(s) and %d critical secret(s)",
			highOrWorse, criticalSecrets)
	}
	return nil
}

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Scan installed tools for CVEs — and your workstation for leaked secrets",
	Long: `Cross-references the versions of your installed tools against the OSV.dev
vulnerability database and reports which ones have known CVEs and should be
upgraded. Only tools with an OSV mapping in the catalog are checked.

With --secrets, also scans the places credentials habitually leak — shell
history, shell rc files, and local .env files — and reports masked findings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}

		if output.JSON {
			return auditJSON(cat)
		}

		sub := "installed tool versions vs the OSV.dev vulnerability database"
		if auditSecrets {
			sub = "CVEs in your tools + credentials leaking on your workstation"
		}
		fmt.Println(ui.Header("opsforge audit", sub))
		fmt.Println()

		if auditSecrets {
			if err := runSecretsScan(); err != nil {
				return err
			}
			fmt.Println()
		}

		targets := CollectOSVTargets(cat)
		if len(targets) == 0 {
			fmt.Println("No auditable tools installed (no installed tool carries an OSV mapping).")
			return nil
		}

		fmt.Printf("Auditing %d installed tool(s) against OSV.dev…\n\n", len(targets))
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		findings := audit.ScanTools(ctx, targets)

		// Sort: most severe first, then by tool name.
		sort.Slice(findings, func(a, b int) bool {
			if findings[a].TopSeverity() != findings[b].TopSeverity() {
				return findings[a].TopSeverity() > findings[b].TopSeverity()
			}
			return findings[a].Tool < findings[b].Tool
		})

		vulnerable := 0
		highOrWorse := 0
		for _, f := range findings {
			if len(f.Vulns) == 0 {
				fmt.Printf("%s %-14s %s\n", ui.OK.Render("✓"), f.Tool,
					ui.Dim.Render(f.Version+" — no known vulnerabilities"))
				continue
			}
			vulnerable++
			if f.TopSeverity() >= audit.SevHigh {
				highOrWorse++
			}
			fmt.Printf("%s %-14s %s\n", sevStyle(f.TopSeverity()).Render("⚠"), f.Tool,
				ui.Dim.Render(f.Version))
			for _, v := range f.Vulns {
				fix := ""
				if v.FixedIn != "" {
					fix = ui.Dim.Render("  → fixed in " + v.FixedIn)
				}
				id := ui.Hyperlink(vulnURL(v.ID), v.ID)
				summary := truncate(v.Summary, 90-len(v.ID))
				fmt.Printf("    %s %s %s%s\n",
					sevStyle(v.Severity).Render(fmt.Sprintf("[%s]", v.Severity)),
					id, summary, fix)
			}
		}

		fmt.Println()
		if vulnerable == 0 {
			fmt.Println(ui.OK.Render("All audited tools are free of known vulnerabilities."))
			return nil
		}
		fmt.Printf("%s in %d tool(s). Run `opsforge upgrade` or update the affected tools.\n",
			ui.SevHigh.Render("Found vulnerabilities"), vulnerable)
		// Non-zero exit on HIGH/CRITICAL so `opsforge audit` can gate CI.
		if highOrWorse > 0 {
			return fmt.Errorf("%d tool(s) with HIGH or CRITICAL vulnerabilities", highOrWorse)
		}
		return nil
	},
}

func sevStyle(s audit.Severity) lipgloss.Style {
	switch s {
	case audit.SevCritical:
		return ui.SevCritical
	case audit.SevHigh:
		return ui.SevHigh
	case audit.SevMedium:
		return ui.SevMedium
	default:
		return ui.SevLow
	}
}

func truncate(s string, n int) string {
	if n < 1 {
		n = 1
	}
	if len(s) > n {
		return s[:n-1] + "…"
	}
	return s
}

// vulnURL returns the canonical advisory page for a vuln id: the NVD
// page for CVEs, the OSV.dev page otherwise (GHSA, GO-…).
func vulnURL(id string) string {
	if strings.HasPrefix(id, "CVE-") {
		return "https://nvd.nist.gov/vuln/detail/" + id
	}
	return "https://osv.dev/vulnerability/" + id
}

// runSecretsScan scans the workstation for leaked credentials and prints
// masked findings grouped by file.
func runSecretsScan() error {
	fmt.Println("Scanning your workstation for leaked secrets…")
	fmt.Println(ui.Dim.Render("  (shell history, shell rc files, local .env files — values are masked)"))
	fmt.Println()

	findings := secrets.ScanWorkstation()
	if len(findings) == 0 {
		fmt.Println(ui.OK.Render("✓ No leaked credentials found."))
		return nil
	}

	// Group by source file for a readable report.
	bySource := map[string][]secrets.Finding{}
	var order []string
	for _, f := range findings {
		if _, seen := bySource[f.Source]; !seen {
			order = append(order, f.Source)
		}
		bySource[f.Source] = append(bySource[f.Source], f)
	}

	critical := 0
	for _, src := range order {
		fmt.Printf("%s\n", ui.SevHigh.Render(src))
		for _, f := range bySource[src] {
			style := ui.SevMedium
			if f.Rule.Severity == secrets.SevCritical {
				style = ui.SevCritical
				critical++
			}
			fmt.Printf("  %s line %-6d %s  %s\n",
				style.Render(fmt.Sprintf("[%s]", f.Rule.Severity)),
				f.Line, f.Rule.Desc, ui.Dim.Render(f.Excerpt))
		}
	}

	fmt.Println()
	fmt.Printf("%s in %d location(s).\n",
		ui.SevHigh.Render(fmt.Sprintf("Found %d potential leak(s)", len(findings))), len(order))
	fmt.Println(ui.Dim.Render(`  Clean up: rotate any real credentials, then remove the lines
  (history: edit ~/.zsh_history · prefer 'read -s' or a secrets manager next time)`))
	if critical > 0 {
		return fmt.Errorf("%d critical secret leak(s) found", critical)
	}
	return nil
}

func init() {
	auditCmd.Flags().BoolVar(&auditSecrets, "secrets", false,
		"also scan shell history, rc files and local .env files for leaked credentials")
	rootCmd.AddCommand(auditCmd)
}
