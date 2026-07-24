package vex

import (
	"testing"

	"github.com/Mrg77/opsforge/internal/audit"
)

func TestBuildStatements(t *testing.T) {
	in := []Input{
		{PURL: "pkg:pypi/ansible@2.19.3", Vulns: []audit.Vuln{
			{ID: "CVE-2021-3533", Severity: audit.SevMedium, FixedIn: "3.0.0"},
			{ID: "CVE-2020-0000", Severity: audit.SevHigh}, // no fix
		}},
		{PURL: "", Vulns: []audit.Vuln{{ID: "CVE-X"}}}, // no coords → skipped
	}
	doc := Build(in, "id-1", "2026-01-01T00:00:00Z")
	if doc.Context != contextURL || doc.Author != "opsforge" {
		t.Fatalf("bad header: %+v", doc)
	}
	if len(doc.Statements) != 2 { // the no-PURL input is skipped
		t.Fatalf("want 2 statements, got %d", len(doc.Statements))
	}
	// Deterministic order (by CVE id).
	if doc.Statements[0].Vulnerability.Name != "CVE-2020-0000" {
		t.Errorf("statements not sorted: %+v", doc.Statements)
	}
	for _, s := range doc.Statements {
		if s.Status != StatusAffected {
			t.Errorf("want affected, got %s", s.Status)
		}
		if s.ActionStatement == "" {
			t.Errorf("affected statement needs an action_statement")
		}
	}
}

func TestBuildActionStatements(t *testing.T) {
	doc := Build([]Input{
		{PURL: "pkg:golang/x@1.0.0", Vulns: []audit.Vuln{
			{ID: "CVE-A", FixedIn: "1.2.0"}, // fix known → "Upgrade to…"
			{ID: "CVE-B"},                   // no fix → "monitor the advisory"
		}},
	}, "id", "2026-01-01T00:00:00Z")

	byID := map[string]Statement{}
	for _, s := range doc.Statements {
		byID[s.Vulnerability.Name] = s
	}
	if got := byID["CVE-A"].ActionStatement; got != "Upgrade to 1.2.0 or later." {
		t.Errorf("CVE-A action = %q", got)
	}
	if got := byID["CVE-B"].ActionStatement; got == "" || got == "Upgrade to  or later." {
		t.Errorf("CVE-B (no fix) should get a monitor action, got %q", got)
	}
}

func TestSummaryCountsDistinctProducts(t *testing.T) {
	doc := Build([]Input{
		{PURL: "pkg:golang/x@1.0.0", Vulns: []audit.Vuln{{ID: "CVE-1"}, {ID: "CVE-2"}}},
		{PURL: "pkg:golang/y@2.0.0", Vulns: []audit.Vuln{{ID: "CVE-3"}}},
	}, "id", "2026-01-01T00:00:00Z")

	// 3 statements across 2 distinct products.
	if n := countProducts(doc); n != 2 {
		t.Errorf("countProducts = %d, want 2 distinct products", n)
	}
	if s := doc.Summary(); s == "" {
		t.Error("Summary should be non-empty")
	}
}

func TestBuildEmptyInputYieldsNoStatements(t *testing.T) {
	doc := Build(nil, "id", "2026-01-01T00:00:00Z")
	if len(doc.Statements) != 0 {
		t.Errorf("empty input should produce no statements, got %d", len(doc.Statements))
	}
	if doc.Author != "opsforge" || doc.Version != docVersion {
		t.Errorf("header should still be populated: %+v", doc)
	}
}

func TestKEVSet(t *testing.T) {
	k := KEVSet{"CVE-2025-1234": true}
	if !k.Has("cve-2025-1234") { // case-insensitive
		t.Error("KEV lookup should be case-insensitive")
	}
	if k.Has("CVE-9999-0000") {
		t.Error("unknown CVE should not be in KEV")
	}
	var nilSet KEVSet
	if nilSet.Has("CVE-1") {
		t.Error("nil KEV set should return false")
	}
}
