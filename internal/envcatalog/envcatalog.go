// Package envcatalog is the single source of truth for the DevOps environment
// variables opsforge knows about — the standard vars of the tools it manages
// (AWS, GCP, Azure, Kubernetes, Terraform, Vault, Docker, Git/GitHub, proxies).
//
// It powers two completions that must agree:
//   - `export <TAB>` in the zsh shell layer (see modules/env-completion.zsh)
//   - `opsforge env set <TAB>` (cobra ValidArgsFunction, see cmd/env.go)
//
// The zsh module keeps its own copy of the names for a zero-subprocess prompt;
// this Go list is what the CLI itself uses. Keep the two in sync — the test in
// envcatalog_test.go guards the shape here, and the shell module is validated
// separately. Curated on purpose (the vars a DevOps exports by hand), not
// exhaustive: a wall of rarely-set vars would bury the useful ones.
package envcatalog

// Var is a known environment variable: its name and a one-line description.
type Var struct {
	Name string
	Desc string
	// Secret marks a variable whose VALUE is sensitive (a key/token/password).
	// Value completion is never offered for these.
	Secret bool
}

// Vars is the curated catalog, grouped by tool in declaration order.
var Vars = []Var{
	// AWS
	{"AWS_ACCESS_KEY_ID", "AWS access key ID", false},
	{"AWS_SECRET_ACCESS_KEY", "AWS secret access key", true},
	{"AWS_SESSION_TOKEN", "AWS temporary session token", true},
	{"AWS_REGION", "Default AWS region", false},
	{"AWS_DEFAULT_REGION", "Default AWS region (SDK/CLI fallback)", false},
	{"AWS_PROFILE", "Named profile from ~/.aws/config", false},
	{"AWS_VAULT", "Active aws-vault profile", false},
	{"AWS_ENDPOINT_URL", "Override the AWS API endpoint (e.g. LocalStack)", false},
	// GCP
	{"GOOGLE_APPLICATION_CREDENTIALS", "Path to a GCP service-account key file", false},
	{"GOOGLE_CLOUD_PROJECT", "Default GCP project ID", false},
	{"CLOUDSDK_CORE_PROJECT", "gcloud default project", false},
	// Azure
	{"AZURE_SUBSCRIPTION_ID", "Azure subscription ID", false},
	{"AZURE_TENANT_ID", "Azure tenant (directory) ID", false},
	{"AZURE_CLIENT_ID", "Azure service-principal client ID", false},
	{"AZURE_CLIENT_SECRET", "Azure service-principal secret", true},
	// Kubernetes
	{"KUBECONFIG", "Path(s) to kubeconfig file(s)", false},
	{"KUBE_EDITOR", "Editor for `kubectl edit`", false},
	{"K9S_CONFIG_DIR", "k9s config directory", false},
	// Terraform / OpenTofu
	{"TF_LOG", "Terraform log level (TRACE/DEBUG/INFO/WARN/ERROR)", false},
	{"TF_LOG_PATH", "File to write Terraform logs to", false},
	{"TF_WORKSPACE", "Terraform workspace to select", false},
	{"TF_DATA_DIR", "Terraform working dir (default .terraform)", false},
	{"TF_CLI_ARGS", "Extra args prepended to every terraform command", false},
	// Vault
	{"VAULT_ADDR", "HashiCorp Vault server address", false},
	{"VAULT_TOKEN", "Vault auth token", true},
	{"VAULT_NAMESPACE", "Vault Enterprise namespace", false},
	{"VAULT_SKIP_VERIFY", "Skip TLS verification (1 to enable — unsafe)", false},
	// Docker
	{"DOCKER_HOST", "Docker daemon socket/host", false},
	{"DOCKER_CONTEXT", "Active docker context", false},
	{"DOCKER_BUILDKIT", "Enable BuildKit (1) or not (0)", false},
	{"DOCKER_DEFAULT_PLATFORM", "Default build/run platform (e.g. linux/amd64)", false},
	// Git / GitHub / CI
	{"GITHUB_TOKEN", "GitHub API/CLI token", true},
	{"GH_TOKEN", "GitHub CLI token", true},
	{"GITLAB_TOKEN", "GitLab API token", true},
	{"GIT_SSH_COMMAND", "SSH command git uses (e.g. a specific key)", false},
	// Misc common
	{"HTTP_PROXY", "HTTP proxy URL", false},
	{"HTTPS_PROXY", "HTTPS proxy URL", false},
	{"NO_PROXY", "Hosts that bypass the proxy", false},
	{"EDITOR", "Default editor", false},
	{"PAGER", "Default pager", false},
}

// Names returns just the variable names, in catalog order.
func Names() []string {
	out := make([]string, len(Vars))
	for i, v := range Vars {
		out[i] = v.Name
	}
	return out
}

// Completions returns "NAME\tdescription" entries for cobra's shell completion,
// which renders the tab-separated description alongside each candidate.
func Completions() []string {
	out := make([]string, len(Vars))
	for i, v := range Vars {
		out[i] = v.Name + "\t" + v.Desc
	}
	return out
}
