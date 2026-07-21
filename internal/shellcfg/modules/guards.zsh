# opsforge guards — confirm before destructive commands run against a
# production-looking Kubernetes context.
#
# Implemented as an accept-line ZLE widget: when you press Enter, the
# buffer is inspected before it runs. This is the only reliable place to
# *cancel* a command in zsh (a preexec hook cannot stop execution).
# Set OPSFORGE_GUARDS=0 to disable for a session.
#
# Like the prompt, this NEVER invokes kubectl — it reads current-context
# from the kubeconfig file, so it can't trigger an OIDC browser login on
# machines where kubectl is a cloud-SDK dispatcher.

# _opsforge_current_ctx echoes the current kube context by reading the
# kubeconfig file directly (no kubectl invocation).
_opsforge_current_ctx() {
  local cfg="${KUBECONFIG%%:*}"
  [[ -z "$cfg" ]] && cfg="$HOME/.kube/config"
  [[ -r "$cfg" ]] || return 1
  grep -m1 '^current-context:' "$cfg" 2>/dev/null \
    | sed 's/current-context:[[:space:]]*//; s/["'\'']//g'
}

_opsforge_ctx_is_prod() {
  local ctx
  ctx=$(_opsforge_current_ctx) || return 1
  [[ "$ctx" == *prod* || "$ctx" == *production* ]]
}

# Commands destructive enough to confirm in prod.
_opsforge_is_destructive() {
  local cmd="$1"
  case "$cmd" in
    *"kubectl delete"*|*"kubectl drain"*|*"kubectl cordon"*) return 0 ;;
    *"kubectl apply"*|*"kubectl replace"*)                   return 0 ;;
    *"helm uninstall"*|*"helm delete"*|*"helm rollback"*)    return 0 ;;
    *"terraform destroy"*|*"terraform apply"*)               return 0 ;;
    *) return 1 ;;
  esac
}

_opsforge_accept_line() {
  if [[ "$OPSFORGE_GUARDS" != "0" ]] \
     && _opsforge_is_destructive "$BUFFER" \
     && _opsforge_ctx_is_prod; then
    local ctx
    ctx=$(_opsforge_current_ctx)
    zle -M ""
    print -P "\n%F{red}%B⚠  PRODUCTION context (${ctx})%b%f"
    print -P "%F{yellow}   ${BUFFER}%f"
    print -n "Type 'yes' to run this: "
    local answer
    read -r answer
    if [[ "$answer" != "yes" ]]; then
      print -P "%F{red}Aborted by opsforge guard.%f"
      # Discard the line without running it.
      BUFFER=""
      zle reset-prompt
      return
    fi
  fi
  zle .accept-line
}

# Only install the widget in an interactive shell with ZLE available.
if [[ -o interactive ]] && zle -l >/dev/null 2>&1; then
  zle -N accept-line _opsforge_accept_line
fi
