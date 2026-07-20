# opsforge prompt — context-aware DevOps segments on the right prompt.
#
# Segments appear only when relevant and are cached per-command so the
# prompt stays fast. The kube segment turns red on a production-looking
# context so you notice before running something dangerous.

# --- kube context: cluster:namespace, red when it looks like prod ---
_opsforge_kube_segment() {
  command -v kubectl >/dev/null 2>&1 || return
  local ctx ns
  ctx=$(kubectl config current-context 2>/dev/null) || return
  [[ -z "$ctx" ]] && return
  ns=$(kubectl config view --minify -o 'jsonpath={..namespace}' 2>/dev/null)
  [[ -z "$ns" ]] && ns="default"
  local color="%F{cyan}"
  # Heuristic: anything that reads like prod is flagged red.
  if [[ "$ctx" == *prod* || "$ctx" == *production* || "$ns" == *prod* ]]; then
    color="%F{red}%B"
  fi
  print -n "${color}⎈ ${ctx}:${ns}%b%f"
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
