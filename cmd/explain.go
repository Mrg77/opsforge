package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	explainDim  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	explainHead = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
)

var explainLast bool

// lastCmdDir is where the shell hook records the last command and its
// exit status (written by the help.zsh precmd hook).
func lastCmdDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "opsforge"), nil
}

var explainCmd = &cobra.Command{
	Use:   "explain [command...]",
	Short: "Explain a shell command or your last failure, via your AI CLI",
	Long: `Ask an AI to explain a command — what it does, why it might have failed,
and how to fix it.

  opsforge explain --last          # explain the last command you ran (?? in the shell)
  opsforge explain "kubectl drain" # explain any command

The AI backend is pluggable:
  1. $OPSFORGE_AI_CMD if set — a shell command receiving the prompt on stdin
  2. the 'claude' CLI when installed
  3. the 'ollama' CLI when installed (model llama3.2)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var prompt string
		switch {
		case explainLast:
			dir, err := lastCmdDir()
			if err != nil {
				return err
			}
			cmdBytes, err := os.ReadFile(filepath.Join(dir, "last-cmd"))
			if err != nil {
				return fmt.Errorf("no last command recorded — enable the shell environment (`opsforge shell install`) first")
			}
			status, _ := os.ReadFile(filepath.Join(dir, "last-status"))
			lastCmd := strings.TrimSpace(string(cmdBytes))
			lastStatus := strings.TrimSpace(string(status))
			if lastCmd == "" {
				return fmt.Errorf("no last command recorded yet")
			}
			fmt.Printf("%s %s %s\n\n", explainHead.Render("Explaining:"), lastCmd,
				explainDim.Render("(exit "+lastStatus+")"))
			prompt = fmt.Sprintf(
				"I ran this shell command on macOS and it exited with code %s:\n\n  %s\n\n"+
					"Explain briefly what likely went wrong and give the corrected command. "+
					"Be concise: a short diagnosis and the fix.", lastStatus, lastCmd)
		case len(args) > 0:
			target := strings.Join(args, " ")
			fmt.Printf("%s %s\n\n", explainHead.Render("Explaining:"), target)
			prompt = fmt.Sprintf(
				"Explain this shell command concisely: what it does, its risks, and a usage example:\n\n  %s",
				target)
		default:
			return fmt.Errorf("nothing to explain — pass a command or use --last")
		}
		return runAI(prompt)
	},
}

// runAI resolves the AI backend and streams its answer to the terminal.
func runAI(prompt string) error {
	if custom := os.Getenv("OPSFORGE_AI_CMD"); custom != "" {
		c := exec.Command("sh", "-c", custom)
		c.Stdin = strings.NewReader(prompt)
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		return c.Run()
	}
	if _, err := exec.LookPath("claude"); err == nil {
		c := exec.Command("claude", "-p", prompt)
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		return c.Run()
	}
	if _, err := exec.LookPath("ollama"); err == nil {
		c := exec.Command("ollama", "run", "llama3.2", prompt)
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		return c.Run()
	}
	fmt.Println(explainDim.Render(`No AI backend found. Configure one of:
  - install the Claude CLI (https://claude.com/claude-code), or
  - install ollama (https://ollama.com) and pull a model, or
  - set OPSFORGE_AI_CMD to any command that reads a prompt on stdin`))
	return fmt.Errorf("no AI backend available")
}

func init() {
	explainCmd.Flags().BoolVar(&explainLast, "last", false,
		"explain the last command you ran (used by the shell '??' shortcut)")
	rootCmd.AddCommand(explainCmd)
}
