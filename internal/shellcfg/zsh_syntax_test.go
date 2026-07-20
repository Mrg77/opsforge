package shellcfg

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestModulesAreValidZsh runs `zsh -n` (parse only, no execution) on
// every embedded module and on the full env snippet. A syntax error here
// would break the user's shell at startup, so this must never regress.
// Skipped when zsh is unavailable (unlikely on macOS/Linux CI).
func TestModulesAreValidZsh(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available; skipping shell syntax check")
	}
	mods, err := Modules()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, m := range mods {
		path := filepath.Join(dir, m.Name+".zsh")
		if err := os.WriteFile(path, []byte(m.Body), 0o644); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command("zsh", "-n", path).CombinedOutput(); err != nil {
			t.Errorf("module %s has invalid zsh syntax:\n%s", m.Name, out)
		}
	}

	// The eval'd env snippet must also parse.
	env, err := Env()
	if err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(dir, "env.zsh")
	os.WriteFile(envPath, []byte(env), 0o644)
	if out, err := exec.Command("zsh", "-n", envPath).CombinedOutput(); err != nil {
		t.Errorf("env snippet has invalid zsh syntax:\n%s", out)
	}
}
