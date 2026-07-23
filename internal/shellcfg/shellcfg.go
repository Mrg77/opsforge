// Package shellcfg generates and manages the opsforge zsh layer: a set
// of modular scripts (prompt, guards, aliases, integrations) plus cached
// tool completions, wired into the user's shell through a single eval
// line in ~/.zshrc.
//
// The user-facing contract mirrors starship/direnv/mise:
//
//	eval "$(opsforge shell env)"   # in ~/.zshrc, added by `shell install`
//	opsforge shell sync            # regenerates cached completion scripts
//
// Modules are embedded from modules/*.zsh and written to
// ~/.config/opsforge/shell/ on `shell install`, which keeps ~/.zshrc to
// a single line and makes `shell doctor`/`shell uninstall` clean.
package shellcfg

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
)

//go:embed modules/*.zsh
var moduleFS embed.FS

const (
	markerStart = "# >>> opsforge shell layer >>>"
	markerEnd   = "# <<< opsforge shell layer <<<"
)

// Module is one embedded zsh feature file.
type Module struct {
	Name string // e.g. "prompt"
	Body string
}

// Modules returns the embedded feature modules in load order. Order
// matters: aliases before integrations (so integrations can override),
// guards last (the accept-line widget should wrap everything).
func Modules() ([]Module, error) {
	order := []string{"leftprompt", "prompt", "aliases", "integrations", "completions-special", "interactive", "help", "guards", "notify"}
	var mods []Module
	for _, name := range order {
		body, err := moduleFS.ReadFile("modules/" + name + ".zsh")
		if err != nil {
			return nil, fmt.Errorf("reading module %s: %w", name, err)
		}
		mods = append(mods, Module{Name: name, Body: string(body)})
	}
	return mods, nil
}

// ConfigDir is where opsforge writes its shell modules.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "opsforge", "shell"), nil
}

// CompletionsDir is where cached completion scripts live.
func CompletionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "opsforge", "completions"), nil
}

// WriteModules materializes the embedded modules to ConfigDir and
// returns the directory. Called by `shell install`.
func WriteModules() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	mods, err := Modules()
	if err != nil {
		return "", err
	}
	for _, m := range mods {
		path := filepath.Join(dir, m.Name+".zsh")
		if err := os.WriteFile(path, []byte(m.Body), 0o644); err != nil {
			return "", fmt.Errorf("writing %s: %w", path, err)
		}
	}
	return dir, nil
}

// Env renders the zsh snippet meant to be eval'd from ~/.zshrc. It must
// stay fast: PATH lookups and sourcing of prewritten files only, no tool
// subprocesses at shell-startup time.
func Env() (string, error) {
	cfgDir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	complDir, err := CompletionsDir()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", markerStart)
	// Put cached completions on fpath BEFORE compinit so #compdef files
	// are autoloaded (not sourced — sourcing a #compdef file at top level
	// calls _arguments outside a completion context and errors). Then run
	// compinit, guarded so a shell that already ran it isn't slowed twice.
	fmt.Fprintf(&b, `if [ -d %[1]q ]; then fpath=(%[1]q $fpath); fi
if ! typeset -f compdef >/dev/null 2>&1; then
  autoload -Uz compinit && compinit -u
else
  # completions dir was added after the user's compinit — reload it
  autoload -Uz compinit && compinit -u -C
fi
# opsforge modules (prompt, aliases, integrations, guards)
if [ -d %[2]q ]; then
  for _of_m in %[2]q/leftprompt.zsh %[2]q/prompt.zsh %[2]q/aliases.zsh %[2]q/integrations.zsh %[2]q/completions-special.zsh %[2]q/interactive.zsh %[2]q/help.zsh %[2]q/guards.zsh %[2]q/notify.zsh; do
    [ -r "$_of_m" ] && source "$_of_m"
  done
  unset _of_m
fi
`, complDir, cfgDir)
	fmt.Fprintf(&b, "%s\n", markerEnd)
	return b.String(), nil
}

// Sync regenerates cached completion scripts for every installed tool
// exposing a native zsh completion command. Returns the synced names.
//
// It also re-materializes the feature modules (prompt, aliases,
// interactive, guards…) so that upgrading opsforge and running `shell
// sync` actually ships the new module behavior — otherwise the shell keeps
// running the modules written at first `shell install`.
func Sync(tools []catalog.Tool) ([]string, error) {
	if _, err := WriteModules(); err != nil {
		return nil, err
	}
	dir, err := CompletionsDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	// Drop legacy *.zsh completion files: we now write fpath-style _<bin>
	// files (autoloaded), and sourcing the old .zsh ones broke on
	// #compdef scripts like bat's.
	if old, _ := filepath.Glob(filepath.Join(dir, "*.zsh")); old != nil {
		for _, f := range old {
			os.Remove(f)
		}
	}

	var synced []string
	// Always generate opsforge's own completion first — the tool that
	// manages everyone else's completions should have its own.
	if out := runCompletion([]string{"opsforge", "completion", "zsh"}); looksLikeCompletion(out) {
		if err := os.WriteFile(filepath.Join(dir, "_opsforge"), out, 0o644); err == nil {
			synced = append(synced, "opsforge")
		}
	}
	for _, t := range tools {
		if !detect.Tool(t).Installed {
			continue
		}
		out := generateCompletion(t)
		if len(out) == 0 {
			continue
		}
		// fpath convention: the file is named _<bin> so compinit
		// autoloads it for that command.
		path := filepath.Join(dir, "_"+t.Bin)
		if err := os.WriteFile(path, out, 0o644); err != nil {
			return synced, fmt.Errorf("writing %s: %w", path, err)
		}
		synced = append(synced, t.Name)
	}
	sort.Strings(synced)
	return synced, nil
}

