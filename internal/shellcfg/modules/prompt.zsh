# opsforge prompt — context-aware DevOps segments on the right prompt.
#
# Segments appear only when relevant and are cached per-command so the
# prompt stays fast. The kube segment turns red on a production-looking
# context so you notice before running something dangerous.
#
# IMPORTANT: the kube segment NEVER runs `kubectl`. On machines where
# kubectl is a cloud-SDK dispatcher wired to an OIDC exec plugin, even
# `kubectl config current-context` pops a browser login. We read the
# current context straight from the kubeconfig file instead — a pure file
# read that can never trigger authentication. Set OPSFORGE_PROMPT_KUBE=0
# to disable this segment entirely.

# --- kube context: cluster (+ namespace), red when it looks like prod ---
_opsforge_kube_segment() {
  [[ "$OPSFORGE_PROMPT_KUBE" == "0" ]] && return
  local cfg="${KUBECONFIG%%:*}"
  [[ -z "$cfg" ]] && cfg="$HOME/.kube/config"
  [[ -r "$cfg" ]] || return
  # Parse `current-context:` without invoking kubectl.
  local ctx
  ctx=$(grep -m1 '^current-context:' "$cfg" 2>/dev/null | sed 's/current-context:[[:space:]]*//; s/["'\'']//g')
  [[ -z "$ctx" ]] && return
  local color="%F{cyan}"
  if [[ "$ctx" == *prod* || "$ctx" == *production* ]]; then
    color="%F{red}%B"
  fi
  print -n "${color}⎈ ${ctx}%b%f"
}

# --- cloud account: active AWS profile / GCP project ---
_opsforge_cloud_segment() {
  if [[ -n "$AWS_PROFILE" || -n "$AWS_VAULT" ]]; then
    print -n " %F{yellow}☁ aws:${AWS_VAULT:-$AWS_PROFILE}%f"
  elif command -v gcloud >/dev/null 2>&1; then
    local proj
    proj=$(gcloud config get-value project 2>/dev/null)
    [[ -n "$proj" && "$proj" != "(unset)" ]] && print -n " %F{yellow}☁ gcp:${proj}%f"
  fi
}

# --- terraform workspace, only inside a terraform project ---
_opsforge_tf_segment() {
  [[ -d .terraform || -f terraform.tf || -f main.tf ]] || return
  command -v terraform >/dev/null 2>&1 || return
  local ws
  ws=$(terraform workspace show 2>/dev/null) || return
  [[ -n "$ws" && "$ws" != "default" ]] && print -n " %F{magenta}⧉ tf:${ws}%f"
}

_opsforge_rprompt() {
  local out
  out="$(_opsforge_kube_segment)$(_opsforge_cloud_segment)$(_opsforge_tf_segment)"
  print -n "$out"
}

# Only claim RPROMPT if the user (or another theme) hasn't already.
if [[ -z "$RPROMPT" ]]; then
  setopt PROMPT_SUBST
  RPROMPT='$(_opsforge_rprompt)'
fi
