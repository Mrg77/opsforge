package sbom

import (
	"encoding/json"
	"testing"

	"github.com/Mrg77/opsforge/internal/audit"
)

func TestBuildComponentsAndPURL(t *testing.T) {
	in := []Input{
		{Name: "helm", Version: "v4.2.3+g43e8b7f", Description: "k8s pkg mgr",
			Ecosystem: "Go", Package: "helm.sh/helm/v3"},
		{Name: "jq", Version: "jq-1.8.2", Description: "json"}, // no OSV → no purl
	}
	doc := Build(in, "2026-01-01T00:00:00Z")

	if doc.BOMFormat != "CycloneDX" || doc.SpecVersion != "1.6" {
		t.Fatalf("wrong header: %s %s", doc.BOMFormat, doc.SpecVersion)
	}
	if len(doc.Components) != 2 {
		t.Fatalf("want 2 components, got %d", len(doc.Components))
	}
	helm := doc.Components[0]
	if helm.Version != "4.2.3" { // normalized
		t.Errorf("helm version = %q, want 4.2.3", helm.Version)
	}
	if helm.PURL != "pkg:golang/helm.sh/helm/v3@4.2.3" {
		t.Errorf("helm purl = %q", helm.PURL)
	}
	if doc.Components[1].PURL != "" {
		t.Errorf("jq should have no purl, got %q", doc.Components[1].PURL)
	}
}

func TestBuildEmbedsVulnerabilities(t *testing.T) {
	in := []Input{{
		Name: "argocd", Version: "v2.11.0", Ecosystem: "Go", Package: "github.com/argoproj/argo-cd/v2",
		Vulns: []audit.Vuln{
			{ID: "CVE-2025-47933", Summary: "XSS", Severity: audit.SevCritical, FixedIn: "2.13.8"},
			{ID: "GHSA-xxxx", Summary: "panic", Severity: audit.SevHigh},
		},
	}}
	doc := Build(in, "2026-01-01T00:00:00Z")
	if len(doc.Vulnerabilities) != 2 {
		t.Fatalf("want 2 vulns, got %d", len(doc.Vulnerabilities))
	}
	v := doc.Vulnerabilities[0]
	if v.ID != "CVE-2025-47933" || len(v.Affects) != 1 || v.Affects[0].Ref != "tool:argocd" {
		t.Errorf("vuln affects wrong: %+v", v)
	}
	if v.Ratings[0].Severity != "critical" {
		t.Errorf("severity = %q, want critical", v.Ratings[0].Severity)
	}
	if v.Source.Name != "NVD" { // CVE- → NVD
		t.Errorf("source = %q, want NVD", v.Source.Name)
	}
	if doc.Vulnerabilities[1].Source.Name != "OSV" { // GHSA → OSV
		t.Errorf("GHSA source = %q, want OSV", doc.Vulnerabilities[1].Source.Name)
	}
}

func TestBuildMarshalsToValidJSON(t *testing.T) {
	doc := Build([]Input{{Name: "fzf", Version: "0.65.0"}}, "2026-01-01T00:00:00Z")
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]any
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}
	if back["bomFormat"] != "CycloneDX" {
		t.Errorf("bomFormat lost in marshal")
	}
}

func TestPURLEcosystemMapping(t *testing.T) {
	cases := map[string]string{"Go": "golang", "PyPI": "pypi", "crates.io": "cargo", "npm": "npm"}
	for osv, want := range cases {
		if got := purlEcosystem(osv); got != want {
			t.Errorf("purlEcosystem(%q) = %q, want %q", osv, got, want)
		}
	}
}
