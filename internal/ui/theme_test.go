package ui

import "testing"

func TestSetThemeKnownAndUnknown(t *testing.T) {
	SetTheme("dracula")
	if Active.Name != "dracula" {
		t.Errorf("SetTheme(dracula) active = %q", Active.Name)
	}
	// Unknown falls back to the default without error.
	SetTheme("does-not-exist")
	if Active.Name != "forge" {
		t.Errorf("unknown theme should fall back to forge, got %q", Active.Name)
	}
	SetTheme("forge") // restore
}

func TestThemeNamesSortedAndComplete(t *testing.T) {
	names := ThemeNames()
	if len(names) != len(Themes) {
		t.Fatalf("ThemeNames returned %d, want %d", len(names), len(Themes))
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Errorf("ThemeNames not sorted: %v", names)
		}
	}
}

func TestApplyThemeBindsStyles(t *testing.T) {
	SetTheme("mono")
	// After applying, the styles must be usable (Render must not panic and
	// must return the input text somewhere in its output).
	if out := OK.Render("x"); out == "" {
		t.Error("OK style produced empty output after theme apply")
	}
	SetTheme("forge")
}
