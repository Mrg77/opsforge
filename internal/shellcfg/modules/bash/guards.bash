# opsforge guards — policy-as-code for destructive commands (bash).
#
# HONEST LIMITATION: bash has no clean way to cancel a command before it runs
# the way zsh (accept-line widget) and fish (bind enter) do. opsforge binds
# Enter to a function via `bind -x` and re-runs the line itself with `eval`,
# which lets it confirm/deny — but at a cost: this path doesn't preserve bash's
# job control, so backgrounding with a trailing `&`, `fg`/`bg`, and some
# multi-line constructs behave differently than an un-guarded bash. If that
# matters to you, or `bind -x` isn't available (very old bash), the guard falls
# back to a DEBUG-trap WARNING that can't block — it prints the policy decision
# but the command still runs.
#
# For a guarantee that a destructive command is stopped, use zsh or fish.
# Real prod protection belongs elsewhere anyway (prevent_destroy, separate
# accounts, CI approvals) — guards are a safety net, not a security boundary.
#
# Set OPSFORGE_GUARDS=0 to disable for a session.

# Only meaningful in an interactive shell.
case $- in *i*) ;; *) return 0 2>/dev/null || true ;; esac

# Derive the prefilter once from the ACTIVE policy (built-in or guards.yaml).
if [[ -z "$OPSFORGE_GUARD_PREFILTER" ]] && command -v opsforge >/dev/null 2>&1; then
  OPSFORGE_GUARD_PREFILTER="$(opsforge guard prefilter 2>/dev/null)"
fi
: "${OPSFORGE_GUARD_PREFILTER:=(kubectl|helm|terraform|kubens|kubectx|argocd|flux|k)}"

# _opsforge_looks_dangerous: a cheap, in-shell gate. The prefilter is a regex
# alternation; test the lowercased buffer against it so most commands skip the
# Go call entirely.
_opsforge_looks_dangerous() {
  local buf="${1,,}"
  [[ "$buf" =~ $OPSFORGE_GUARD_PREFILTER ]]
}

# _opsforge_guard_eval evaluates the policy and returns 0 to run, 1 to block.
# It prints the confirm/warn/deny UI to the terminal.
_opsforge_guard_eval() {
  local buf="$1"
  [[ "$OPSFORGE_GUARDS" == "0" ]] && return 0
  command -v opsforge >/dev/null 2>&1 || return 0
  [[ -z "$buf" ]] && return 0
  _opsforge_looks_dangerous "$buf" || return 0

  local reply action message
  reply=$(opsforge guard check "$buf" 2>/dev/null)
  action="${reply%%|*}"
  message="${reply#*|}"
  [[ "$message" == "$action" ]] && message=""

  case "$action" in
    deny)
      printf '\n\033[1;31m✗  Blocked by opsforge guard\033[0m\n'
      [[ -n "$message" ]] && printf '\033[31m   %s\033[0m\n' "$message"
      printf '\033[33m   %s\033[0m\n' "$buf"
      printf '\033[2m   (disable guards for this session with OPSFORGE_GUARDS=0)\033[0m\n'
      return 1
      ;;
    warn)
      printf '\n\033[1;33m⚠  %s\033[0m\n' "${message:-opsforge guard}"
      return 0
      ;;
    confirm)
      printf '\n\033[1;31m⚠  opsforge guard\033[0m\n'
      [[ -n "$message" ]] && printf '\033[31m   %s\033[0m\n' "$message"
      printf '\033[33m   %s\033[0m\n' "$buf"
      printf '\033[2m   (to skip guards this session: OPSFORGE_GUARDS=0)\033[0m\n'
      local answer
      read -r -p "Type 'yes' to run this: " answer
      if [[ "$answer" != "yes" ]]; then
        printf '\033[31mAborted by opsforge guard.\033[0m\n'
        return 1
      fi
      return 0
      ;;
  esac
  return 0
}

# Preferred path: rebind Enter so we can actually cancel. `bind -x` runs the
# hook with the typed line in $READLINE_LINE; we evaluate, then either clear the
# line (block) or run it ourselves (allow).
_opsforge_accept_line() {
  local buf="$READLINE_LINE"
  if _opsforge_guard_eval "$buf"; then
    if [[ -n "$buf" ]]; then
      history -s "$buf"          # keep it in history as if typed normally
      printf '\n'
      eval "$buf"
    fi
  fi
  READLINE_LINE=""
  READLINE_POINT=0
}

# Fallback path: a DEBUG trap that can only WARN (bash can't block a simple
# command from DEBUG at the prompt). Used when bind -x isn't available.
_opsforge_debug_warn() {
  local cmd="$BASH_COMMAND"
  case "$cmd" in
    _opsforge_*|*OPSFORGE_*|trap\ *) return 0 ;;
  esac
  _opsforge_looks_dangerous "$cmd" || return 0
  local reply action message
  reply=$(opsforge guard check "$cmd" 2>/dev/null)
  action="${reply%%|*}"; message="${reply#*|}"
  [[ "$message" == "$action" ]] && message=""
  case "$action" in
    deny|confirm)
      printf '\033[1;33m⚠  opsforge guard: %s\033[0m\n' "${message:-this looks destructive on prod}" >&2
      printf '\033[2m   (bash can'\''t stop it here — use zsh/fish for a blocking guard)\033[0m\n' >&2
      ;;
  esac
  return 0
}

if bind -x '"\C-m": _opsforge_accept_line' 2>/dev/null; then
  : # blocking guard active via readline
elif command -v opsforge >/dev/null 2>&1; then
  # bind -x unavailable: degrade to a non-blocking warning.
  shopt -s extdebug 2>/dev/null
  trap '_opsforge_debug_warn' DEBUG
fi
