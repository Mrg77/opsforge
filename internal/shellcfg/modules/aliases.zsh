# opsforge aliases & functions — muscle-memory shortcuts and small OPS
# helpers. Everything is guarded on the underlying tool being installed,
# so nothing shadows a command you do not have.

if command -v kubectl >/dev/null 2>&1; then
  alias k='kubectl'
  alias kg='kubectl get'
  alias kd='kubectl describe'
  alias kl='kubectl logs'
  alias kga='kubectl get all'

  # kx: switch kube context (fzf picker when available, else list/set).
  kx() {
    if [[ -n "$1" ]]; then kubectl config use-context "$1"; return; fi
    if command -v fzf >/dev/null 2>&1; then
      local ctx
      ctx=$(kubectl config get-contexts -o name | fzf --height 40% --prompt='context> ') || return
      [[ -n "$ctx" ]] && kubectl config use-context "$ctx"
    else
      kubectl config get-contexts
    fi
  }

  # kn: switch namespace for the current context.
  kn() {
    if [[ -n "$1" ]]; then kubectl config set-context --current --namespace="$1"; return; fi
    if command -v fzf >/dev/null 2>&1; then
      local ns
      ns=$(kubectl get ns -o name 2>/dev/null | sed 's|namespace/||' \
            | fzf --height 40% --prompt='namespace> ') || return
      [[ -n "$ns" ]] && kubectl config set-context --current --namespace="$ns"
    else
      kubectl get ns
    fi
  }
fi

command -v terraform >/dev/null 2>&1 && { alias tf='terraform'; alias tfp='terraform plan'; }
command -v docker    >/dev/null 2>&1 && alias dc='docker compose'
command -v helm      >/dev/null 2>&1 && alias h='helm'
command -v git       >/dev/null 2>&1 && { alias gst='git status'; alias gd='git diff'; }
command -v bat       >/dev/null 2>&1 && alias cat='bat --paging=never'
command -v eza       >/dev/null 2>&1 && alias ls='eza'

# `history` (the zsh builtin) shows only ~16 lines by default. Make the
# bare command show the last 200, while keeping `history 50`, `history -50`,
# `history 1` (everything) and every other argument working as before.
history() {
  if (( $# == 0 )); then
    builtin fc -l -200      # last 200; `history 1` still shows everything
  else
    builtin fc -l "$@"
  fi
}
# `hg <term>` — grep your whole history fast. For a DevOps-tool view,
# `opsforge history kube` groups it by family (kube/git/tf/docker/cloud…).
hg() { builtin fc -l 1 | grep -i -- "$@"; }
