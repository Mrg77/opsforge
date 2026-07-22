package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/ui"
)

// helpRow is one line in the shell cheat-sheet.
type helpRow struct{ key, desc string }

// helpGroup is a titled set of rows.
type helpGroup struct {
	title string
	rows  []helpRow
}

var shellHelpGroups = []helpGroup{
	{"Interactive editing", []helpRow{
		{"type…", "a completion menu opens automatically — navigate with ↑↓ or Tab"},
		{"→", "accept the grey inline suggestion (from your history)"},
		{"<cmd> ?", "show that command's --help, nicely rendered (e.g. `kubectl get ?`)"},
		{"?", "this help panel"},
		{"??", "let AI explain your last command / why it failed"},
	}},
	{"Kubernetes", []helpRow{
		{"k", "kubectl"},
		{"kx [ctx]", "switch context (fzf picker with no arg)"},
		{"kn [ns]", "switch namespace (fzf picker with no arg)"},
		{"kg / kd / kl", "kubectl get / describe / logs"},
	}},
	{"Aliases", []helpRow{
		{"tf", "terraform"},
		{"dc", "docker compose"},
		{"h", "helm"},
		{"gst / gd", "git status / git diff"},
	}},
	{"Prompt", []helpRow{
		{"dir  branch*", "repo-relative path · git branch (*=dirty ?=untracked ⇡⇣=ahead/behind)"},
		{"3.0s", "duration of the last command when it was slow"},
		{"❯", "cyan normally, red when the last command failed"},
	}},
	{"Safety", []helpRow{
		{"prod guard", "destructive commands on a prod-looking kube context ask to confirm"},
		{"OPSFORGE_GUARDS=0", "disable the guard for this session"},
	}},
	{"Manage it", []helpRow{
		{"opsforge status", "workstation cockpit at a glance"},
		{"opsforge doctor", "full health check"},
		{"opsforge theme", "change the color theme"},
		{"opsforge shell uninstall", "remove this environment (restores ~/.zshrc)"},
	}},
}

var shellHelpCmd = &cobra.Command{
	Use:    "help",
	Short:  "Show the opsforge shell cheat-sheet (also: press ? on an empty line)",
	Hidden: true, // primarily invoked by the ? widget
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ui.Header("opsforge shell", "your DevOps shell — keys, aliases, and helpers"))
		fmt.Println()
		// Widest key across all groups, for aligned columns.
		w := 0
		for _, g := range shellHelpGroups {
			for _, r := range g.rows {
				if l := len([]rune(r.key)); l > w {
					w = l
				}
			}
		}
		for _, g := range shellHelpGroups {
			fmt.Println(ui.Section(g.title))
			for _, r := range g.rows {
				fmt.Printf("  %s  %s\n", ui.Accent.Render(pad(r.key, w)), ui.Dim.Render(r.desc))
			}
			fmt.Println()
		}
		fmt.Println(ui.Dim.Render("  Press ? anytime · this is `opsforge shell help`"))
		return nil
	},
}

func pad(s string, w int) string {
	for len([]rune(s)) < w {
		s += " "
	}
	return s
}

func init() {
	shellCmd.AddCommand(shellHelpCmd)
}
