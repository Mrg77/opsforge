package shellcfg

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestModulesAreValidFish runs `fish --no-execute` (parse only) on every
// embedded fish module and on the fish env snippet. A syntax error here would
// break a fish user's shell at startup. Skipped when fish is unavailable.
func TestModulesAreValidFish(t *testing.T) {
	if _, err := exec.LookPath("fish"); err != nil {
		t.Skip("fish not available; skipping fish syntax check")
	}
	mods, err := Fish.modulesFor()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, m := range mods {
		path := filepath.Join(dir, m.Name+".fish")
		if err := os.WriteFile(path, []byte(m.Body), 0o644); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command("fish", "--no-execute", path).CombinedOutput(); err != nil {
			t.Errorf("module %s has invalid fish syntax:\n%s", m.Name, out)
		}
	}

	// The sourced env snippet must also parse.
	env, err := Fish.EnvFor()
	if err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(dir, "env.fish")
	if err := os.WriteFile(envPath, []byte(env), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("fish", "--no-execute", envPath).CombinedOutput(); err != nil {
		t.Errorf("fish env snippet has invalid syntax:\n%s", out)
	}
}

// TestFishModuleCount guards against a module going missing from the fish port:
// every fish module in the load order must exist and be non-empty.
func TestFishModuleCount(t *testing.T) {
	mods, err := Fish.modulesFor()
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 8 {
		t.Errorf("expected 8 fish modules, got %d", len(mods))
	}
	for _, m := range mods {
		if len(m.Body) == 0 {
			t.Errorf("fish module %s is empty", m.Name)
		}
	}
}
