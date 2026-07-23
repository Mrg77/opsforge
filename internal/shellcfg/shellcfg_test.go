package shellcfg

import (
	"strings"
	"testing"
)

func TestRemoveBlock(t *testing.T) {
	block := markerStart + "\neval \"$(opsforge shell env)\"\n" + markerEnd + "\n"
	cases := []struct {
		name, in, want string
	}{
		{"no block", "export FOO=1\n", "export FOO=1\n"},
		{"only block", block, ""},
		{"block at end", "export FOO=1\n" + block, "export FOO=1\n"},
		{"block in middle", "a\n" + block + "b\n", "a\nb\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := RemoveBlock(c.in); got != c.want {
				t.Errorf("RemoveBlock() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestEnvIsFastAndWellFormed(t *testing.T) {
	out, err := Env()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, markerStart) {
		t.Error("env output does not start with the opsforge marker")
	}
	if !strings.Contains(out, "compinit") {
		t.Error("env output does not guard compinit")
	}
	if !strings.Contains(out, markerEnd) {
		t.Error("env output is not terminated by the end marker")
	}
	// Env must not shell out — it only sources prewritten files.
	if strings.Contains(out, "$(") && !strings.Contains(out, "opsforge shell env") {
		// the eval line itself is added to zshrc, not here; env output
		// should have no command substitutions of its own.
		t.Error("env output contains a command substitution; it must stay startup-cheap")
	}
}

func TestModulesLoadAndAreNonEmpty(t *testing.T) {
	mods, err := Modules()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"prompt": true, "aliases": true, "integrations": true,
		"completions-special": true, "interactive": true, "help": true, "guards": true, "leftprompt": true,
		"cvenotify": true,
	}
	for _, m := range mods {
		if !want[m.Name] {
			t.Errorf("unexpected module %q", m.Name)
		}
		if strings.TrimSpace(m.Body) == "" {
			t.Errorf("module %q is empty", m.Name)
		}
		delete(want, m.Name)
	}
	if len(want) != 0 {
		t.Errorf("missing modules: %v", want)
	}
}

func TestGuardsModuleDelegatesToPolicyEngine(t *testing.T) {
	mods, _ := Modules()
	var guards string
	for _, m := range mods {
		if m.Name == "guards" {
			guards = m.Body
		}
	}
	// The rules now live in the Go policy engine; the module must wire the
	// accept-line widget, delegate the decision to `opsforge guard check`,
	// keep a cheap prefilter, and still document the escape hatch.
	for _, needle := range []string{"accept-line", "opsforge guard check", "_opsforge_looks_dangerous", "OPSFORGE_GUARDS=0"} {
		if !strings.Contains(guards, needle) {
			t.Errorf("guards module missing expected content %q", needle)
		}
	}
}
