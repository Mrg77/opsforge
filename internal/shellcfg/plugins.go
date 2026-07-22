package shellcfg

import (
	"os/exec"
)

// InteractivePlugins are the zsh plugins that provide the Warp/Fish-like
// editing experience: a live completion menu that appears as you type
// (zsh-autocomplete), gray inline suggestions (zsh-autosuggestions), and
// command-line coloring (zsh-syntax-highlighting). They are brew formulas;
// interactive.zsh sources whichever are present, so a failed install just
// means a missing feature, never a broken shell.
var InteractivePlugins = []string{
	"zsh-autocomplete",
	"zsh-autosuggestions",
	"zsh-syntax-highlighting",
}

// RenderingTools are CLI tools (not zsh plugins) that make the experience
// nicer when present: bat gives syntax-colored `?` help and file viewing.
// They degrade gracefully if missing, so install failures are non-fatal.
var RenderingTools = []string{"bat"}

// PluginStatus reports whether an interactive plugin is installed.
type PluginStatus struct {
	Name      string
	Installed bool
}

// InteractivePluginStatus checks which interactive plugins brew has.
func InteractivePluginStatus() []PluginStatus {
	out := make([]PluginStatus, 0, len(InteractivePlugins))
	for _, p := range InteractivePlugins {
		out = append(out, PluginStatus{Name: p, Installed: brewHas(p)})
	}
	return out
}

// EnsureInteractivePlugins installs any missing interactive plugin via
// brew. It returns the names it installed and the names it failed on;
// failures are non-fatal (the module degrades gracefully). When brew is
// absent, everything is reported as failed and nothing is attempted.
func EnsureInteractivePlugins() (installed, failed []string) {
	if _, err := exec.LookPath("brew"); err != nil {
		return nil, append([]string(nil), InteractivePlugins...)
	}
	for _, p := range append(append([]string{}, InteractivePlugins...), RenderingTools...) {
		if brewHas(p) {
			continue
		}
		if err := exec.Command("brew", "install", p).Run(); err != nil {
			failed = append(failed, p)
			continue
		}
		installed = append(installed, p)
	}
	return installed, failed
}

func brewHas(formula string) bool {
	// `brew list --formula <name>` exits 0 only when installed.
	return exec.Command("brew", "list", "--formula", formula).Run() == nil
}
