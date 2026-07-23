// Package families is the single source of truth for opsforge's "tool
// families" — the DevOps domains (Kubernetes, Terraform/IaC, Git…) that
// group the CLIs an engineer thinks of together.
//
// Before this package the same taxonomy was hard-coded in three diverging
// places (the history command, the guard policy, and the shell prefilter).
// Now history filtering, the built-in guard rules, and anything else that
// needs "which tools are 'kube'?" all derive from here, so adding a tool
// to a family is a one-line change in one file.
package families

// Family is a named group of CLIs, plus the destructive subcommands that
// make a command worth guarding on a production context.
type Family struct {
	// Key is the short name used on the CLI (`opsforge history kube`).
	Key string
	// Label is the human name shown in listings.
	Label string
	// Bins are the executables that belong to this family.
	Bins []string
	// DangerVerbs are the subcommands that mutate/destroy state; the
	// built-in guard policy confirms these on a prod context. Empty means
	// the family has no default guard (e.g. git, cloud read tools).
	DangerVerbs []string
}

// All is the ordered list of built-in families. Keep it the single place
// these tools are grouped.
var All = []Family{
	{
		Key: "kube", Label: "Kubernetes",
		Bins:        []string{"kubectl", "helm", "k9s", "kubectx", "kubens", "kustomize", "stern", "kubeseal", "flux", "argocd", "k", "kx", "kn"},
		DangerVerbs: []string{"delete", "drain", "cordon", "apply", "replace"},
	},
	{
		Key: "git", Label: "Git",
		Bins: []string{"git", "gh", "glab", "lazygit", "tig"},
		// push --force is destructive but hard to match by verb alone; left
		// to user rules rather than a noisy default.
	},
	{
		Key: "tf", Label: "Terraform / IaC",
		Bins:        []string{"terraform", "tofu", "terragrunt", "tflint", "terraform-docs", "tf"},
		DangerVerbs: []string{"destroy", "apply"},
	},
	{
		Key: "docker", Label: "Containers",
		Bins:        []string{"docker", "docker-compose", "podman", "nerdctl", "colima", "dc"},
		DangerVerbs: []string{"rm", "rmi", "prune", "kill"},
	},
	{
		Key: "cloud", Label: "Cloud CLIs",
		Bins: []string{"aws", "gcloud", "az", "doctl", "eksctl", "flyctl", "vercel"},
	},
	{
		Key: "ansible", Label: "Ansible",
		Bins: []string{"ansible", "ansible-playbook", "ansible-galaxy", "ansible-vault"},
	},
}

// ByKey returns the family with the given key, or ok=false.
func ByKey(key string) (Family, bool) {
	for _, f := range All {
		if f.Key == key {
			return f, true
		}
	}
	return Family{}, false
}

// Keys returns the family keys in order.
func Keys() []string {
	keys := make([]string, 0, len(All))
	for _, f := range All {
		keys = append(keys, f.Key)
	}
	return keys
}

// BinFamily maps every executable to its family key, for reverse lookups
// (e.g. "which family does `kubectl` belong to?"). Built once from All.
func BinFamily() map[string]string {
	m := map[string]string{}
	for _, f := range All {
		for _, b := range f.Bins {
			m[b] = f.Key
		}
	}
	return m
}
