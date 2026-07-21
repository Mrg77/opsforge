package userprofiles

import (
	"testing"

	"github.com/Mrg77/opsforge/internal/catalog"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Empty on first run.
	if ps, err := Load(); err != nil || len(ps) != 0 {
		t.Fatalf("fresh Load() = %v, %v; want empty", ps, err)
	}

	if err := Save(catalog.Profile{Name: "web", Tools: []string{"kubectl", "helm"}}); err != nil {
		t.Fatal(err)
	}
	ps, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 1 || ps[0].Name != "web" || len(ps[0].Tools) != 2 {
		t.Fatalf("after save, Load() = %+v", ps)
	}
}

func TestSaveReplacesByName(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	Save(catalog.Profile{Name: "web", Tools: []string{"kubectl"}})
	Save(catalog.Profile{Name: "web", Tools: []string{"kubectl", "helm", "k9s"}})
	ps, _ := Load()
	if len(ps) != 1 {
		t.Fatalf("replace created a duplicate: %d profiles", len(ps))
	}
	if len(ps[0].Tools) != 3 {
		t.Errorf("replace did not update tools: %v", ps[0].Tools)
	}
}

func TestSaveKeepsProfilesSorted(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	Save(catalog.Profile{Name: "zeta", Tools: []string{"jq"}})
	Save(catalog.Profile{Name: "alpha", Tools: []string{"yq"}})
	ps, _ := Load()
	if ps[0].Name != "alpha" || ps[1].Name != "zeta" {
		t.Errorf("profiles not sorted: %s, %s", ps[0].Name, ps[1].Name)
	}
}

func TestDelete(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	Save(catalog.Profile{Name: "a", Tools: []string{"jq"}})
	Save(catalog.Profile{Name: "b", Tools: []string{"yq"}})
	if err := Delete("a"); err != nil {
		t.Fatal(err)
	}
	ps, _ := Load()
	if len(ps) != 1 || ps[0].Name != "b" {
		t.Errorf("after delete, Load() = %+v", ps)
	}
	// Deleting a missing profile is not an error.
	if err := Delete("nope"); err != nil {
		t.Errorf("Delete(missing) = %v, want nil", err)
	}
}

func TestSaveRejectsInvalid(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := Save(catalog.Profile{Name: "", Tools: []string{"jq"}}); err == nil {
		t.Error("Save with empty name should fail")
	}
	if err := Save(catalog.Profile{Name: "x", Tools: nil}); err == nil {
		t.Error("Save with no tools should fail")
	}
}
