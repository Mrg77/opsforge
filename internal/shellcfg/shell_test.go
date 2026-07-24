package shellcfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseShell(t *testing.T) {
	for _, name := range []string{"zsh", "fish"} {
		if sh, err := ParseShell(name); err != nil || string(sh) != name {
			t.Errorf("ParseShell(%q) = (%v, %v)", name, sh, err)
		}
	}
	if _, err := ParseShell("bash"); err == nil {
		t.Error("ParseShell(bash) should error — bash isn't supported yet")
	}
}

func TestDetectShell(t *testing.T) {
	t.Setenv("SHELL", "/opt/homebrew/bin/fish")
	if DetectShell() != Fish {
		t.Error("a fish $SHELL should detect Fish")
	}
	t.Setenv("SHELL", "/bin/zsh")
	if DetectShell() != Zsh {
		t.Error("a zsh $SHELL should detect Zsh")
	}
	t.Setenv("SHELL", "")
	if DetectShell() != Zsh {
		t.Error("an unknown $SHELL should default to Zsh")
	}
}

func TestFishEnvSnippetSourcesModules(t *testing.T) {
	snippet, err := Fish.EnvFor()
	if err != nil {
		t.Fatal(err)
	}
	// It must use fish's `source` idiom and reference the guards module.
	if !strings.Contains(snippet, "source") || !strings.Contains(snippet, "guards.fish") {
		t.Errorf("fish env snippet missing source/guards:\n%s", snippet)
	}
	// The zsh compinit wiring must NOT leak into the fish snippet.
	if strings.Contains(snippet, "compinit") || strings.Contains(snippet, "fpath") {
		t.Errorf("fish env snippet contains zsh-only wiring:\n%s", snippet)
	}
}

func TestFishInstallRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := Fish.InstallTo()
	if err != nil {
		t.Fatalf("InstallTo(fish): %v", err)
	}
	// config.fish is created with the opsforge block.
	if filepath.Base(path) != "config.fish" {
		t.Errorf("fish rc path = %q, want …/config.fish", path)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), markerStart) || !strings.Contains(string(data), "opsforge shell env --shell fish") {
		t.Errorf("config.fish missing opsforge block:\n%s", data)
	}
	if !Fish.InstalledIn() {
		t.Error("InstalledIn(fish) should be true after install")
	}

	// Modules were materialized to the fish config dir.
	dir, _ := Fish.configDir()
	if _, err := os.Stat(filepath.Join(dir, "guards.fish")); err != nil {
		t.Errorf("guards.fish not written: %v", err)
	}

	// Uninstall removes the block and the modules.
	if _, err := Fish.UninstallFrom(); err != nil {
		t.Fatalf("UninstallFrom(fish): %v", err)
	}
	if Fish.InstalledIn() {
		t.Error("InstalledIn(fish) should be false after uninstall")
	}
	data, _ = os.ReadFile(path)
	if strings.Contains(string(data), markerStart) {
		t.Error("opsforge block should be gone from config.fish after uninstall")
	}
}

// TestZshUnaffectedByFishSupport is a guard that the fish work didn't change
// the zsh paths: the zsh config dir and rc must be the historical ones.
func TestZshUnaffectedByFishSupport(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir, _ := Zsh.configDir()
	if filepath.Base(dir) != "shell" {
		t.Errorf("zsh config dir moved: %q (want …/opsforge/shell)", dir)
	}
	rc, _ := Zsh.RcPath()
	if filepath.Base(rc) != ".zshrc" {
		t.Errorf("zsh rc path changed: %q", rc)
	}
}
