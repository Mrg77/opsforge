package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
)

var (
	sevCritical = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	sevHigh     = lipgloss.NewStyle().Foreground(lipgloss.Color("202"))
	sevMedium   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	sevLow      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	auditOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	auditDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Scan installed tools for known CVEs (via OSV.dev)",
	Long: `Cross-references the versions of your installed tools against the OSV.dev
vulnerability database and reports which ones have known CVEs and should be
upgraded. Only tools with an OSV mapping in the catalog are checked.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}

		// Collect installed tools that have an OSV mapping.
		type target struct {
			tool    catalog.Tool
			version string
		}
		var targets []target
		auditable := 0
		for _, t := range cat.Tools() {
			if t.OSV == nil {
				continue
			}
			auditable++
			s := detect.Tool(t)
			if !s.Installed {
				continue
			}
			ver := audit.NormalizeVersion(s.Version)
			if ver == "" {
				continue
			}
			targets = append(targets, target{tool: t, version: ver})
		}

		if len(targets) == 0 {
			fmt.Printf("No auditable tools installed (%d tools in the catalog carry an OSV mapping).\n", auditable)
			return nil
		}

		fmt.Printf("Auditing %d installed tool(s) against OSV.dev…\n\n", len(targets))

		// Query OSV concurrently.
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		findings := make([]audit.Finding, len(targets))
		var wg sync.WaitGroup
		for i, tg := range targets {
			wg.Add(1)
			go func(i int, tg target) {
				defer wg.Done()
				vulns, err := audit.Query(ctx, tg.tool.OSV.Ecosystem, tg.tool.OSV.Name, tg.version)
				f := audit.Finding{Tool: tg.tool.Name, Version: tg.version, Auditable: true}
				if err == nil {
					f.Vulns = vulns
				}
				findings[i] = f
			}(i, tg)
		}
		wg.Wait()

		// Sort: most severe first, then by tool name.
		sort.Slice(findings, func(a, b int) bool {
			if findings[a].TopSeverity() != findings[b].TopSeverity() {
				return findings[a].TopSeverity() > findings[b].TopSeverity()
			}
			return findings[a].Tool < findings[b].Tool
		})

		vulnerable := 0
		for _, f := range findings {
			if len(f.Vulns) == 0 {
				fmt.Printf("%s %-14s %s\n", auditOK.Render("✓"), f.Tool,
					auditDim.Render(f.Version+" — no known vulnerabilities"))
				continue
			}
			vulnerable++
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
			sevHigh.Render(fmt.Sprintf("Found vulnerabilities")), vulnerable)
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

func init() {
	rootCmd.AddCommand(auditCmd)
}
