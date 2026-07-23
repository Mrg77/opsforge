package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/history"
	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/ui"
)

var historyLimit int

var historyCmd = &cobra.Command{
	Use:   "history [family|tool]",
	Short: "Your recent shell commands, filtered by tool family (kube, git, tf…)",
	Long: `Pull your recent shell history for a family of DevOps tools — so you can
find that kubectl port-forward from last week without scrolling through
everything else.

Built-in families group the tools you think of together:
  kube    kubectl, helm, k9s, kubectx, kustomize, stern, argocd…
  git     git, gh, glab, lazygit
  tf      terraform, tofu, terragrunt, tflint
  docker  docker, docker-compose, podman, colima
  cloud   aws, gcloud, az, eksctl, flyctl
  ansible ansible, ansible-playbook, ansible-vault

Pass a family name, or any executable to filter by that single tool.`,
	Example: `  opsforge history kube          # recent kubectl/helm/k9s… commands
  opsforge history git           # recent git commands
  opsforge history terraform     # a single tool, by name
  opsforge history kube --json   # machine-readable`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := history.HistoryFile()
		if path == "" {
			return fmt.Errorf("no shell history file found (set $HISTFILE, or use zsh/bash)")
		}

		if len(args) == 0 {
			return historyOverview(path)
		}

		key := args[0]
		var bins []string
		var label string
		if fam, ok := history.FamilyByKey(key); ok {
			bins, label = fam.Bins, fam.Label
		} else {
			// Not a known family — treat the argument as a single tool name.
			bins, label = []string{key}, key
		}

		entries, err := history.Query(path, bins, historyLimit)
		if err != nil {
			return err
		}

		if output.JSON {
			return output.Emit(struct {
				Family  string          `json:"family"`
				Count   int             `json:"count"`
				Entries []history.Entry `json:"entries"`
			}{key, len(entries), entries})
		}

		fmt.Println(ui.Header("opsforge history — "+label,
			fmt.Sprintf("your recent %s commands, most recent first", label)))
		fmt.Println()
		if len(entries) == 0 {
			fmt.Println(ui.Dim.Render("  Nothing yet for this family in your history."))
			return nil
		}
		for _, e := range entries {
			line := "  " + ui.Prompt + " " + e.Command
			if e.Count > 1 {
				line += ui.Faint.Render(fmt.Sprintf("  ×%d", e.Count))
			}
			fmt.Println(line)
		}
		fmt.Println()
		fmt.Println(ui.Faint.Render(fmt.Sprintf("  %d command(s) · try `opsforge history <family>` — %s",
			len(entries), strings.Join(historyFamilyKeys(), ", "))))
		return nil
	},
}

// historyOverview lists the families with how many recent commands each
// has, so `opsforge history` with no argument is a useful menu.
func historyOverview(path string) error {
	type row struct {
		Key   string `json:"family"`
		Label string `json:"label"`
		Count int    `json:"count"`
	}
	// One pass over the history file for all families (not one read each).
	counts, err := history.CountByFamily(path, history.Families)
	if err != nil {
		return err
	}
	var rows []row
	for _, f := range history.Families {
		rows = append(rows, row{f.Key, f.Label, counts[f.Key]})
	}

	if output.JSON {
		return output.Emit(rows)
	}

	fmt.Println(ui.Header("opsforge history", "your shell history, grouped by DevOps tool family"))
	fmt.Println()
	for _, r := range rows {
		count := ui.Dim.Render(fmt.Sprintf("%d command(s)", r.Count))
		if r.Count == 0 {
			count = ui.Faint.Render("none yet")
		}
		fmt.Printf("  %s %s  %s\n", ui.OKMark(), ui.Label(r.Key, 9), count)
		fmt.Printf("      %s\n", ui.Faint.Render(r.Label))
	}
	fmt.Println()
	fmt.Println(ui.Dim.Render("  See one: ") + ui.Accent.Render("opsforge history kube"))
	return nil
}

func historyFamilyKeys() []string {
	keys := make([]string, 0, len(history.Families))
	for _, f := range history.Families {
		keys = append(keys, f.Key)
	}
	return keys
}

func init() {
	historyCmd.Flags().IntVarP(&historyLimit, "limit", "n", 20,
		"max number of commands to show (0 = all)")
	rootCmd.AddCommand(historyCmd)
}
