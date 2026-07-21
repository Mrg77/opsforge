package shellcfg

import (
	"os/exec"
)

// InteractivePlugins are the zsh plugins that provide the modern editing
// experience (inline suggestions + syntax highlighting). They are brew
// formulas; interactive.zsh sources whichever are present, so a failed
// install just means a missing feature, never a broken shell.
//
// Note: zsh-autocomplete is deliberately excluded — it conflicts with an
// existing compinit in the user's .zshrc and breaks TAB. The navigable
// TAB menu in interactive.zsh (native zsh) covers that need reliably.
var InteractivePlugins = []string{
	"zsh-autosuggestions",
	"zsh-syntax-highlighting",
}

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
	for _, p := range InteractivePlugins {
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
