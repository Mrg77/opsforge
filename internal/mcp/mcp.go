// Package mcp builds the read-only payloads that opsforge exposes to AI
// agents through the Model Context Protocol (see cmd/mcp.go for the stdio
// server wiring). The functions here are pure: they take data opsforge
// already computes (catalog, detection statuses, audit findings, guard
// policy decisions) and shape it into JSON-serializable structs. Keeping
// the shaping out of the I/O layer makes it unit-testable without a live
// MCP client or a network round-trip.
//
// SECURITY: every payload here is derived from READ-ONLY sources. Nothing
// in this package installs, upgrades, or mutates the machine — that is a
// deliberate boundary so an agent driving opsforge over MCP can inspect
// the workstation but never change it. See cmd/mcp.go for the rationale.
package mcp

import (
	"sort"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/shellcfg"
)

// InstalledTool is one entry of the list_installed_tools result.
type InstalledTool struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Version  string `json:"version,omitempty"`
	Outdated bool   `json:"outdated"`
}

// InstalledToolsResult is the list_installed_tools payload.
type InstalledToolsResult struct {
	Count int             `json:"count"`
	Tools []InstalledTool `json:"tools"`
}

// BuildInstalledTools shapes the catalog + detection statuses into the
// list_installed_tools payload, keeping only installed tools and sorting
// them by category then name for a stable, agent-friendly order.
func BuildInstalledTools(cat *catalog.Catalog, statuses map[string]detect.Status) InstalledToolsResult {
	tools := []InstalledTool{}
	for _, c := range cat.Categories {
		for _, t := range c.Tools {
			s := statuses[t.Name]
			if !s.Installed {
				continue
			}
			tools = append(tools, InstalledTool{
				Name:     t.Name,
				Category: c.Name,
				Version:  s.Version,
				Outdated: s.Outdated,
			})
		}
	}
	sort.Slice(tools, func(a, b int) bool {
		if tools[a].Category != tools[b].Category {
			return tools[a].Category < tools[b].Category
		}
		return tools[a].Name < tools[b].Name
	})
	return InstalledToolsResult{Count: len(tools), Tools: tools}
}

// VulnEntry is one CVE affecting a tool in the audit result.
type VulnEntry struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Summary  string `json:"summary,omitempty"`
	FixedIn  string `json:"fixed_in,omitempty"`
}

// ToolFinding is the vulnerability status of a single tool.
type ToolFinding struct {
	Tool        string      `json:"tool"`
	Version     string      `json:"version"`
	TopSeverity string      `json:"top_severity"`
	Vulnerable  bool        `json:"vulnerable"`
	Vulns       []VulnEntry `json:"vulnerabilities"`
}

// AuditResult is the audit_vulnerabilities payload.
type AuditResult struct {
	ToolsScanned   int           `json:"tools_scanned"`
	HighOrCritical int           `json:"high_or_critical"`
	Findings       []ToolFinding `json:"findings"`
}

// BuildAudit shapes the raw audit findings into the audit_vulnerabilities
// payload, sorted most-severe-first then by tool name — the same ordering
// the `opsforge audit` command uses.
func BuildAudit(findings []audit.Finding) AuditResult {
	sort.Slice(findings, func(a, b int) bool {
		if findings[a].TopSeverity() != findings[b].TopSeverity() {
			return findings[a].TopSeverity() > findings[b].TopSeverity()
		}
		return findings[a].Tool < findings[b].Tool
	})

	out := AuditResult{ToolsScanned: len(findings), Findings: []ToolFinding{}}
	for _, f := range findings {
		vulns := make([]VulnEntry, 0, len(f.Vulns))
		for _, v := range f.Vulns {
			vulns = append(vulns, VulnEntry{
				ID:       v.ID,
				Severity: v.Severity.String(),
				Summary:  v.Summary,
				FixedIn:  v.FixedIn,
			})
		}
		if f.TopSeverity() >= audit.SevHigh {
			out.HighOrCritical++
		}
		out.Findings = append(out.Findings, ToolFinding{
			Tool:        f.Tool,
			Version:     f.Version,
			TopSeverity: f.TopSeverity().String(),
			Vulnerable:  len(f.Vulns) > 0,
			Vulns:       vulns,
		})
	}
	return out
}

// StatusResult is the workstation_status payload: the same one-glance
// numbers `opsforge status` shows, minus anything that needs the network.
type StatusResult struct {
	ToolsInstalled  int    `json:"tools_installed"`
	ToolsTotal      int    `json:"tools_total"`
	UpdatesPending  int    `json:"updates_pending"`
	VulnerableTools int    `json:"vulnerable_tools"`
	HighOrCritical  int    `json:"high_or_critical_cves"`
	ShellLayer      bool   `json:"shell_layer_active"`
	ActiveContext   string `json:"active_context,omitempty"`
}

// BuildStatus summarizes detection statuses and (optional) audit findings
// into the workstation_status payload. Pass nil findings to skip the CVE
// counts (e.g. when the caller wants an instant, network-free summary).
func BuildStatus(statuses map[string]detect.Status, total int, findings []audit.Finding, shellOn bool, activeContext string) StatusResult {
	installed, outdated := 0, 0
	for _, s := range statuses {
		if s.Installed {
			installed++
		}
		if s.Outdated {
			outdated++
		}
	}
	vulnerable, highOrWorse := 0, 0
	for _, f := range findings {
		if len(f.Vulns) > 0 {
			vulnerable++
		}
		if f.TopSeverity() >= audit.SevHigh {
			highOrWorse++
		}
	}
	return StatusResult{
		ToolsInstalled:  installed,
		ToolsTotal:      total,
		UpdatesPending:  outdated,
		VulnerableTools: vulnerable,
		HighOrCritical:  highOrWorse,
		ShellLayer:      shellOn,
		ActiveContext:   activeContext,
	}
}

// GuardResult is the check_guard_policy payload: the decision the shell
// guard engine would reach for a command in a context.
type GuardResult struct {
	Command     string `json:"command"`
	Context     string `json:"context"`
	Action      string `json:"action"` // allow | warn | confirm | deny
	MatchedRule string `json:"matched_rule,omitempty"`
	Message     string `json:"message,omitempty"`
}

// BuildGuard evaluates a command/context against the loaded guard policy
// and shapes the decision into the check_guard_policy payload. Evaluating
// a rule is passive — it never invokes kubectl/gcloud (see shellcfg).
func BuildGuard(policy *shellcfg.GuardPolicy, command, context string) GuardResult {
	d := policy.Evaluate(command, context)
	return GuardResult{
		Command:     command,
		Context:     context,
		Action:      string(d.Action),
		MatchedRule: d.Rule,
		Message:     d.Message,
	}
}
