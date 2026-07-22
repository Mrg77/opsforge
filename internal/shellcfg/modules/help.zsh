# opsforge inline help — press "?" at the end of a command line to see
# what it does, without leaving your line. It runs the command's native
# --help, rendered cleanly (bat when available, else a light colorizer),
# then restores your prompt so you can keep typing. Disable with
# OPSFORGE_HELP=0.
#
# Example: type "kubectl get " then "?"  → shows `kubectl get --help`.

[[ "$OPSFORGE_HELP" == "0" ]] && return

# _opsforge_help_widget is bound to "?". It only intercepts when "?" is
# typed at the very end of a non-empty line whose first word is a real
# command; otherwise it inserts a literal "?" so globbing/other uses are
# unaffected.
_opsforge_help_widget() {
  # "?" on an empty line — show the opsforge shell help panel.
  if [[ -z "$BUFFER" ]]; then
    print
    _opsforge_help_panel
    zle reset-prompt
    return
  fi

  # "??" — the buffer already holds one "?" and the user typed a second:
  # explain the LAST command via AI (opsforge explain --last).
  if [[ "$BUFFER" == "?" ]]; then
    BUFFER=""
    print
    opsforge explain --last
    zle reset-prompt
    return
  fi

  if [[ $CURSOR -ne ${#BUFFER} || -z "${BUFFER// /}" ]]; then
    zle self-insert
    return
  fi

  local cmd="${${(z)BUFFER}[1]}"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    zle self-insert
    return
  fi

  # The '?' isn't in $BUFFER yet, so the buffer is exactly the command to
  # explain, e.g. "kubectl get node" -> kubectl get node --help.
  local -a parts
  parts=(${(z)BUFFER})

  # Capture help text once (KUBECONFIG neutralized so a kubectl --help can
  # never trigger cloud auth — /dev/null exposes no exec credential plugin).
  local help
  help="$(KUBECONFIG=/dev/null "${parts[@]}" --help 2>&1)"
  [[ -z "$help" ]] && help="$(KUBECONFIG=/dev/null "${parts[@]}" help 2>&1)"

  print                                   # newline after the current line
  _opsforge_help_render "${parts[*]}" "$help" | _opsforge_help_page
  zle reset-prompt
}

# _opsforge_help_render prints an elegant header (command + one-line gist)
# followed by the colorized body.
_opsforge_help_render() {
  local title="$1" body="$2"
  local width=${COLUMNS:-80}
  (( width > 100 )) && width=100

  # --- header bar ---------------------------------------------------------
  local rule="${(l:$width::─:)}"
  print -P "%F{cyan}${rule}%f"
  print -P "%F{cyan}%B  ❯ ${title} --help%b%f"
  # One-line gist: the first non-empty, non-"Usage:" line of the help.
  local gist
  gist="$(print -r -- "$body" | awk 'NF && $0 !~ /^Usage/ && $0 !~ /^[A-Za-z]+:/ {print; exit}')"
  [[ -n "$gist" ]] && print -P "%F{244}  ${gist//\%/%%}%f"
  print -P "%F{cyan}${rule}%f"

  # --- body ---------------------------------------------------------------
  if command -v bat >/dev/null 2>&1; then
    print -r -- "$body" | bat --style=plain --language=man --color=always --paging=never 2>/dev/null \
      || print -r -- "$body" | _opsforge_help_colorize
  else
    print -r -- "$body" | _opsforge_help_colorize
  fi
}

# _opsforge_help_colorize: light ANSI coloring fallback when bat is absent.
_opsforge_help_colorize() {
  awk '
    /^[A-Za-z][A-Za-z &-]*:[[:space:]]*$/ { printf "\033[1;36m%s\033[0m\n", $0; next }
    /^[[:space:]]*#/                      { printf "\033[32m%s\033[0m\n", $0; next }
    /^[[:space:]]*--?[A-Za-z]/            { printf "\033[33m%s\033[0m\n", $0; next }
    { print }
  '
}

# _opsforge_help_page: quit if it fits on one screen, else page with a
# clear "how to leave" prompt. No stray ':' or escape junk.
_opsforge_help_page() {
  if command -v less >/dev/null 2>&1; then
    LESS='-FRXQ' less --prompt='  ↑↓ scroll · q to close help ' 2>/dev/null || cat
  else
    cat
  fi
}

# _opsforge_help_panel renders the opsforge shell cheat-sheet — the help
# you get by pressing "?" on an empty line. It's rendered by opsforge
# itself (single source of truth, themed), falling back to a plain panel.
_opsforge_help_panel() {
  if command -v opsforge >/dev/null 2>&1; then
    opsforge shell help 2>/dev/null && return
  fi
  # Fallback if the binary is unavailable for some reason.
  print -P "%F{212}%B  opsforge shell%b%f — press ? for help, ?? to explain the last command"
  print -P "  %F{39}Interactive%f  ↑ history by prefix · → accept suggestion · <cmd> ? for its help"
  print -P "  %F{39}Aliases%f      k=kubectl · tf=terraform · dc=docker compose · kx/kn switch ctx/ns"
}

# Track the last command + exit status so `??` (and `opsforge explain
# --last`) know what to explain. Cheap: two tiny file writes per prompt.
_opsforge_track_last() {
  local code=$?
  local dir="$HOME/.cache/opsforge"
  [[ -d "$dir" ]] || mkdir -p "$dir" 2>/dev/null
  print -r -- "$code" >| "$dir/last-status" 2>/dev/null
  fc -ln -1 2>/dev/null | sed 's/^[[:space:]]*//' >| "$dir/last-cmd" 2>/dev/null
}
autoload -Uz add-zsh-hook 2>/dev/null
add-zsh-hook precmd _opsforge_track_last 2>/dev/null

# Install the widget only in an interactive shell with ZLE.
if [[ -o interactive ]] && zle -l >/dev/null 2>&1; then
  zle -N _opsforge_help_widget
  bindkey '?' _opsforge_help_widget
fi
