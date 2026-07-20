package catalog

import (
	"strings"
	"testing"
)

func TestLoadValidCatalog(t *testing.T) {
	cat, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if len(cat.Categories) == 0 {
		t.Fatal("catalog has no categories")
	}
	if len(cat.Tools()) < 20 {
		t.Errorf("catalog unexpectedly small: %d tools", len(cat.Tools()))
	}
	if len(cat.Profiles) == 0 {
		t.Fatal("catalog has no profiles")
	}
}

func TestProfileAndToolLookups(t *testing.T) {
	cat, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	p, ok := cat.Profile("k8s")
	if !ok {
		t.Fatal("profile k8s not found")
	}
	for _, name := range p.Tools {
		if _, ok := cat.Tool(name); !ok {
			t.Errorf("profile k8s references tool %q not resolvable via Tool()", name)
		}
	}
	if _, ok := cat.Profile("does-not-exist"); ok {
		t.Error("Profile() returned ok for an unknown profile")
	}
	if _, ok := cat.Tool("does-not-exist"); ok {
		t.Error("Tool() returned ok for an unknown tool")
	}
}

func TestCompletionCommandsUseOwnBinary(t *testing.T) {
	cat, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range cat.Tools() {
		if len(tool.CompletionZsh) == 0 {
			continue
		}
		if tool.CompletionZsh[0] != tool.Bin {
			t.Errorf("tool %q: completion command %q does not start with its bin %q",
				tool.Name, tool.CompletionZsh[0], tool.Bin)
		}
		if !strings.Contains(strings.Join(tool.CompletionZsh, " "), "zsh") {
			t.Errorf("tool %q: completion command does not mention zsh", tool.Name)
		}
	}
}
