# opsforge inline help — press "?" at the end of a command line to see what it
# does, without leaving your line (bash). Runs the command's native --help,
# rendered cleanly (bat when available). Press "?" on an empty line for the
# opsforge cheat-sheet, "??" to explain the last command via AI. Disable with
# OPSFORGE_HELP=0.

case $- in *i*) ;; *) return 0 2>/dev/null || true ;; esac

if [[ "$OPSFORGE_HELP" != "0" ]]; then

  _opsforge_help_colorize() {
    awk '
      /^[A-Za-z][A-Za-z &-]*:[[:space:]]*$/ { printf "\033[1;36m%s\033[0m\n", $0; next }
      /^[[:space:]]*#/                      { printf "\033[32m%s\033[0m\n", $0; next }
      /^[[:space:]]*--?[A-Za-z]/            { printf "\033[33m%s\033[0m\n", $0; next }
      { print }
    '
  }

  _opsforge_help_panel() {
    if command -v opsforge >/dev/null 2>&1; then
      opsforge shell help 2>/dev/null && return
    fi
    printf '\033[1;38;5;212m  opsforge shell\033[0m — press ? for help, ?? to explain the last command\n'
  }

  _opsforge_help_render() {
    local title="$1" body="$2"
    local width="${COLUMNS:-80}"; (( width > 100 )) && width=100
    local rule; rule=$(printf '─%.0s' $(seq 1 "$width"))
    printf '\033[36m%s\033[0m\n' "$rule"
    printf '\033[1;36m  ❯ %s --help\033[0m\n' "$title"
    printf '\033[36m%s\033[0m\n' "$rule"
    if command -v bat >/dev/null 2>&1; then
      printf '%s\n' "$body" | bat --style=plain --language=man --color=always --paging=never 2>/dev/null \
        || printf '%s\n' "$body" | _opsforge_help_colorize
    else
      printf '%s\n' "$body" | _opsforge_help_colorize
    fi
  }

  # Bound to "?" via bind -x. $READLINE_LINE holds what's typed so far.
  _opsforge_help_widget() {
    local buf="$READLINE_LINE"
    if [[ -z "$buf" ]]; then
      printf '\n'; _opsforge_help_panel; return
    fi
    if [[ "$buf" == "?" ]]; then
      READLINE_LINE=""; printf '\n'; opsforge explain --last; return
    fi
    # "?" mid-line or after a non-command: insert a literal "?".
    local first="${buf%% *}"
    if ! command -v "$first" >/dev/null 2>&1; then
      READLINE_LINE="${buf}?"; READLINE_POINT=$(( ${#READLINE_LINE} )); return
    fi
    local help
    help=$(KUBECONFIG=/dev/null $buf --help 2>&1)
    [[ -z "$help" ]] && help=$(KUBECONFIG=/dev/null $buf help 2>&1)
    printf '\n'
    if command -v less >/dev/null 2>&1; then
      _opsforge_help_render "$buf" "$help" | LESS='-FRXQ' less --prompt='  q to close help ' 2>/dev/null
    else
      _opsforge_help_render "$buf" "$help"
    fi
  }

  # Track the last command + status for `??` / `opsforge explain --last`.
  _opsforge_track_last() {
    local code=$?
    local dir="$HOME/.cache/opsforge"
    [[ -d "$dir" ]] || mkdir -p "$dir" 2>/dev/null
    printf '%s\n' "$code" > "$dir/last-status" 2>/dev/null
    fc -ln -1 2>/dev/null | sed 's/^[[:space:]]*//' > "$dir/last-cmd" 2>/dev/null
    return $code
  }
  case "$PROMPT_COMMAND" in
    *_opsforge_track_last*) ;;
    "") PROMPT_COMMAND="_opsforge_track_last" ;;
    *)  PROMPT_COMMAND="_opsforge_track_last; $PROMPT_COMMAND" ;;
  esac

  bind -x '"?": _opsforge_help_widget' 2>/dev/null || true
fi
