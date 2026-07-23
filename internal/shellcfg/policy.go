// Guard policy engine: policy-as-code for the interactive shell.
//
// The zsh guards module no longer hard-codes which commands are dangerous
// and what "prod" means — it delegates to this engine through
// `opsforge guard check`. Rules are declared in
// ~/.config/opsforge/guards.yaml; when that file is absent, DefaultPolicy
// reproduces the behavior the module used to have (so upgrading changes
// nothing until the user opts into custom rules).
//
// Like the prompt and the old guards, this NEVER invokes kubectl/gcloud:
// the "context" it matches against is read passively from the kubeconfig
// file and a few environment variables, so evaluating a rule can't trigger
// an OIDC browser login.
package shellcfg

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Action is what a matching rule tells the shell to do with a command.
type Action string

const (
	// ActionAllow lets the command run without interruption. It is also
	// the result when no rule matches.
	ActionAllow Action = "allow"
	// ActionWarn prints the message but still runs the command.
	ActionWarn Action = "warn"
	// ActionConfirm requires an explicit "yes" before running.
	ActionConfirm Action = "confirm"
	// ActionDeny blocks the command outright.
	ActionDeny Action = "deny"
)

// validActions is the closed set accepted by the parser/validator.
var validActions = map[Action]bool{
	ActionAllow:   true,
	ActionWarn:    true,
	ActionConfirm: true,
	ActionDeny:    true,
}

// GuardMatch is the predicate part of a rule. An empty field matches
// anything. `command` is matched as a substring by default, or as a regex
// when it fails to appear literally is NOT how it works — see Rule.compile:
// both are regexes, but a plain string like "kubectl delete" is a valid
// regex that behaves like a substring match, which keeps the YAML simple.
type GuardMatch struct {
	Command string `yaml:"command"`
	Context string `yaml:"context"`
}

// GuardRule is one declarative guard: when a command matches Command and
// the current context matches Context, apply Action (showing Message).
type GuardRule struct {
	Name    string     `yaml:"name"`
	Match   GuardMatch `yaml:"match"`
	Action  Action     `yaml:"action"`
	Message string     `yaml:"message"`

	// compiled forms, populated by compile().
	cmdRe *regexp.Regexp
	ctxRe *regexp.Regexp
}

// GuardPolicy is the top-level document: a version and an ordered list of
// rules. Order matters — the first matching rule wins (see Evaluate).
type GuardPolicy struct {
	Version int         `yaml:"version"`
	Rules   []GuardRule `yaml:"rules"`
}

// Decision is the outcome of evaluating a command against a policy.
type Decision struct {
	Action  Action
	Message string
	Rule    string // name of the matching rule, empty when none matched
}

// compile builds the regexes for a rule and validates its action. Command
// and context are treated as case-insensitive regexes; an empty pattern
// matches anything.
func (r *GuardRule) compile() error {
	if r.Action == "" {
		r.Action = ActionConfirm // sensible default: ask, don't block
	}
	if !validActions[r.Action] {
		return fmt.Errorf("rule %q: invalid action %q (want allow|warn|confirm|deny)", r.Name, r.Action)
	}
	var err error
	if r.cmdRe, err = compilePattern(r.Match.Command); err != nil {
		return fmt.Errorf("rule %q: bad command pattern: %w", r.Name, err)
	}
	if r.ctxRe, err = compilePattern(r.Match.Context); err != nil {
		return fmt.Errorf("rule %q: bad context pattern: %w", r.Name, err)
	}
	return nil
}

// compilePattern compiles a case-insensitive regex. An empty pattern
// yields nil, which matches() treats as "always matches".
func compilePattern(pat string) (*regexp.Regexp, error) {
	if pat == "" {
		return nil, nil
	}
	return regexp.Compile("(?i)" + pat)
}

// matches reports whether the rule fires for this command/context pair.
func (r *GuardRule) matches(command, context string) bool {
	if r.cmdRe != nil && !r.cmdRe.MatchString(command) {
		return false
	}
	if r.ctxRe != nil && !r.ctxRe.MatchString(context) {
		return false
	}
	return true
}

