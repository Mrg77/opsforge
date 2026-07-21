# opsforge prompt — optional context segments on the right prompt.
#
# By default ONLY the kube-context segment is on, and only when it can be
# read cheaply from the kubeconfig file. Cloud and terraform segments are
# opt-in, because probing gcloud/aws on every prompt is slow and noisy.
#
# Toggle segments with env vars (set in ~/.zshrc, before the eval line):
#   OPSFORGE_PROMPT_KUBE=0    # disable the kube segment
#   OPSFORGE_PROMPT_CLOUD=1   # enable the cloud segment (off by default)
#   OPSFORGE_PROMPT_TF=1      # enable the terraform segment (off by default)
#
# The kube segment NEVER runs kubectl (it reads the kubeconfig file), so
# it can't trigger an OIDC browser login.

# --- kube context: cluster name, red when it looks like prod ---
_opsforge_kube_segment() {
  [[ "$OPSFORGE_PROMPT_KUBE" == "0" ]] && return
  local cfg="${KUBECONFIG%%:*}"
  [[ -z "$cfg" ]] && cfg="$HOME/.kube/config"
  [[ -r "$cfg" ]] || return
  local ctx
  ctx=$(grep -m1 '^current-context:' "$cfg" 2>/dev/null | sed 's/current-context:[[:space:]]*//; s/["'\'']//g')
  [[ -z "$ctx" ]] && return
  local color="%F{cyan}"
  if [[ "$ctx" == *prod* || "$ctx" == *production* ]]; then
    color="%F{red}%B"
  fi
  print -n "${color}⎈ ${ctx}%b%f"
}

# --- cloud account: OFF by default. Only reads env vars, never probes
#     gcloud/aws (which would be slow and surprising). Opt in with
#     OPSFORGE_PROMPT_CLOUD=1. ---
_opsforge_cloud_segment() {
  [[ "$OPSFORGE_PROMPT_CLOUD" == "1" ]] || return
  if [[ -n "$AWS_VAULT" ]]; then
    print -n " %F{yellow}☁ aws:${AWS_VAULT}%f"
  elif [[ -n "$AWS_PROFILE" ]]; then
    print -n " %F{yellow}☁ aws:${AWS_PROFILE}%f"
  fi
}

# --- terraform workspace: OFF by default. Opt in with OPSFORGE_PROMPT_TF=1. ---
_opsforge_tf_segment() {
  [[ "$OPSFORGE_PROMPT_TF" == "1" ]] || return
  [[ -d .terraform ]] || return
  local ws
  ws=$(cat .terraform/environment 2>/dev/null)
  [[ -n "$ws" && "$ws" != "default" ]] && print -n " %F{magenta}⧉ tf:${ws}%f"
}

_opsforge_rprompt() {
  print -n "$(_opsforge_kube_segment)$(_opsforge_cloud_segment)$(_opsforge_tf_segment)"
}

# Only claim RPROMPT if the user (or another theme) hasn't already.
if [[ -z "$RPROMPT" ]]; then
  setopt PROMPT_SUBST
  RPROMPT='$(_opsforge_rprompt)'
fi
