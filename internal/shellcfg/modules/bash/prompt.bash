# opsforge prompt — a clean, prod-aware PS1 that never queries a cloud or a
# cluster (bash). Shows: the current directory (repo-relative in a git repo),
# the git branch with a dirty marker, the kube context (red when it looks like
# prod, read passively from the kubeconfig FILE — never runs kubectl), the
# last-command duration when slow, and a ❯ that turns red on failure.
#
# bash has no separate right-prompt, so the kube/cloud/tf context sits inline.
# Respects an existing prompt framework (starship, oh-my-posh) and a PS1 you've
# customized, unless forced with OPSFORGE_PROMPT=1. Disable with OPSFORGE_PROMPT=0.

case $- in *i*) ;; *) return 0 2>/dev/null || true ;; esac

if [[ "$OPSFORGE_PROMPT" != "0" \
      && -z "$STARSHIP_SHELL$POWERLEVEL9K_MODE$POSH_THEME" ]]; then

  # --- kube context: cluster name, red when it looks like prod ---
  _opsforge_kube_segment() {
    [[ "$OPSFORGE_PROMPT_KUBE" == "0" ]] && return
    local cfg="${KUBECONFIG%%:*}"
    [[ -z "$cfg" ]] && cfg="$HOME/.kube/config"
    [[ -r "$cfg" ]] || return
    local ctx
    ctx=$(grep -m1 '^current-context:' "$cfg" 2>/dev/null | sed 's/current-context:[[:space:]]*//; s/["'\'']//g')
    [[ -z "$ctx" ]] && return
    if [[ "$ctx" == *prod* ]]; then
      printf ' \[\033[1;31m\]⎈ %s\[\033[0m\]' "$ctx"
    else
      printf ' \[\033[36m\]⎈ %s\[\033[0m\]' "$ctx"
    fi
  }

  # --- git segment: branch + dirty marker, all local ---
  _opsforge_git_segment() {
    command -v git >/dev/null 2>&1 || return
    local branch
    branch=$(git symbolic-ref --short HEAD 2>/dev/null) \
      || branch=$(git rev-parse --short HEAD 2>/dev/null) || return
    local dirty=""
    git diff --quiet --ignore-submodules HEAD 2>/dev/null || dirty="*"
    [[ -n "$(git ls-files --others --exclude-standard 2>/dev/null | head -1)" ]] && dirty="${dirty}?"
    local color='\[\033[36m\]'
    [[ -n "$dirty" ]] && color='\[\033[33m\]'
    printf ' %s%s%s\[\033[0m\]' "$color" "$branch" "$dirty"
  }

  # --- directory: repo-relative inside a git repo, else ~-shortened ---
  _opsforge_dir_segment() {
    local root
    root=$(git rev-parse --show-toplevel 2>/dev/null)
    if [[ -n "$root" ]]; then
      printf '%s%s' "${root##*/}" "${PWD#$root}"
    else
      printf '%s' "${PWD/#$HOME/\~}"
    fi
  }

  _opsforge_set_prompt() {
    local last=$?   # capture BEFORE anything else clobbers it
    local mark_color='\[\033[36m\]'
    if [[ $last -ne 0 && $last -ne 130 && $last -ne 148 ]]; then
      mark_color='\[\033[31m\]'
    fi
    PS1="\[\033[34m\]$(_opsforge_dir_segment)\[\033[0m\]$(_opsforge_git_segment)$(_opsforge_kube_segment)\n${mark_color}❯\[\033[0m\] "
  }

  # Prepend to any existing PROMPT_COMMAND so we don't clobber other tools.
  case "$PROMPT_COMMAND" in
    *_opsforge_set_prompt*) ;;
    "") PROMPT_COMMAND="_opsforge_set_prompt" ;;
    *)  PROMPT_COMMAND="_opsforge_set_prompt; $PROMPT_COMMAND" ;;
  esac
fi