// generateCompletion produces a tool's zsh completion script. It uses the
// catalog's explicit CompletionZsh command when present, otherwise tries
// the common conventions (`<bin> completion zsh`, `<bin> --completion
// zsh`) so tools without an explicit entry still get completion when they
// follow the Cobra/urfave standard. Returns nil when nothing works.
func generateCompletion(t catalog.Tool) []byte {
	var attempts [][]string
	if len(t.CompletionZsh) > 0 {
		attempts = append(attempts, t.CompletionZsh)
	} else {
		attempts = [][]string{
			{t.Bin, "completion", "zsh"},
			{t.Bin, "--completion", "zsh"},
		}
	}
	for _, argv := range attempts {
		out := runCompletion(argv)
		// A real completion script defines a completion function; this
		// filters out help text or error output that exited zero.
		if looksLikeCompletion(out) {
			return out
		}
	}
	return nil
}

func runCompletion(argv []string) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	gen := exec.CommandContext(ctx, argv[0], argv[1:]...)
	gen.Env = detect.SafeProbeEnv() // never let a probe trigger cloud auth
	gen.WaitDelay = time.Second     // children may hold the pipe open
	out, err := gen.Output()
	if err != nil {
		return nil
	}
	return out
}

// looksLikeCompletion checks the output actually registers a zsh
// completion, so help/usage text that happens to exit 0 is rejected.
func looksLikeCompletion(out []byte) bool {
	if len(out) < 20 {
		return false
	}
	s := string(out)
	return strings.Contains(s, "#compdef") ||
		strings.Contains(s, "compdef ") ||
		strings.Contains(s, "_arguments") ||
		strings.Contains(s, "autoload")
}

// InstallToZshrc idempotently adds the eval line to ~/.zshrc inside
// marker comments and writes the module files.
func InstallToZshrc() (string, error) {
	if _, err := WriteModules(); err != nil {
		return "", err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".zshrc")
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	block := fmt.Sprintf("%s\neval \"$(opsforge shell env)\"\n%s\n", markerStart, markerEnd)
	content := RemoveBlock(string(existing))
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += block
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	// Drop any stale completion dump: it can hold references to functions
	// from plugins that are no longer installed, which makes zsh spew
	// "function definition file not found" on every prompt. zsh rebuilds
	// it cleanly on next start.
	clearCompDump(home)
	return path, nil
}

// clearCompDump removes zsh's cached completion dump so it is rebuilt
// fresh, discarding references to uninstalled plugins.
func clearCompDump(home string) {
	for _, name := range []string{".zcompdump", ".zcompdump.zwc"} {
		os.Remove(filepath.Join(home, name))
	}
	// Some setups store it under $ZDOTDIR or with a host suffix; a glob
	// covers the common ~/.zcompdump-<host>-<version> variants.
	matches, _ := filepath.Glob(filepath.Join(home, ".zcompdump-*"))
	for _, m := range matches {
		os.Remove(m)
	}
}

// UninstallFromZshrc removes the opsforge block from ~/.zshrc and deletes
// the module directory. Completion cache is left in place (cheap, and the
// user may re-enable). Returns the zshrc path touched.
func UninstallFromZshrc() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".zshrc")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(RemoveBlock(string(data))), 0o644); err != nil {
		return "", err
	}
	if dir, err := ConfigDir(); err == nil {
		os.RemoveAll(dir)
	}
	return path, nil
}

// InstalledInZshrc reports whether ~/.zshrc contains the opsforge block.
func InstalledInZshrc() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	data, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		return false
	}
	return strings.Contains(string(data), markerStart)
}

// RemoveBlock strips a previously installed opsforge block, returning the
// remaining content unchanged when no block is present.
func RemoveBlock(content string) string {
	start := strings.Index(content, markerStart)
	if start == -1 {
		return content
	}
	end := strings.Index(content, markerEnd)
	if end == -1 {
		return content
	}
	rest := content[end+len(markerEnd):]
	rest = strings.TrimPrefix(rest, "\n")
	return content[:start] + rest
}
