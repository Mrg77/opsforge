package shellcfg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnvFor renders the snippet a shell eval's/sources at startup to load the
// opsforge layer. It must stay fast: sourcing prewritten files only, no tool
// subprocesses. zsh keeps its completion/compinit wiring (see Env); fish just
// sources its modules.
func (sh Shell) EnvFor() (string, error) {
	if sh == Zsh {
		return Env() // the original, completion-aware zsh path
	}
	cfgDir, err := sh.configDir()
	if err != nil {
		return "", err
	}
	sp := specs[sh]
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", markerStart)
	fmt.Fprintf(&b, "if test -d %[1]q\n", cfgDir)
	for _, name := range sp.order {
		fmt.Fprintf(&b, "    test -r %[1]q/%[2]s%[3]s; and source %[1]q/%[2]s%[3]s\n", cfgDir, name, sp.ext)
	}
	fmt.Fprintf(&b, "end\n")
	fmt.Fprintf(&b, "%s\n", markerEnd)
	return b.String(), nil
}

// evalLine is the single line opsforge writes into the rc file. Both zsh and
// fish support `opsforge shell env --shell <sh> | source`-style loading; we use
// each shell's idiom.
func (sh Shell) evalLine() string {
	if sh == Fish {
		return "opsforge shell env --shell fish | source"
	}
	return `eval "$(opsforge shell env)"`
}

// InstallTo idempotently wires the opsforge layer into a shell's rc file and
// writes the module files. Returns the rc path touched.
func (sh Shell) InstallTo() (string, error) {
	if _, err := sh.writeModules(); err != nil {
		return "", err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := specs[sh].rcPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	block := fmt.Sprintf("%s\n%s\n%s\n", markerStart, sh.evalLine(), markerEnd)
	content := RemoveBlock(string(existing))
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += block
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	if sh == Zsh {
		clearCompDump(home)
	}
	return path, nil
}

// UninstallFrom removes the opsforge block from a shell's rc file and deletes
// its module directory.
func (sh Shell) UninstallFrom() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := specs[sh].rcPath(home)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(RemoveBlock(string(data))), 0o644); err != nil {
		return "", err
	}
	if dir, err := sh.configDir(); err == nil {
		os.RemoveAll(dir)
	}
	return path, nil
}

// InstalledIn reports whether a shell's rc file contains the opsforge block.
func (sh Shell) InstalledIn() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	data, err := os.ReadFile(specs[sh].rcPath(home))
	if err != nil {
		return false
	}
	return strings.Contains(string(data), markerStart)
}

// RcPath returns the rc file opsforge wires into for a shell.
func (sh Shell) RcPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return specs[sh].rcPath(home), nil
}

// Ext is the module file extension for a shell (".zsh" / ".fish").
func (sh Shell) Ext() string { return specs[sh].ext }

// ModuleNames returns the module base names in load order (for `shell doctor`).
func (sh Shell) ModuleNames() []string { return specs[sh].order }

// ModuleDir exposes the shell's module directory (for `shell doctor`).
func (sh Shell) ModuleDir() (string, error) { return sh.configDir() }

// Shell is a supported interactive shell opsforge can wire itself into.
type Shell string

const (
	Zsh  Shell = "zsh"
	Fish Shell = "fish"
)

// SupportedShells lists the shells opsforge can install into.
func SupportedShells() []Shell { return []Shell{Zsh, Fish} }

// spec holds the per-shell knobs the generic install/env code needs.
type spec struct {
	ext    string   // module file extension, incl. dot (".zsh")
	subdir string   // embed subdir under modules/ ("" for zsh, "fish" for fish)
	order  []string // module load order (base names, no extension)
	rcPath func(home string) string
}

var specs = map[Shell]spec{
	Zsh: {
		ext:    ".zsh",
		subdir: "",
		order:  []string{"leftprompt", "prompt", "aliases", "integrations", "completions-special", "interactive", "help", "guards", "notify"},
		rcPath: func(home string) string { return filepath.Join(home, ".zshrc") },
	},
	Fish: {
		ext:    ".fish",
		subdir: "fish",
		// No completions-special: fish handles completions natively.
		order:  []string{"leftprompt", "prompt", "aliases", "integrations", "interactive", "help", "guards", "notify"},
		rcPath: func(home string) string { return filepath.Join(home, ".config", "fish", "config.fish") },
	},
}

// DetectShell picks the shell to target from $SHELL, defaulting to zsh (the
// original, fully-featured target) when it can't tell.
func DetectShell() Shell {
	sh := os.Getenv("SHELL")
	switch {
	case strings.Contains(sh, "fish"):
		return Fish
	default:
		return Zsh
	}
}

// ParseShell resolves a user-provided shell name, or an error listing the
// supported ones.
func ParseShell(name string) (Shell, error) {
	switch Shell(name) {
	case Zsh:
		return Zsh, nil
	case Fish:
		return Fish, nil
	default:
		return "", fmt.Errorf("unsupported shell %q (supported: zsh, fish)", name)
	}
}

// modulesFor returns the embedded feature modules for a shell, in load order.
func (sh Shell) modulesFor() ([]Module, error) {
	sp := specs[sh]
	dir := "modules"
	if sp.subdir != "" {
		dir = "modules/" + sp.subdir
	}
	var mods []Module
	for _, name := range sp.order {
		body, err := moduleFS.ReadFile(dir + "/" + name + sp.ext)
		if err != nil {
			return nil, fmt.Errorf("reading %s module %s: %w", sh, name, err)
		}
		mods = append(mods, Module{Name: name, Body: string(body)})
	}
	return mods, nil
}

// configDir is where opsforge writes a given shell's modules.
func (sh Shell) configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// zsh keeps the historical path (~/.config/opsforge/shell); fish gets its
	// own subdir so the two never collide.
	if sh == Zsh {
		return filepath.Join(home, ".config", "opsforge", "shell"), nil
	}
	return filepath.Join(home, ".config", "opsforge", "shell-"+string(sh)), nil
}

// writeModules materializes a shell's modules to its config dir.
func (sh Shell) writeModules() (string, error) {
	dir, err := sh.configDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	mods, err := sh.modulesFor()
	if err != nil {
		return "", err
	}
	sp := specs[sh]
	for _, m := range mods {
		path := filepath.Join(dir, m.Name+sp.ext)
		if err := os.WriteFile(path, []byte(m.Body), 0o644); err != nil {
			return "", fmt.Errorf("writing %s: %w", path, err)
		}
	}
	return dir, nil
}
