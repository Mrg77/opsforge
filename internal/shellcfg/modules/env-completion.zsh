# opsforge env-var completion — `export <TAB>` knows the DevOps ecosystem.
#
# Native zsh only completes variables that ALREADY exist in the session, so a
# var you've never exported (say AWS_SECRET_ACCESS_KEY) is never offered — you
# retype it in full every time. This layer teaches `export`/`typeset`/`declare`
# the standard variables of the tools opsforge manages (AWS, GCP, Kubernetes,
# Terraform, Vault, Docker, GitHub…), each with a one-line description, so a
# variable you've never set is one TAB away.
#
# Scope note: this completes variable NAMES. Completing an assignment's VALUE
# (`AWS_REGION=<TAB>`) goes through a different zsh code path (the `-value-`
# context, not the `export` command completer), so it's handled separately —
# see _opsforge_env_value below, wired via the value context. Value completion
# is intentionally limited to enumerable, NON-secret sources (profiles from
# ~/.aws/config, a region list, kubeconfig files); a secret's value is never
# suggested or read from disk.
#
# Disable with OPSFORGE_ENVCOMP=0.

[[ "$OPSFORGE_ENVCOMP" == "0" ]] && return

# --- the known-variable catalog: name:description, grouped by tool ---------
# Kept intentionally curated (the vars a DevOps actually exports by hand), not
# exhaustive — a wall of 200 rarely-set vars would bury the useful ones.
_opsforge_env_vars=(
  # AWS
  'AWS_ACCESS_KEY_ID:AWS access key ID'
  'AWS_SECRET_ACCESS_KEY:AWS secret access key (sensitive)'
  'AWS_SESSION_TOKEN:AWS temporary session token (sensitive)'
  'AWS_REGION:Default AWS region'
  'AWS_DEFAULT_REGION:Default AWS region (SDK/CLI fallback)'
  'AWS_PROFILE:Named profile from ~/.aws/config'
  'AWS_VAULT:Active aws-vault profile'
  'AWS_ENDPOINT_URL:Override the AWS API endpoint (e.g. LocalStack)'
  # GCP
  'GOOGLE_APPLICATION_CREDENTIALS:Path to a GCP service-account key file'
  'GOOGLE_CLOUD_PROJECT:Default GCP project ID'
  'CLOUDSDK_CORE_PROJECT:gcloud default project'
  # Azure
  'AZURE_SUBSCRIPTION_ID:Azure subscription ID'
  'AZURE_TENANT_ID:Azure tenant (directory) ID'
  'AZURE_CLIENT_ID:Azure service-principal client ID'
  'AZURE_CLIENT_SECRET:Azure service-principal secret (sensitive)'
  # Kubernetes
  'KUBECONFIG:Path(s) to kubeconfig file(s)'
  'KUBE_EDITOR:Editor for `kubectl edit`'
  'K9S_CONFIG_DIR:k9s config directory'
  # Terraform / OpenTofu
  'TF_LOG:Terraform log level (TRACE/DEBUG/INFO/WARN/ERROR)'
  'TF_LOG_PATH:File to write Terraform logs to'
  'TF_WORKSPACE:Terraform workspace to select'
  'TF_DATA_DIR:Terraform working dir (default .terraform)'
  'TF_VAR_:Prefix for a Terraform input variable (TF_VAR_name)'
  'TF_CLI_ARGS:Extra args prepended to every terraform command'
  # Vault
  'VAULT_ADDR:HashiCorp Vault server address'
  'VAULT_TOKEN:Vault auth token (sensitive)'
  'VAULT_NAMESPACE:Vault Enterprise namespace'
  'VAULT_SKIP_VERIFY:Skip TLS verification (1 to enable — unsafe)'
  # Docker
  'DOCKER_HOST:Docker daemon socket/host'
  'DOCKER_CONTEXT:Active docker context'
  'DOCKER_BUILDKIT:Enable BuildKit (1) or not (0)'
  'DOCKER_DEFAULT_PLATFORM:Default build/run platform (e.g. linux/amd64)'
  # Git / GitHub / CI
  'GITHUB_TOKEN:GitHub API/CLI token (sensitive)'
  'GH_TOKEN:GitHub CLI token (sensitive)'
  'GITLAB_TOKEN:GitLab API token (sensitive)'
  'GIT_SSH_COMMAND:SSH command git uses (e.g. a specific key)'
  # Misc common
  'HTTP_PROXY:HTTP proxy URL'
  'HTTPS_PROXY:HTTPS proxy URL'
  'NO_PROXY:Hosts that bypass the proxy'
  'EDITOR:Default editor'
  'PAGER:Default pager'
)

