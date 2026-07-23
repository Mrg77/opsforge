package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

// baseCatalog is a tiny embedded-like catalog for overlay tests.
func baseCatalog() *Catalog {
	return &Catalog{
		Categories: []Category{
			{Name: "Kubernetes", Tools: []Tool{
				{Name: "kubectl", Bin: "kubectl", Brew: "kubernetes-cli", Description: "k8s CLI"},
			}},
		},
		Profiles: []Profile{{Name: "k8s", Tools: []string{"kubectl"}}},
	}
}

func writeOverlay(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "overlay.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOverlayAppendsNewToolAndCategory(t *testing.T) {
	c := baseCatalog()
	p := writeOverlay(t, `
categories:
  - name: Internal
    tools:
      - name: acme-cli
        bin: acme
        brew: acmecorp/tap/acme-cli
        description: internal deploy CLI
`)
	if err := c.MergeOverlays([]string{p}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.validated(); err != nil {
		t.Fatalf("merged catalog invalid: %v", err)
	}
	if tool, ok := c.Tool("acme-cli"); !ok || tool.Bin != "acme" {
		t.Fatalf("acme-cli not merged: %+v %v", tool, ok)
	}
}

func TestOverlayOverridesExistingTool(t *testing.T) {
	c := baseCatalog()
	p := writeOverlay(t, `
categories:
  - name: Kubernetes
    tools:
      - name: kubectl
        bin: kubectl
        brew: my-tap/kubectl
        description: pinned internal build
`)
	if err := c.MergeOverlays([]string{p}); err != nil {
		t.Fatal(err)
	}
	tool, _ := c.Tool("kubectl")
	if tool.Brew != "my-tap/kubectl" {
		t.Fatalf("override failed, brew = %q", tool.Brew)
	}
	// Still exactly one kubectl (replaced, not duplicated).
	n := 0
	for _, tl := range c.Tools() {
		if tl.Name == "kubectl" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("expected 1 kubectl after override, got %d", n)
	}
}

func TestOverlayAddsAndOverridesProfiles(t *testing.T) {
	c := baseCatalog()
	p := writeOverlay(t, `
categories:
  - name: Internal
    tools:
      - name: acme
        bin: acme
        brew: acme
        description: x
profiles:
  - name: mine
    tools: [acme, kubectl]
  - name: k8s
    tools: [kubectl, acme]
`)
	if err := c.MergeOverlays([]string{p}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.validated(); err != nil {
		t.Fatalf("invalid: %v", err)
	}
	if p, ok := c.Profile("mine"); !ok || len(p.Tools) != 2 {
		t.Errorf("profile 'mine' not added: %+v %v", p, ok)
	}
	if p, _ := c.Profile("k8s"); len(p.Tools) != 2 {
		t.Errorf("profile 'k8s' not overridden: %+v", p)
	}
}

func TestOverlayRejectsUnknownFields(t *testing.T) {
	c := baseCatalog()
	p := writeOverlay(t, `
categories:
  - name: Internal
    tools:
      - name: acme
        bin: acme
        brew: acme
        descriptionn: typo here
`)
	if err := c.MergeOverlays([]string{p}); err == nil {
		t.Fatal("expected error on unknown field, got nil")
	}
}

func TestOverlayMissingFileIsSkipped(t *testing.T) {
	c := baseCatalog()
	if err := c.MergeOverlays([]string{"/no/such/overlay.yaml"}); err != nil {
		t.Fatalf("missing overlay should be skipped, got %v", err)
	}
}