// prefilterTermRe pulls alphanumeric words (2+ chars) out of a command
// pattern, so "kubectl (delete|drain)" yields kubectl, delete, drain.
var prefilterTermRe = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9_-]{1,}`)

// PrefilterTerms returns the distinct lowercase keywords that appear in
// the policy's command patterns. The shell prefilter uses these to decide,
// with zero subprocesses, whether a typed command could possibly match a
// rule — so it is derived from the ACTUAL rules (built-in or the user's
// guards.yaml) instead of a hard-coded verb list. That closes the gap
// where a custom rule (e.g. `terraform import`) was silently skipped
// because the static prefilter never let it reach the engine.
func (p *GuardPolicy) PrefilterTerms() []string {
	seen := map[string]bool{}
	var terms []string
	for i := range p.Rules {
		for _, w := range prefilterTermRe.FindAllString(p.Rules[i].Match.Command, -1) {
			w = strings.ToLower(w)
			if seen[w] {
				continue
			}
			seen[w] = true
			terms = append(terms, w)
		}
	}
	return terms
}

// Validate compiles every rule and reports the first problem. A policy
// that fails validation is never used (the engine falls back to default),
// so a typo in the user's YAML can't silently disable all guards.
func (p *GuardPolicy) Validate() error {
	if p.Version != 1 {
		return fmt.Errorf("unsupported policy version %d (want 1)", p.Version)
	}
	for i := range p.Rules {
		if p.Rules[i].Name == "" {
			return fmt.Errorf("rule %d has no name", i)
		}
		if err := p.Rules[i].compile(); err != nil {
			return err
		}
	}
	return nil
}

// Evaluate returns the decision for a command in a context. The first rule
// that matches wins; if none match, the command is allowed. Callers must
// have run Validate (ParsePolicy and DefaultPolicy both do).
func (p *GuardPolicy) Evaluate(command, context string) Decision {
	for i := range p.Rules {
		r := &p.Rules[i]
		if r.cmdRe == nil && r.ctxRe == nil && r.Match.Command == "" && r.Match.Context == "" {
			// A rule with no predicate would match everything; skip it
			// defensively so an empty match can't block the whole shell.
			continue
		}
		if r.matches(command, context) {
			return Decision{Action: r.Action, Message: r.Message, Rule: r.Name}
		}
	}
	return Decision{Action: ActionAllow}
}

// ParsePolicy decodes and validates a YAML policy document.
func ParsePolicy(data []byte) (*GuardPolicy, error) {
	var p GuardPolicy
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true) // reject unknown keys so typos surface loudly
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("parsing guards policy: %w", err)
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

// DefaultPolicy is the built-in policy used when the user has no
// guards.yaml. It ports the commands the old hard-coded guards.zsh
// protected, so upgrading to the policy engine is behavior-preserving.
func DefaultPolicy() *GuardPolicy {
	p := &GuardPolicy{
		Version: 1,
		Rules: []GuardRule{
			{
				Name:    "confirm destructive kubectl on prod",
				Match:   GuardMatch{Command: `kubectl (delete|drain|cordon|apply|replace)`, Context: `prod|production`},
				Action:  ActionConfirm,
				Message: "This changes PRODUCTION Kubernetes resources.",
			},
			{
				Name:    "confirm helm changes on prod",
				Match:   GuardMatch{Command: `helm (uninstall|delete|rollback)`, Context: `prod|production`},
				Action:  ActionConfirm,
				Message: "This changes a PRODUCTION helm release.",
			},
			{
				Name:    "confirm terraform destroy/apply on prod (by context)",
				Match:   GuardMatch{Command: `(terraform|tofu|terragrunt) (destroy|apply)`, Context: `prod|production`},
				Action:  ActionConfirm,
				Message: "This changes PRODUCTION infrastructure (prod context detected).",
			},
			{
				// Context detection (the tf workspace) misses the common
				// real-world cases: -var-file=prod.tfvars, an environments/prod
				// directory, or `workspace select prod`. So also look at the
				// command line itself, which the guard already sees in full.
				// Go's RE2 has no lookahead, so we spell out both orders (verb
				// then marker, marker then verb) to stay order-independent —
				// covering chained `tofu workspace select prod && tofu destroy`.
				Name:    "confirm terraform destroy/apply targeting prod (by command)",
				Match:   GuardMatch{Command: `(terraform|tofu|terragrunt).*((destroy|apply).*(-var-?file[= ]\S*prod|environments?/prod|prod\.tfvars|workspace\s+select\s+prod)|(-var-?file[= ]\S*prod|environments?/prod|prod\.tfvars|workspace\s+select\s+prod).*(destroy|apply))`},
				Action:  ActionConfirm,
				Message: "This terraform command targets PRODUCTION (prod var-file / directory / workspace).",
			},
		},
	}
	// DefaultPolicy is trusted; if it ever fails to compile that's a
	// programming error, so panic in tests via Validate would be caught.
	if err := p.Validate(); err != nil {
		panic("opsforge: default guard policy is invalid: " + err.Error())
	}
	return p
}

// PolicyPath returns the path to the user's guards.yaml
// (~/.config/opsforge/guards.yaml).
func PolicyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home + "/.config/opsforge/guards.yaml", nil
}

// LoadPolicy loads the user's guards.yaml, falling back to DefaultPolicy
// when the file is absent. The bool reports whether a user file was used.
// A present-but-invalid file returns an error: better to tell the user
// their policy is broken than to silently run with the defaults.
func LoadPolicy() (*GuardPolicy, bool, error) {
	path, err := PolicyPath()
	if err != nil {
		return nil, false, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultPolicy(), false, nil
	}
	if err != nil {
		return nil, false, err
	}
	p, err := ParsePolicy(data)
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", path, err)
	}
	return p, true, nil
}

// CurrentContext reads a best-effort context string used for matching,
// WITHOUT ever invoking kubectl/gcloud. It concatenates the kube
// current-context, the cloud profile, and the terraform workspace so a
// rule's context regex can match any of them. Reading is purely passive.
func CurrentContext() string {
	var parts []string
	if ctx := kubeCurrentContext(); ctx != "" {
		parts = append(parts, ctx)
	}
	if c := cloudProfile(); c != "" {
		parts = append(parts, c)
	}
	if ws := terraformWorkspace(); ws != "" {
		parts = append(parts, ws)
	}
	return strings.Join(parts, " ")
}

// kubeCurrentContext reads current-context straight from the kubeconfig
// file — the same passive method the prompt and old guards used, so it
// can't trigger an exec-credential OIDC login.
func kubeCurrentContext() string {
	cfg := firstKubeconfig()
	if cfg == "" {
		return ""
	}
	f, err := os.Open(cfg)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "current-context:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "current-context:"))
			return strings.Trim(v, `"'`)
		}
	}
	return ""
}

