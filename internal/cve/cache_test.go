package cve

import (
	"testing"
	"time"

	"github.com/Mrg77/opsforge/internal/audit"
)

func TestStale(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	var empty Summary
	if !empty.Stale(DefaultTTL, now) {
		t.Error("never-scanned summary should be stale")
	}
	fresh := Summary{ScannedAt: now.Add(-1 * time.Hour)}
	if fresh.Stale(DefaultTTL, now) {
		t.Error("1h-old summary should be fresh with a 6h TTL")
	}
	old := Summary{ScannedAt: now.Add(-7 * time.Hour)}
	if !old.Stale(DefaultTTL, now) {
		t.Error("7h-old summary should be stale")
	}
}

func TestSummarize(t *testing.T) {
	now := time.Now().UTC()
	findings := []audit.Finding{
		{Tool: "argocd", Vulns: []audit.Vuln{{ID: "CVE-1", Severity: audit.SevCritical}}},
		{Tool: "helm", Vulns: nil}, // clean → not counted
		{Tool: "kubectl", Vulns: []audit.Vuln{{ID: "CVE-2", Severity: audit.SevMedium}}},
	}
	s := Summarize(findings, now)
	if s.Vulnerable != 2 {
		t.Errorf("Vulnerable = %d, want 2", s.Vulnerable)
	}
	if s.HighOrCritical != 1 { // only argocd
		t.Errorf("HighOrCritical = %d, want 1", s.HighOrCritical)
	}
	if len(s.Tools) != 2 {
		t.Errorf("Tools = %d, want 2", len(s.Tools))
	}
}
