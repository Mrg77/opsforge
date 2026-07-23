package shellcfg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPolicyMatchesLegacyBehavior(t *testing.T) {
	p := DefaultPolicy()
	cases := []struct {
		command, context string
		want             Action
	}{
		// Destructive on prod → confirm (as the old guards.zsh did).
		{"kubectl delete pod foo", "prod-eu-cluster", ActionConfirm},
		{"kubectl drain node-1", "gke_acme_production", ActionConfirm},
		{"kubectl apply -f m.yaml", "prod", ActionConfirm},
		{"helm uninstall app", "prod", ActionConfirm},
		{"terraform destroy", "prod", ActionConfirm},
		{"terraform apply", "production", ActionConfirm},
		// Same commands on a non-prod context → allow.
		{"kubectl delete pod foo", "staging", ActionAllow},
		{"terraform destroy", "dev", ActionAllow},
		{"terraform destroy", "", ActionAllow},
		// Non-destructive commands → allow even on prod.
		{"kubectl get pods", "prod", ActionAllow},
		{"ls -la", "prod", ActionAllow},
		{"terraform plan", "prod", ActionAllow},
	}
	for _, c := range cases {
		got := p.Evaluate(c.command, c.context).Action
		if got != c.want {
			t.Errorf("Evaluate(%q, %q) = %q, want %q", c.command, c.context, got, c.want)
		}
	}
}