// firstKubeconfig returns the first path in KUBECONFIG (colon-separated),
// or ~/.kube/config. It returns "" when that maps to /dev/null so tests
// running with KUBECONFIG=/dev/null get an empty context.
func firstKubeconfig() string {
	cfg := os.Getenv("KUBECONFIG")
	if i := strings.IndexByte(cfg, ':'); i >= 0 {
		cfg = cfg[:i]
	}
	if cfg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		cfg = home + "/.kube/config"
	}
	if cfg == os.DevNull {
		return ""
	}
	return cfg
}

// cloudProfile mirrors the prompt's cloud segment: env-var only, never a
// gcloud/aws probe.
func cloudProfile() string {
	if v := os.Getenv("AWS_VAULT"); v != "" {
		return "aws:" + v
	}
	if v := os.Getenv("AWS_PROFILE"); v != "" {
		return "aws:" + v
	}
	if v := os.Getenv("CLOUDSDK_ACTIVE_CONFIG_NAME"); v != "" {
		return "gcloud:" + v
	}
	return ""
}

// terraformWorkspace reads .terraform/environment in the current dir, like
// the prompt's tf segment.
func terraformWorkspace() string {
	data, err := os.ReadFile(".terraform/environment")
	if err != nil {
		return ""
	}
	ws := strings.TrimSpace(string(data))
	if ws == "" || ws == "default" {
		return ""
	}
	return "tf:" + ws
}
