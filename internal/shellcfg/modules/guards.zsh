# opsforge guards — policy-as-code for destructive commands.
#
# Implemented as an accept-line ZLE widget: when you press Enter, the
# buffer is inspected before it runs. This is the only reliable place to
# *cancel* a command in zsh (a preexec hook cannot stop execution).
#
# The policy itself lives in ~/.config/opsforge/guards.yaml (or a built-in
# default). The decision is made by `opsforge guard check`, which is fast
# and reads the context passively — it NEVER invokes kubectl, so it can't
# trigger an OIDC browser login on machines where kubectl is a cloud-SDK
# dispatcher.
#
# Set OPSFORGE_GUARDS=0 to disable for a session.

# PERF: the widget runs on every command. To stay cheap we first do a
# zero-subprocess prefilter — only commands mentioning a potentially
# destructive verb reach the Go binary. Tune the list with
# OPSFORGE_GUARD_PREFILTER (a zsh extended-glob alternation).
: ${OPSFORGE_GUARD_PREFILTER:='(kubectl|helm|terraform|kubens|kubectx|k) *'}

# _opsforge_looks_dangerous is a cheap, in-shell gate: true only when the
# buffer might match a rule, so most commands skip the Go call entirely.
_opsforge_looks_dangerous() {
  setopt localoptions extendedglob
  local buf="$1"
  case "$buf" in
    *delete*|*destroy*|*drain*|*cordon*|*apply*|*replace*|*uninstall*|*rollback*)
      return 0 ;;
  esac
  return 1
}

_opsforge_accept_line() {
  if [[ "$OPSFORGE_GUARDS" != "0" ]] \
     && _opsforge_looks_dangerous "$BUFFER" \
     && (( $+commands[opsforge] )); then
    local reply action message
    # `guard check` reads the current context itself (passively). Output is
    # "action" or "action|message".
    reply=$(opsforge guard check "$BUFFER" 2>/dev/null)
    action="${reply%%|*}"
    message="${reply#*|}"
    [[ "$message" == "$action" ]] && message=""

    case "$action" in
      deny)
        zle -M ""
        print -P "\n%F{red}%B✗  Blocked by opsforge guard%b%f"
        [[ -n "$message" ]] && print -P "%F{red}   ${message}%f"
        print -P "%F{yellow}   ${BUFFER}%f"
        print -P "%F{242}   (disable guards for this session with OPSFORGE_GUARDS=0)%f"
        BUFFER=""
        zle reset-prompt
        return
        ;;
      warn)
        zle -M ""
        print -P "\n%F{yellow}%B⚠  ${message:-opsforge guard}%b%f"
        # warn does not stop the command; fall through to run it.
        ;;
      confirm)
        zle -M ""
        print -P "\n%F{red}%B⚠  opsforge guard%b%f"
        [[ -n "$message" ]] && print -P "%F{red}   ${message}%f"
        print -P "%F{yellow}   ${BUFFER}%f"
        print -P "%F{242}   (to skip guards this session: OPSFORGE_GUARDS=0)%f"
        print -n "Type 'yes' to run this: "
        local answer
        read -r answer
        if [[ "$answer" != "yes" ]]; then
          print -P "%F{red}Aborted by opsforge guard.%f"
          BUFFER=""
          zle reset-prompt
          return
        fi
        ;;
    esac
  fi
  zle .accept-line
}

# Only install the widget in an interactive shell with ZLE available.
if [[ -o interactive ]] && zle -l >/dev/null 2>&1; then
  zle -N accept-line _opsforge_accept_line
fi
