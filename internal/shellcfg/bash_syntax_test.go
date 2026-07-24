package shellcfg

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestModulesAreValidBash runs `bash -n` (parse only) on every embedded bash
// module and on the bash env snippet. A syntax error here would break a bash
// user's shell at startup. Skipped when bash is unavailable.
func TestModulesAreValidBash(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available; skipping bash syntax check")
	}
	mods, err := Bash.modulesFor()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, m := range mods {
		path := filepath.Join(dir, m.Name+".bash")
		if err := os.WriteFile(path, []byte(m.Body), 0o644); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command(bash, "-n", path).CombinedOutput(); err != nil {
			t.Errorf("module %s has invalid bash syntax:\n%s", m.Name, out)
		}
	}

	env, err := Bash.EnvFor()
	if err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(dir, "env.bash")
	if err := os.WriteFile(envPath, []byte(env), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(bash, "-n", envPath).CombinedOutput(); err != nil {
		t.Errorf("bash env snippet has invalid syntax:\n%s", out)
	}
}

func TestBashModuleCount(t *testing.T) {
	mods, err := Bash.modulesFor()
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 6 {
		t.Errorf("expected 6 bash modules, got %d", len(mods))
	}
	for _, m := range mods {
		if len(m.Body) == 0 {
			t.Errorf("bash module %s is empty", m.Name)
		}
	}
}
