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

  # Render below the prompt, keep the line intact.
  print                     # newline after the current line
  {
    # Neutralize KUBECONFIG so a help probe can never trigger cloud auth
    # (--help never needs a cluster). /dev/null is an unreadable, empty
    # kubeconfig, so no exec credential plugin can be discovered.
    KUBECONFIG=/dev/null "${parts[@]}" --help 2>&1 \
      || KUBECONFIG=/dev/null "${parts[@]}" help 2>&1
  } | _opsforge_help_pager
  zle reset-prompt
}

# _opsforge_help_pager colorizes lightly and pages long help through the
# user's pager (bat if available for syntax coloring, else less -R).
_opsforge_help_pager() {
  if command -v bat >/dev/null 2>&1; then
    bat --style=plain --language=help --paging=auto --color=always 2>/dev/null \
      || bat --style=plain --paging=auto 2>/dev/null || cat
  elif command -v less >/dev/null 2>&1; then
    less -FRX
  else
    cat
  fi
}

# Install the widget only in an interactive shell with ZLE.
if [[ -o interactive ]] && zle -l >/dev/null 2>&1; then
  zle -N _opsforge_help_widget
  bindkey '?' _opsforge_help_widget
fi
