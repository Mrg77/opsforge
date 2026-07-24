package cmd

import (
	"testing"

	"github.com/Mrg77/opsforge/internal/catalog"
)

// These tests cover the pure helpers behind the CLI commands. The full
// RunE closures are exercised through their extracted functions here; the
// package had no tests before, despite exit codes being sold as a CI gate.

func TestMatchesSearch(t *testing.T) {
	tool := catalog.Tool{Name: "doggo", Description: "Modern DNS client"}
	cases := []struct {
		term string
		want bool
	}{
		{"", true},           // empty term matches everything
		{"dns", true},        // in description
		{"DNS", true},        // case-insensitive
		{"dog", true},        // in name
		{"Networking", true}, // in category (passed separately)
		{"terraform", false}, // unrelated
	}
	for _, c := range cases {
		if got := matchesSearch(tool, "Networking & HTTP", c.term); got != c.want {
			t.Errorf("matchesSearch(term=%q) = %v, want %v", c.term, got, c.want)
		}
	}
}

func TestResolveProfileFromCatalog(t *testing.T) {
	cat := &catalog.Catalog{
		Categories: []catalog.Category{{Name: "K", Tools: []catalog.Tool{
			{Name: "kubectl", Bin: "kubectl", Brew: "kubernetes-cli"},
		}}},
		Profiles: []catalog.Profile{{Name: "k8s", Tools: []string{"kubectl"}}},
	}
	if p, ok := resolveProfile(cat, "k8s"); !ok || len(p.Tools) != 1 {
		t.Errorf("resolveProfile(k8s) = %+v, %v", p, ok)
	}
	if _, ok := resolveProfile(cat, "ghost"); ok {
		t.Error("resolveProfile(ghost) should be false")
	}
}

func TestCollectOSVTargetsOnlyMappedInstalled(t *testing.T) {
	// A tool with no OSV mapping is skipped even if installed; the function
	// is what feeds audit/sbom, so it must only surface auditable tools.
	cat := &catalog.Catalog{
		Categories: []catalog.Category{{Name: "K", Tools: []catalog.Tool{
			{Name: "no-osv", Bin: "definitely-not-installed-xyz", Brew: "x"},
		}}},
	}
	got := CollectOSVTargets(cat)
	for _, tg := range got {
		if tg.Name == "no-osv" {
			t.Errorf("tool without OSV mapping leaked into targets: %+v", tg)
		}
	}
}
