// Package shellcfg generates and manages the opsforge zsh layer:
// cached completions for installed tools, a few curated aliases and an
// optional kube-context segment in the right prompt.
//
// The user-facing contract mirrors starship/direnv/mise:
//
//	eval "$(opsforge shell env)"   # in ~/.zshrc, added by `shell install`
//	opsforge shell sync            # regenerates cached completion scripts
package shellcfg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
)

const (
	markerStart = "# >>> opsforge shell layer >>>"
	markerEnd   = "# <<< opsforge shell layer <<<"
)

// aliases maps a catalog tool name to the aliases it enables. Kept
// deliberately short: only near-universal muscle-memory shortcuts.
var aliases = map[string][]string{
	"kubectl":   {"alias k=kubectl"},
	"terraform": {"alias tf=terraform"},
	"docker":    {`alias dc="docker compose"`},
}

// CompletionsDir is where cached completion scripts live.
func CompletionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "opsforge", "completions"), nil
}

// Env renders the zsh snippet meant to be eval'd from ~/.zshrc. It must
// stay fast: only PATH lookups, no tool subprocesses.
func Env(tools []catalog.Tool) (string, error) {
	dir, err := CompletionsDir()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", markerStart)
	// compinit may already have run in the user's zshrc; the compdef
	// guard avoids a second, slow invocation.
	fmt.Fprintf(&b, `if ! typeset -f compdef >/dev/null 2>&1; then
  autoload -Uz compinit && compinit -u
fi
if [ -d %[1]q ]; then
  for _of_script in %[1]q/*.zsh(N); do
    source "$_of_script"
  done
  unset _of_script
fi
`, dir)
	for _, t := range tools {
		if _, err := exec.LookPath(t.Bin); err != nil {
			continue
		}
		for _, a := range aliases[t.Name] {
			fmt.Fprintf(&b, "%s\n", a)
		}
		if t.Name == "kubectl" {
			b.WriteString(kubePrompt)
		}
	}
	fmt.Fprintf(&b, "%s\n", markerEnd)
	return b.String(), nil
}

// kubePrompt shows the current kube context on the right prompt, only
// when the user has not already claimed RPROMPT.
const kubePrompt = `_opsforge_kube_ctx() {
  local c
  c=$(kubectl config current-context 2>/dev/null) || return
  print -n "⎈ ${c}"
}
if [[ -z "$RPROMPT" ]]; then
  setopt PROMPT_SUBST
  RPROMPT='%F{blue}$(_opsforge_kube_ctx)%f'
fi
`

// Sync regenerates cached completion scripts for every installed tool
// exposing a native zsh completion command. It returns the tool names
// that were synced.
func Sync(tools []catalog.Tool) ([]string, error) {
	dir, err := CompletionsDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var synced []string
	for _, t := range tools {
		if len(t.CompletionZsh) == 0 || !detect.Tool(t).Installed {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		gen := exec.CommandContext(ctx, t.CompletionZsh[0], t.CompletionZsh[1:]...)
		gen.Env = detect.SafeProbeEnv() // never let a probe trigger cloud auth
		gen.WaitDelay = time.Second     // see detect.version: children may hold the pipe
		out, err := gen.Output()
		cancel()
		if err != nil || len(out) == 0 {
			continue // a tool refusing to emit its completion is not fatal
		}
		path := filepath.Join(dir, t.Name+".zsh")
		if err := os.WriteFile(path, out, 0o644); err != nil {
			return synced, fmt.Errorf("writing %s: %w", path, err)
		}
		synced = append(synced, t.Name)
	}
	return synced, nil
}

// InstallToZshrc idempotently adds the eval line to ~/.zshrc inside
// marker comments, replacing any previous opsforge block.
func InstallToZshrc() (string, error) {
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
	return path, nil
}

// InstalledInZshrc reports whether ~/.zshrc already contains the block.
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

// RemoveBlock strips a previously installed opsforge block, returning
// the remaining content unchanged when no block is present.
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
