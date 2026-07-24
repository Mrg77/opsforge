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