# Secrets have NO value completer below (keys, tokens, passwords, *_SECRET):
# there's nothing safe to suggest and we won't read one from disk. Absence,
# not a filter, is what keeps a secret's value from ever being offered.

# --- value completers for the safe, enumerable variables -------------------
# Each reads only local, non-secret config. Silent no-op when the source is
# absent, so completion never errors on a machine without that tool.

_opsforge_val_aws_profile() {
  local cfg="${AWS_CONFIG_FILE:-$HOME/.aws/config}"
  [[ -r "$cfg" ]] || return
  local -a profiles
  # Lines like "[profile prod]" or "[default]".
  profiles=(${(f)"$(sed -n 's/^\[profile \(.*\)\]$/\1/p; s/^\[\(default\)\]$/\1/p' "$cfg" 2>/dev/null)"})
  (( ${#profiles} )) && _describe -t aws-profiles 'AWS profile' profiles
}

_opsforge_val_aws_region() {
  local -a regions
  regions=(
    us-east-1 us-east-2 us-west-1 us-west-2
    eu-west-1 eu-west-2 eu-west-3 eu-central-1 eu-north-1 eu-south-1
    ap-south-1 ap-southeast-1 ap-southeast-2 ap-northeast-1 ap-northeast-2
    ca-central-1 sa-east-1
  )
  _describe -t aws-regions 'AWS region' regions
}

_opsforge_val_kubeconfig() { _files }

_opsforge_val_tf_log() {
  local -a levels=(TRACE DEBUG INFO WARN ERROR)
  _describe -t tf-log 'Terraform log level' levels
}

_opsforge_val_docker_platform() {
  local -a plats=(linux/amd64 linux/arm64 linux/arm/v7)
  _describe -t platforms 'platform' plats
}

# --- NAME completion: wired onto export/typeset/declare -------------------
# Offers the known catalog (with descriptions) AND the live variables,
# appending '=' so the next TAB moves on to the value. zsh de-dupes, so a var
# that's both known and live shows once.
_opsforge_export() {
  _describe -t opsforge-env 'DevOps env var' _opsforge_env_vars -S '=' -q
  _parameters -S '=' -q 2>/dev/null
}
compdef _opsforge_export export
compdef _opsforge_export typeset
compdef _opsforge_export declare

# --- VALUE completion: wired onto the `-value-,NAME,-default-` context -----
# An assignment's value (`AWS_REGION=<TAB>`) goes through zsh's value context,
# NOT the `export` command completer — so it's bound per-variable here. Only
# the enumerable, non-secret vars get a completer; every secret is deliberately
# absent, so a key/token's value is never suggested.
compdef _opsforge_val_aws_profile      '-value-,AWS_PROFILE,-default-'
compdef _opsforge_val_aws_region       '-value-,AWS_REGION,-default-'
compdef _opsforge_val_aws_region       '-value-,AWS_DEFAULT_REGION,-default-'
compdef _opsforge_val_kubeconfig       '-value-,KUBECONFIG,-default-'
compdef _opsforge_val_kubeconfig       '-value-,GOOGLE_APPLICATION_CREDENTIALS,-default-'
compdef _opsforge_val_tf_log           '-value-,TF_LOG,-default-'
compdef _opsforge_val_docker_platform  '-value-,DOCKER_DEFAULT_PLATFORM,-default-'
