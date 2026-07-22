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
	"github.com/Mrg77/opsforge/internal/secrets"
	"github.com/Mrg77/opsforge/internal/ui"
)

var (
	sevCritical = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	sevHigh     = lipgloss.NewStyle().Foreground(lipgloss.Color("202"))
	sevMedium   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	sevLow      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	auditOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	auditDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
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
				fmt.Printf("%s %-14s %s\n", auditOK.Render("✓"), f.Tool,
					auditDim.Render(f.Version+" — no known vulnerabilities"))
				continue
			}
			vulnerable++
			if f.TopSeverity() >= audit.SevHigh {
				highOrWorse++
			}
			fmt.Printf("%s %-14s %s\n", sevStyle(f.TopSeverity()).Render("⚠"), f.Tool,
				auditDim.Render(f.Version))
			for _, v := range f.Vulns {
				fix := ""
				if v.FixedIn != "" {
					fix = auditDim.Render("  → fixed in " + v.FixedIn)
				}
				id := hyperlink(vulnURL(v.ID), v.ID)
				summary := truncate(v.Summary, 90-len(v.ID))
				fmt.Printf("    %s %s %s%s\n",
					sevStyle(v.Severity).Render(fmt.Sprintf("[%s]", v.Severity)),
					id, summary, fix)
			}
		}

		fmt.Println()
		if vulnerable == 0 {
			fmt.Println(auditOK.Render("All audited tools are free of known vulnerabilities."))
			return nil
		}
		fmt.Printf("%s in %d tool(s). Run `opsforge upgrade` or update the affected tools.\n",
			sevHigh.Render("Found vulnerabilities"), vulnerable)
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
		return sevCritical
	case audit.SevHigh:
		return sevHigh
	case audit.SevMedium:
		return sevMedium
	default:
		return sevLow
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

// hyperlink wraps text in an OSC 8 terminal hyperlink escape sequence.
// Terminals that support it (iTerm2, WezTerm, Ghostty, kitty, modern
// gnome-terminal) render `text` as a clickable link to `url`; others
// simply show `text` unchanged.
func hyperlink(url, text string) string {
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// runSecretsScan scans the workstation for leaked credentials and prints
// masked findings grouped by file.
func runSecretsScan() error {
	fmt.Println("Scanning your workstation for leaked secrets…")
	fmt.Println(auditDim.Render("  (shell history, shell rc files, local .env files — values are masked)"))
	fmt.Println()

	findings := secrets.ScanWorkstation()
	if len(findings) == 0 {
		fmt.Println(auditOK.Render("✓ No leaked credentials found."))
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
		fmt.Printf("%s\n", sevHigh.Render(src))
		for _, f := range bySource[src] {
			style := sevMedium
			if f.Rule.Severity == secrets.SevCritical {
				style = sevCritical
				critical++
			}
			fmt.Printf("  %s line %-6d %s  %s\n",
				style.Render(fmt.Sprintf("[%s]", f.Rule.Severity)),
				f.Line, f.Rule.Desc, auditDim.Render(f.Excerpt))
		}
	}

	fmt.Println()
	fmt.Printf("%s in %d location(s).\n",
		sevHigh.Render(fmt.Sprintf("Found %d potential leak(s)", len(findings))), len(order))
	fmt.Println(auditDim.Render(`  Clean up: rotate any real credentials, then remove the lines
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
