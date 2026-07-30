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

// TestPromptModulesLoadWithoutRuntimeError actually SOURCES the prompt modules
// in a real zsh and fails on any stderr output. `zsh -n` only parses — it can't
// catch runtime faults like a `(( ... ))` math expression over an empty/unset
// array ("bad math expression: operand expected"), which is exactly the class
// of bug that once broke the prompt at `exec zsh`. We load the two prompt
// modules under the harshest conditions: an empty precmd_functions array, no
// prior hooks — the state at first module load.
func TestPromptModulesLoadWithoutRuntimeError(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available; skipping shell runtime check")
	}
	mods, err := Modules()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	byName := map[string]string{}
	for _, m := range mods {
		path := filepath.Join(dir, m.Name+".zsh")
		if err := os.WriteFile(path, []byte(m.Body), 0o644); err != nil {
			t.Fatal(err)
		}
		byName[m.Name] = path
	}

	for _, name := range []string{"leftprompt", "prompt"} {
		path, ok := byName[name]
		if !ok {
			continue
		}
		// Source the module in a fresh non-interactive zsh with an explicitly
		// empty precmd_functions — the exact edge case that produced the "bad
		// math expression" fault. Any stderr means a runtime error.
		script := "precmd_functions=(); autoload -Uz add-zsh-hook 2>/dev/null; source " + path
		out, _ := exec.Command("zsh", "-c", script).CombinedOutput()
		if len(out) > 0 {
			t.Errorf("module %s produced runtime output when sourced (expected none):\n%s", name, out)
		}
	}
}
