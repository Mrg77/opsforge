package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
)

func write(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, FileName)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func testCatalog() *catalog.Catalog {
	return &catalog.Catalog{
		Categories: []catalog.Category{{Name: "K", Tools: []catalog.Tool{
			{Name: "kubectl", Bin: "kubectl", Brew: "kubernetes-cli"},
			{Name: "helm", Bin: "helm", Brew: "helm"},
			{Name: "jq", Bin: "jq", Brew: "jq"},
		}}},
		Profiles: []catalog.Profile{{Name: "core", Tools: []string{"jq", "helm"}}},
	}
}

func TestLoadValidatesVersionAndFailOn(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, "version: 1\ntools: [kubectl]\nfail_on: high\n")
	if _, err := Load(p); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
	bad := write(t, t.TempDir(), "version: 2\ntools: [kubectl]\n")
	if _, err := Load(bad); err == nil {
		t.Error("expected error on version 2")
	}
	badfail := write(t, t.TempDir(), "version: 1\nfail_on: medium\n")
	if _, err := Load(badfail); err == nil {
		t.Error("expected error on fail_on: medium")
	}
	typo := write(t, t.TempDir(), "version: 1\ntoolss: [kubectl]\n")
	if _, err := Load(typo); err == nil {
		t.Error("expected error on unknown field")
	}
}

func TestResolveToolsExpandsProfilesAndDedupes(t *testing.T) {
	cat := testCatalog()
	p := &Project{Version: 1, Tools: []string{"kubectl", "helm"}, Profiles: []string{"core"}}
	tools, unknown := p.ResolveTools(cat)
	// core = {jq, helm}; tools = {kubectl, helm}; union deduped = jq,helm,kubectl
	if len(tools) != 3 {
		t.Fatalf("want 3 distinct tools, got %v", tools)
	}
	if len(unknown) != 0 {
		t.Errorf("unexpected unknown: %v", unknown)
	}
}

func TestResolveToolsFlagsUnknown(t *testing.T) {
	cat := testCatalog()
	p := &Project{Version: 1, Tools: []string{"nope"}, Profiles: []string{"ghost"}}
	tools, unknown := p.ResolveTools(cat)
	if len(tools) != 0 {
		t.Errorf("expected no known tools, got %v", tools)
	}
	if len(unknown) != 2 { // nope + profile:ghost
		t.Errorf("expected 2 unknown, got %v", unknown)
	}
}

func TestBuildPlanSplitsMissingAndPresent(t *testing.T) {
	cat := testCatalog()
	p := &Project{Version: 1, Tools: []string{"kubectl", "helm", "jq"}}
	statuses := map[string]detect.Status{
		"kubectl": {Installed: true},
		"helm":    {Installed: false},
		"jq":      {Installed: true},
	}
	plan := BuildPlan(p, cat, statuses)
	if len(plan.Install) != 1 || plan.Install[0] != "helm" {
		t.Errorf("Install = %v, want [helm]", plan.Install)
	}
	if len(plan.Present) != 2 {
		t.Errorf("Present = %v, want 2", plan.Present)
	}
}

func TestFindWalksUp(t *testing.T) {
	root := t.TempDir()
	write(t, root, "version: 1\n")
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, ok := Find(sub)
	if !ok || got != filepath.Join(root, FileName) {
		t.Errorf("Find from subdir = %q, %v", got, ok)
	}
	if _, ok := Find(t.TempDir()); ok {
		t.Error("Find in empty dir should be false")
	}
}
