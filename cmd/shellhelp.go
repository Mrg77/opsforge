package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/ui"
)

// helpRow is one line in the shell cheat-sheet.
type helpRow struct{ key, desc string }

// helpGroup is a titled set of rows with a leading icon.
type helpGroup struct {
	icon  string
	title string
	rows  []helpRow
}

var shellHelpGroups = []helpGroup{
	{"⌨", "Interactive editing", []helpRow{
		{"↑ / ↓", "walk history filtered by what you've typed (prefix of the line)"},
		{"→", "accept the grey inline suggestion from history"},
		{"Tab", "complete the current word (menu on a second Tab)"},
		{"^R", "search your whole history"},
		{"cmd ?", "that command's --help, nicely rendered"},
		{"?", "this help panel"},
		{"??", "AI explains your last command / why it failed"},
	}},
	{"⎈", "Kubernetes", []helpRow{
		{"k", "kubectl"},
		{"kx", "switch context (fzf picker, or `kx <ctx>`)"},
		{"kn", "switch namespace (fzf picker, or `kn <ns>`)"},
		{"kg kd kl", "kubectl get · describe · logs"},
	}},
	{"↦", "Aliases", []helpRow{
		{"tf", "terraform"},
		{"dc", "docker compose"},
		{"h", "helm"},
		{"gst gd", "git status · git diff"},
	}},
	{"❯", "Prompt legend", []helpRow{
		{"branch*", "* dirty · ? untracked · ⇡⇣ ahead/behind upstream"},
		{"3.0s", "shown when the last command was slow"},
		{"❯", "cyan — red when the last command failed"},
	}},
	{"⚠", "Safety", []helpRow{
		{"prod guard", "confirms destructive cmds on a prod kube context"},
		{"GUARDS=0", "OPSFORGE_GUARDS=0 disables it for this session"},
	}},
	{"◆", "Manage", []helpRow{
		{"status", "workstation cockpit at a glance"},
		{"doctor", "full health check"},
		{"theme", "change the color theme"},
		{"shell uninstall", "remove this environment (restores ~/.zshrc)"},
	}},
}

var shellHelpCmd = &cobra.Command{
	Use:    "help",
	Short:  "Show the opsforge shell cheat-sheet (also: press ? on an empty line)",
	Hidden: true, // primarily invoked by the ? widget
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ui.Header("opsforge shell", "keys · aliases · helpers — press ? anytime"))
		fmt.Println()
		for _, g := range shellHelpGroups {
			renderHelpGroup(g)
			fmt.Println()
		}
		fmt.Println(ui.Dim.Render("  <cmd> ? shows a command's help · ?? explains your last one"))
		return nil
	},
}

// renderHelpGroup prints one section: an iconed heading, then rows whose
// keys are aligned to the widest key IN THIS GROUP (so short keys like
// `k` don't inherit a huge gap from a long key elsewhere).
func renderHelpGroup(g helpGroup) {
	fmt.Printf("  %s %s\n", ui.Accent.Render(g.icon), ui.Heading.Render(g.title))
	w := 0
	for _, r := range g.rows {
		if l := len([]rune(r.key)); l > w {
			w = l
		}
	}
	for _, r := range g.rows {
		fmt.Printf("    %s  %s\n", ui.Accent.Render(pad(r.key, w)), ui.Dim.Render(r.desc))
	}
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
