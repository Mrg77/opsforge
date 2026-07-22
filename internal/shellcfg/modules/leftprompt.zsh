# opsforge left prompt — a clean, informative PROMPT that never queries a
# cloud or a cluster. It shows: the current directory (repo-relative when
# inside a git repo), the git branch with a dirty/ahead/behind marker, the
# duration of the last command when it was slow, and a ❯ prompt char that
# turns red on a non-zero exit.
#
# opsforge installs its left prompt by default, but never fights an
# existing prompt framework (starship, powerlevel10k, oh-my-posh) — those
# are left untouched. It also skips a PROMPT you've clearly customized
# yourself (anything beyond the stock zsh/oh-my-zsh defaults), unless you
# force it with OPSFORGE_PROMPT=1. Disable entirely with OPSFORGE_PROMPT=0.

[[ "$OPSFORGE_PROMPT" == "0" ]] && return
# Respect an existing prompt framework.
[[ -n "$STARSHIP_SHELL" || -n "$POWERLEVEL9K_MODE" || -n "$POSH_THEME" ]] && return

if [[ "$OPSFORGE_PROMPT" != "1" ]]; then
  # Recognized stock defaults we're happy to replace. Anything else is
  # treated as your own prompt and left alone.
  case "$PROMPT" in
    ''|'%m%# '|'%m%# '|'%n@%m %1~ %# '|'%# '|'%n@%m %~ %# '|'%n@%m %1~ %#'|'%~ %# ') ;;
    *) return ;;
  esac
fi

setopt PROMPT_SUBST
autoload -Uz add-zsh-hook
zmodload zsh/datetime 2>/dev/null   # provides EPOCHREALTIME for timing

# --- timing: measure how long each command took -------------------------
_opsforge_timer_start() { _opsforge_timer=${EPOCHREALTIME:-$SECONDS} }
add-zsh-hook preexec _opsforge_timer_start

_opsforge_cmd_duration() {
  [[ -z "$_opsforge_timer" ]] && return
  local now=${EPOCHREALTIME:-$SECONDS}
  local elapsed=$(( now - _opsforge_timer ))
  unset _opsforge_timer
  # Only show it when it actually took a moment (>2s).
  (( elapsed < 2 )) && return
  if (( elapsed >= 60 )); then
    printf '%dm%ds' $(( elapsed / 60 )) $(( elapsed % 60 ))
  else
    printf '%.1fs' "$elapsed"
  fi
}

# --- git segment: branch + state, all from local files/commands --------
_opsforge_git_segment() {
  command -v git >/dev/null 2>&1 || return
  local branch
  branch=$(git symbolic-ref --short HEAD 2>/dev/null) \
    || branch=$(git rev-parse --short HEAD 2>/dev/null) || return

  # Dirty working tree?
  local dirty=""
  git diff --quiet --ignore-submodules HEAD 2>/dev/null || dirty="*"
  # Untracked files?
  [[ -n "$(git ls-files --others --exclude-standard 2>/dev/null | head -1)" ]] && dirty="${dirty}?"

  # Ahead / behind the upstream.
  local ahead_behind=""
  local counts
  counts=$(git rev-list --left-right --count HEAD...@{upstream} 2>/dev/null)
  if [[ -n "$counts" ]]; then
    local ahead=${counts%%[[:space:]]*} behind=${counts##*[[:space:]]}
    (( ahead > 0 )) && ahead_behind="${ahead_behind}⇡${ahead}"
    (( behind > 0 )) && ahead_behind="${ahead_behind}⇣${behind}"
  fi

  local color="%F{cyan}"
  [[ -n "$dirty" ]] && color="%F{yellow}"
  print -n " ${color}${branch}${dirty}${ahead_behind}%f"
}

# --- assemble the left prompt -------------------------------------------
_opsforge_precmd_prompt() {
  local last=$?
  local dur
  dur=$(_opsforge_cmd_duration)

  # directory: repo-relative path inside a git repo, else ~-shortened cwd
  local dir
  local root
  root=$(git rev-parse --show-toplevel 2>/dev/null)
  if [[ -n "$root" ]]; then
    local reponame=${root:t}
    local rel=${PWD#$root}
    dir="${reponame}${rel}"
  else
    dir="%~"
  fi

  local durseg=""
  [[ -n "$dur" ]] && durseg=" %F{242}${dur}%f"

  # ❯ turns red on failure.
  local mark="%F{cyan}❯%f"
  (( last != 0 )) && mark="%F{red}❯%f"

  PROMPT="%F{blue}${dir}%f\$(_opsforge_git_segment)${durseg}
${mark} "
}
add-zsh-hook precmd _opsforge_precmd_prompt
