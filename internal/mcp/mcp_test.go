package mcp

import (
	"testing"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/shellcfg"
)

func testCatalog() *catalog.Catalog {
	return &catalog.Catalog{
		Categories: []catalog.Category{
			{Name: "Kubernetes", Tools: []catalog.Tool{
				{Name: "kubectl", Bin: "kubectl"},
				{Name: "helm", Bin: "helm"},
			}},
			{Name: "IaC", Tools: []catalog.Tool{
				{Name: "terraform", Bin: "terraform"},
			}},
		},
	}
}

func TestBuildInstalledTools(t *testing.T) {
	cat := testCatalog()
	statuses := map[string]detect.Status{
		"kubectl":   {Installed: true, Version: "v1.29.0", Outdated: true},
		"helm":      {Installed: false},
		"terraform": {Installed: true, Version: "1.7.0"},
	}

	got := BuildInstalledTools(cat, statuses)

	if got.Count != 2 {
		t.Fatalf("Count = %d, want 2 (only installed tools)", got.Count)
	}
	if len(got.Tools) != 2 {
		t.Fatalf("len(Tools) = %d, want 2", len(got.Tools))
	}
	// Sorted by category ("IaC" < "Kubernetes") then name.
	if got.Tools[0].Name != "terraform" || got.Tools[0].Category != "IaC" {
		t.Errorf("first tool = %+v, want terraform/IaC first (category sort)", got.Tools[0])
	}
	if got.Tools[1].Name != "kubectl" || !got.Tools[1].Outdated {
		t.Errorf("second tool = %+v, want kubectl marked outdated", got.Tools[1])
	}
	// helm is not installed and must be excluded.
	for _, tl := range got.Tools {
		if tl.Name == "helm" {
			t.Error("helm is not installed but appears in the result")
		}
	}
}

func TestBuildAudit(t *testing.T) {
	findings := []audit.Finding{
		{Tool: "helm", Version: "3.14.0", Auditable: true}, // clean
		{Tool: "kubectl", Version: "1.29.0", Auditable: true, Vulns: []audit.Vuln{
			{ID: "CVE-2024-0001", Severity: audit.SevMedium, Summary: "medium issue", FixedIn: "1.29.1"},
		}},
		{Tool: "terraform", Version: "1.7.0", Auditable: true, Vulns: []audit.Vuln{
			{ID: "CVE-2024-9999", Severity: audit.SevCritical, Summary: "critical issue"},
		}},
	}

	got := BuildAudit(findings)

	if got.ToolsScanned != 3 {
		t.Errorf("ToolsScanned = %d, want 3", got.ToolsScanned)
	}
	if got.HighOrCritical != 1 {
		t.Errorf("HighOrCritical = %d, want 1", got.HighOrCritical)
	}
	// Most severe first: terraform (CRITICAL) leads.
	if got.Findings[0].Tool != "terraform" || got.Findings[0].TopSeverity != "CRITICAL" {
		t.Errorf("first finding = %+v, want terraform/CRITICAL", got.Findings[0])
	}
	if !got.Findings[0].Vulnerable || len(got.Findings[0].Vulns) != 1 {
		t.Errorf("terraform finding should carry exactly one vuln: %+v", got.Findings[0])
	}
	if got.Findings[0].Vulns[0].ID != "CVE-2024-9999" {
		t.Errorf("vuln id = %q, want CVE-2024-9999", got.Findings[0].Vulns[0].ID)
	}
	// The clean tool must still appear, marked not vulnerable.
	var helm *ToolFinding
	for i := range got.Findings {
		if got.Findings[i].Tool == "helm" {
			helm = &got.Findings[i]
		}
	}
	if helm == nil {
		t.Fatal("clean tool helm missing from findings")
	}
	if helm.Vulnerable || len(helm.Vulns) != 0 {
		t.Errorf("helm should be reported as not vulnerable, got %+v", *helm)
	}
}

func TestBuildStatus(t *testing.T) {
	statuses := map[string]detect.Status{
		"kubectl":   {Installed: true, Version: "1.29.0", Outdated: true},
		"helm":      {Installed: true, Version: "3.14.0"},
		"terraform": {Installed: false},
	}
	findings := []audit.Finding{
		{Tool: "kubectl", Vulns: []audit.Vuln{{Severity: audit.SevHigh}}},
		{Tool: "helm"}, // clean
	}

	got := BuildStatus(statuses, 3, findings, true, "prod-cluster")

	if got.ToolsInstalled != 2 {
		t.Errorf("ToolsInstalled = %d, want 2", got.ToolsInstalled)
	}
	if got.ToolsTotal != 3 {
		t.Errorf("ToolsTotal = %d, want 3", got.ToolsTotal)
	}
	if got.UpdatesPending != 1 {
		t.Errorf("UpdatesPending = %d, want 1", got.UpdatesPending)
	}
	if got.VulnerableTools != 1 || got.HighOrCritical != 1 {
		t.Errorf("vuln counts = %d/%d, want 1/1", got.VulnerableTools, got.HighOrCritical)
	}
	if !got.ShellLayer {
		t.Error("ShellLayer should be true")
	}
	if got.ActiveContext != "prod-cluster" {
		t.Errorf("ActiveContext = %q, want prod-cluster", got.ActiveContext)
	}

	// With nil findings the CVE counts stay zero (network-free summary).
	offline := BuildStatus(statuses, 3, nil, false, "")
	if offline.VulnerableTools != 0 || offline.HighOrCritical != 0 {
		t.Errorf("offline summary should carry no CVE counts, got %+v", offline)
	}
}

func TestBuildGuard(t *testing.T) {
	policy := shellcfg.DefaultPolicy()

	// A destructive kubectl on a prod context should be confirmed.
	got := BuildGuard(policy, "kubectl delete deploy api", "prod")
	if got.Action != string(shellcfg.ActionConfirm) {
		t.Errorf("action = %q, want confirm", got.Action)
	}
	if got.MatchedRule == "" {
		t.Error("expected a matched rule name for a destructive prod kubectl")
	}
	if got.Command != "kubectl delete deploy api" || got.Context != "prod" {
		t.Errorf("command/context not echoed back: %+v", got)
	}

	// A harmless command matches nothing and is allowed.
	allow := BuildGuard(policy, "kubectl get pods", "prod")
	if allow.Action != string(shellcfg.ActionAllow) {
		t.Errorf("action = %q, want allow", allow.Action)
	}
	if allow.MatchedRule != "" {
		t.Errorf("no rule should match a read-only get, got %q", allow.MatchedRule)
	}
}
