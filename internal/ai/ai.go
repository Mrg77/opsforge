// Package ai is opsforge's optional AI layer. It stays true to the tool's
// terminal-native philosophy: instead of asking you for an API key, it drives an
// AI CLI you already have installed (the Claude CLI, gemini-cli, …). Zero config
// if one is present; a clear message pointing you to a free option if not.
//
// Resolution order, most-preferred first:
//  1. $OPSFORGE_AI_CMD — any command that reads a prompt on stdin (full escape hatch)
//  2. a known AI CLI on PATH, in the order below
//
// The AI never runs anything: opsforge feeds it text (a command to explain, or a
// summary of findings) and streams back the answer. Nothing here reads secrets.
package ai

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Backend is a resolved way to reach an AI.
type Backend struct {
	// Name is a short human label ("Claude CLI", "gemini-cli", "custom command"…).
	Name string
	// Kind is a stable key: "custom" | "claude" | "gemini" | "llm" | "aichat" | "ollama".
	Kind string
	// how to invoke it: argv template where the prompt is passed as described by
	// stdinPrompt (prompt on stdin) or appended as the final arg.
	argv        []string
	stdinPrompt bool
}

// knownCLIs are the AI CLIs opsforge detects, in preference order. Claude and the
// hosted-model CLIs come first; ollama (local model) is last — it's a fallback,
// not the headline, since most people don't run a local model.
var knownCLIs = []struct {
	bin         string
	name        string
	kind        string
	argv        []string // {bin} placeholders filled at run time; prompt handling below
	stdinPrompt bool
}{
	{bin: "claude", name: "Claude CLI", kind: "claude", argv: []string{"claude", "-p"}, stdinPrompt: false},
	{bin: "gemini", name: "gemini-cli", kind: "gemini", argv: []string{"gemini", "-p"}, stdinPrompt: false},
	{bin: "llm", name: "llm (Simon Willison)", kind: "llm", argv: []string{"llm"}, stdinPrompt: true},
	{bin: "aichat", name: "aichat", kind: "aichat", argv: []string{"aichat"}, stdinPrompt: true},
	{bin: "ollama", name: "ollama (local model)", kind: "ollama", argv: []string{"ollama", "run", ollamaModel()}, stdinPrompt: false},
}

func ollamaModel() string {
	if m := os.Getenv("OPSFORGE_OLLAMA_MODEL"); m != "" {
		return m
	}
	return "llama3.2"
}

// Detect resolves the AI backend, or returns (nil, nil) if none is available so
// callers can print guidance rather than error out.
func Detect() *Backend {
	if custom := strings.TrimSpace(os.Getenv("OPSFORGE_AI_CMD")); custom != "" {
		return &Backend{
			Name: "custom command ($OPSFORGE_AI_CMD)", Kind: "custom",
			argv: []string{"sh", "-c", custom}, stdinPrompt: true,
		}
	}
	for _, c := range knownCLIs {
		if _, err := exec.LookPath(c.bin); err == nil {
			return &Backend{Name: c.name, Kind: c.kind, argv: c.argv, stdinPrompt: c.stdinPrompt}
		}
	}
	return nil
}

// Available reports whether any AI backend is configured.
func Available() bool { return Detect() != nil }

// Run streams the backend's answer for the given prompt directly to stdout/stderr
// (so the model's own progressive output is preserved). It's for interactive use.
func (b *Backend) Run(ctx context.Context, prompt string) error {
	c := b.command(ctx, prompt)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	return c.Run()
}

// Capture runs the backend and returns its full answer as a string, for callers
// that want to post-process or embed it.
func (b *Backend) Capture(ctx context.Context, prompt string) (string, error) {
	c := b.command(ctx, prompt)
	out, err := c.Output()
	return strings.TrimSpace(string(out)), err
}

func (b *Backend) command(ctx context.Context, prompt string) *exec.Cmd {
	if b.stdinPrompt {
		c := exec.CommandContext(ctx, b.argv[0], b.argv[1:]...)
		c.Stdin = strings.NewReader(prompt)
		return c
	}
	args := append(append([]string{}, b.argv[1:]...), prompt)
	return exec.CommandContext(ctx, b.argv[0], args...)
}

// SetupHint is the message shown when no backend is found — free-first, honest.
func SetupHint() string {
	return `No AI backend found. opsforge drives an AI CLI you already have — pick one:

  • Gemini CLI — free, no card:   brew install gemini-cli   (or: npm i -g @google/gemini-cli)
  • Claude CLI:                    https://claude.com/claude-code
  • llm / aichat:                  pipx install llm   ·   brew install aichat
  • or set OPSFORGE_AI_CMD to any command that reads a prompt on stdin.

A chat subscription (Claude Max, ChatGPT Plus) doesn't grant API access — a CLI
like the Gemini CLI gives you a free key in a couple of clicks.`
}

var _ = fmt.Sprintf // reserved for future formatted hints
