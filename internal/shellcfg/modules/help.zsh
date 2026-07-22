# opsforge inline help — press "?" at the end of a command line to see
# what it does, without leaving your line. It runs the command's native
# --help, colorized and paginated, then restores your prompt so you can
# keep typing. Disable with OPSFORGE_HELP=0.
#
# Example: type "kubectl get " then "?"  → shows `kubectl get --help`.

[[ "$OPSFORGE_HELP" == "0" ]] && return

# _opsforge_help_widget is bound to "?". It only intercepts when "?" is
# typed at the very end of a non-empty line whose first word is a real
# command; otherwise it inserts a literal "?" so globbing/other uses are
# unaffected.
_opsforge_help_widget() {
  # Only act at end of line, on a non-empty buffer.
  if [[ $CURSOR -ne ${#BUFFER} || -z "${BUFFER// /}" ]]; then
    zle self-insert
    return
  fi

  # First word must be an executable command we can ask for help.
  local cmd="${${(z)BUFFER}[1]}"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    zle self-insert
    return
  fi

  # Build the help target from the whole current line. The '?' that
  # triggered this widget has NOT been inserted into $BUFFER yet, so the
  # buffer already holds exactly the command the user wants help for —
  # e.g. "kubectl get node" -> kubectl get node --help. We keep every
  # token (trailing spaces are just dropped by word-splitting).
  local -a parts
  parts=(${(z)BUFFER})

  # A compact, colored title so it's obvious which command this is for.
  print
  print -P "%F{cyan}%B❯ ${parts[*]} --help%b%f"

  {
    # Neutralize KUBECONFIG so a help probe can never trigger cloud auth
    # (--help never needs a cluster). /dev/null is an unreadable, empty
    # kubeconfig, so no exec credential plugin can be discovered.
    KUBECONFIG=/dev/null "${parts[@]}" --help 2>&1 \
      || KUBECONFIG=/dev/null "${parts[@]}" help 2>&1
  } | _opsforge_help_colorize | _opsforge_help_page
  zle reset-prompt
}

# _opsforge_help_colorize adds light ANSI coloring to a --help stream:
# section headers bold, example comments green, flags yellow. Works for
# the common Cobra/kubectl/docker help layout without needing bat.
_opsforge_help_colorize() {
  awk '
    /^[A-Za-z][A-Za-z &-]*:[[:space:]]*$/ { printf "\033[1;36m%s\033[0m\n", $0; next }  # "Usage:", "Options:", "Examples:"
    /^[[:space:]]*#/                      { printf "\033[32m%s\033[0m\n", $0; next }    # example comments
    /^[[:space:]]*--?[A-Za-z]/            { printf "\033[33m%s\033[0m\n", $0; next }    # flag lines
    { print }
  '
}

# _opsforge_help_page pages long output cleanly: no pager escape junk, and
# when the help fits on one screen it just prints and returns (so short
# help never traps you in `less`). Press q to leave long help.
_opsforge_help_page() {
  if command -v less >/dev/null 2>&1; then
    # -F: quit if it fits on one screen; -R: keep our colors; -X: don't
    # clear the screen; -Q: quiet. Prompt tells the user how to leave.
    LESS='-FRXQ' less --prompt='  (q to close help) ' 2>/dev/null || cat
  else
    cat
  fi
}

# Install the widget only in an interactive shell with ZLE.
if [[ -o interactive ]] && zle -l >/dev/null 2>&1; then
  zle -N _opsforge_help_widget
  bindkey '?' _opsforge_help_widget
fi
