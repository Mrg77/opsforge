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
# zero-subprocess prefilter — only commands whose words could match a rule
# reach the Go binary. The pattern is derived ONCE, at shell load, from the
# ACTIVE policy (built-in or your guards.yaml) via `opsforge guard
# prefilter`, so a custom rule (e.g. on `terraform import`) is never
# silently skipped by a stale hard-coded verb list. Override with
# OPSFORGE_GUARD_PREFILTER (a zsh extended-glob alternation like
# '(kubectl|terraform|helm)').
if [[ -z "$OPSFORGE_GUARD_PREFILTER" ]] && (( $+commands[opsforge] )); then
  OPSFORGE_GUARD_PREFILTER="$(opsforge guard prefilter 2>/dev/null)"
fi
# Fallback if the binary was unavailable or the policy was empty.
: ${OPSFORGE_GUARD_PREFILTER:='(kubectl|helm|terraform|kubens|kubectx|argocd|flux|k)'}

# _opsforge_looks_dangerous is a cheap, in-shell gate: true only when a word
# of the buffer appears in the policy-derived prefilter, so most commands
# skip the Go call entirely.
_opsforge_looks_dangerous() {
  setopt localoptions extendedglob
  local buf="${1:l}"   # lowercase, matching the lowercase prefilter terms
  [[ "$buf" == (*[^a-z0-9])#${~OPSFORGE_GUARD_PREFILTER}(|[^a-z0-9]*) ]]
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