func TestEvaluateFirstMatchWins(t *testing.T) {
	p := &GuardPolicy{
		Version: 1,
		Rules: []GuardRule{
			{Name: "deny ns delete", Match: GuardMatch{Command: "kubectl delete ns", Context: "prod"}, Action: ActionDeny, Message: "no"},
			{Name: "confirm any delete", Match: GuardMatch{Command: "kubectl delete", Context: "prod"}, Action: ActionConfirm},
		},
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	// The more specific deny rule comes first and must win.
	if d := p.Evaluate("kubectl delete ns team-a", "prod"); d.Action != ActionDeny || d.Rule != "deny ns delete" {
		t.Errorf("expected deny from first rule, got %+v", d)
	}
	// A generic delete falls through to the confirm rule.
	if d := p.Evaluate("kubectl delete pod x", "prod"); d.Action != ActionConfirm {
		t.Errorf("expected confirm from second rule, got %+v", d)
	}
}

func TestEvaluateReturnsMessageAndRule(t *testing.T) {
	p := DefaultPolicy()
	d := p.Evaluate("terraform destroy", "prod")
	if d.Rule == "" {
		t.Error("expected a matching rule name")
	}
	if d.Message == "" {
		t.Error("expected a message from the matching rule")
	}
}

func TestEmptyContextRuleMatchesAnyContext(t *testing.T) {
	p := &GuardPolicy{
		Version: 1,
		Rules: []GuardRule{
			{Name: "always confirm rm -rf", Match: GuardMatch{Command: "rm -rf /"}, Action: ActionConfirm},
		},
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if d := p.Evaluate("rm -rf / --no-preserve-root", ""); d.Action != ActionConfirm {
		t.Errorf("command-only rule should match any context, got %+v", d)
	}
	if d := p.Evaluate("rm -rf / --no-preserve-root", "prod"); d.Action != ActionConfirm {
		t.Errorf("command-only rule should match any context, got %+v", d)
	}
}

func TestEmptyPredicateRuleIsSkipped(t *testing.T) {
	// A rule that matches everything must not be allowed to silently block
	// the whole shell.
	p := &GuardPolicy{
		Version: 1,
		Rules:   []GuardRule{{Name: "bad catch-all", Action: ActionDeny}},
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if d := p.Evaluate("ls", "prod"); d.Action != ActionAllow {
		t.Errorf("empty-predicate rule must be skipped, got %+v", d)
	}
}

func TestParsePolicyValidYAML(t *testing.T) {
	yaml := `
version: 1
rules:
  - name: block prod destroy
    match:
      command: "terraform destroy"
      context: "prod"
    action: deny
    message: "nope"
`
	p, err := ParsePolicy([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Rules) != 1 {
		t.Fatalf("want 1 rule, got %d", len(p.Rules))
	}
	if d := p.Evaluate("terraform destroy", "prod-cluster"); d.Action != ActionDeny {
		t.Errorf("want deny, got %+v", d)
	}
}

func TestParsePolicyDefaultsActionToConfirm(t *testing.T) {
	yaml := `
version: 1
rules:
  - name: no action given
    match:
      command: "kubectl delete"
`
	p, err := ParsePolicy([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if p.Rules[0].Action != ActionConfirm {
		t.Errorf("missing action should default to confirm, got %q", p.Rules[0].Action)
	}
}

func TestParsePolicyInvalid(t *testing.T) {
	cases := map[string]string{
		"unsupported version": `version: 2
rules: []`,
		"bad action": `version: 1
rules:
  - name: r
    match: {command: "x"}
    action: nuke`,
		"unnamed rule": `version: 1
rules:
  - match: {command: "x"}
    action: deny`,
		"bad regex": `version: 1
rules:
  - name: r
    match: {command: "(unclosed"}
    action: deny`,
		"unknown field": `version: 1
rules:
  - name: r
    matches: {command: "x"}
    action: deny`,
		"malformed yaml": `version: 1
rules: [::::`,
	}
	for name, yaml := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParsePolicy([]byte(yaml)); err == nil {
				t.Errorf("expected error for %s, got nil", name)
			}
		})
	}
}

func TestLoadPolicyFallsBackToDefault(t *testing.T) {
	// Point HOME at an empty dir → no guards.yaml → default policy.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	p, custom, err := LoadPolicy()
	if err != nil {
		t.Fatal(err)
	}
	if custom {
		t.Error("expected custom=false when no file exists")
	}
	if len(p.Rules) != len(DefaultPolicy().Rules) {
		t.Error("fallback policy should equal the default policy")
	}
}

func TestLoadPolicyUsesUserFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	dir := filepath.Join(tmp, ".config", "opsforge")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `version: 1
rules:
  - name: mine
    match: {command: "docker rm", context: ""}
    action: warn
    message: "removing containers"`
	if err := os.WriteFile(filepath.Join(dir, "guards.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	p, custom, err := LoadPolicy()
	if err != nil {
		t.Fatal(err)
	}
	if !custom {
		t.Error("expected custom=true when a file exists")
	}
	if d := p.Evaluate("docker rm foo", ""); d.Action != ActionWarn {
		t.Errorf("want warn from user policy, got %+v", d)
	}
}

func TestLoadPolicyReportsInvalidUserFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	dir := filepath.Join(tmp, ".config", "opsforge")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "guards.yaml"), []byte("version: 9\nrules: []"), 0o644)
	if _, _, err := LoadPolicy(); err == nil {
		t.Error("expected an error for an invalid user policy, got nil")
	}
}

func TestCurrentContextIsPassiveWithDevNullKubeconfig(t *testing.T) {
	// With KUBECONFIG=/dev/null and no cloud/tf signals, the context must
	// be empty and reading it must not touch kubectl.
	t.Setenv("KUBECONFIG", os.DevNull)
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_VAULT", "")
	t.Setenv("CLOUDSDK_ACTIVE_CONFIG_NAME", "")
	// chdir to a temp dir so no stray .terraform is picked up.
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(tmp)
	if ctx := CurrentContext(); ctx != "" {
		t.Errorf("expected empty context under /dev/null kubeconfig, got %q", ctx)
	}
}

func TestCurrentContextReadsKubeconfigFile(t *testing.T) {
	tmp := t.TempDir()
	cfg := filepath.Join(tmp, "config")
	os.WriteFile(cfg, []byte("apiVersion: v1\ncurrent-context: gke_acme_prod\n"), 0o644)
	t.Setenv("KUBECONFIG", cfg)
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_VAULT", "")
	ctx := CurrentContext()
	if ctx != "gke_acme_prod" {
		t.Errorf("want context from kubeconfig, got %q", ctx)
	}
}

func TestCurrentContextIncludesCloudProfile(t *testing.T) {
	t.Setenv("KUBECONFIG", os.DevNull)
	t.Setenv("AWS_VAULT", "")
	t.Setenv("AWS_PROFILE", "prod-admin")
	if ctx := CurrentContext(); ctx != "aws:prod-admin" {
		t.Errorf("want aws:prod-admin, got %q", ctx)
	}
}

func TestPrefilterTerms(t *testing.T) {
	p := &GuardPolicy{
		Version: 1,
		Rules: []GuardRule{
			{Name: "a", Match: GuardMatch{Command: "kubectl (delete|drain)"}},
			{Name: "b", Match: GuardMatch{Command: "terraform import"}},
			{Name: "c", Match: GuardMatch{Command: "kubectl apply"}}, // dup kubectl
		},
	}
	got := p.PrefilterTerms()
	want := map[string]bool{"kubectl": true, "delete": true, "drain": true, "terraform": true, "import": true, "apply": true}
	if len(got) != len(want) {
		t.Fatalf("got %v (%d terms), want %d distinct", got, len(got), len(want))
	}
	for _, term := range got {
		if !want[term] {
			t.Errorf("unexpected term %q", term)
		}
	}
	// A custom rule's verb must be present — this is the regression that
	// let `terraform import` bypass the shell prefilter.
	found := false
	for _, term := range got {
		if term == "import" {
			found = true
		}
	}
	if !found {
		t.Error("custom rule keyword 'import' missing from prefilter — rule would be silently skipped")
	}
}
