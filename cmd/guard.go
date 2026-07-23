package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/shellcfg"
	"github.com/Mrg77/opsforge/internal/ui"
)

// guard is opsforge's policy-as-code layer for the interactive shell: a
// declarative set of rules (~/.config/opsforge/guards.yaml) that decides
// whether a destructive command should run, warn, confirm, or be denied,
// based on the current kube/cloud/terraform context.
//
// The shell's accept-line widget calls `guard check` on every command; the
// other subcommands are for humans (list/test/init).
var guardCmd = &cobra.Command{
	Use:   "guard",
	Short: "Policy-as-code guards for destructive shell commands",
	Long: `opsforge guards confirm, warn on, or block dangerous commands based on
declarative rules matched against your current context (kube/cloud/tf).

Rules live in ~/.config/opsforge/guards.yaml. With no file, a built-in
default policy protects destructive kubectl/helm/terraform commands on a
prod-looking context. Run 'opsforge guard init' to start customizing.

Disable guards for a session with OPSFORGE_GUARDS=0.`,
}

// guardCheckCmd is the machine-facing entry the zsh widget calls on every
// destructive-looking command. It must stay fast (no network, no cloud
// probes) — the context is read passively. Output is a single line:
//
//	allow
//	warn|<message>
//	confirm|<message>
//	deny|<message>
//
// so the zsh side can parse it with a simple ${line%%|*} / ${line#*|}.
var guardCheckCmd = &cobra.Command{
	Use:    "check <command> [context]",
	Short:  "Evaluate a command against the guard policy (used by the shell)",
	Args:   cobra.RangeArgs(1, 2),
	Hidden: true, // internal plumbing, not part of the human-facing UI
	RunE: func(cmd *cobra.Command, args []string) error {
		policy, _, err := shellcfg.LoadPolicy()
		if err != nil {
			// A broken policy must not wedge the shell: fail open (allow)
			// but tell the user on stderr so they can fix their YAML.
			fmt.Fprintln(os.Stderr, "opsforge guard: "+err.Error())
			fmt.Fprintln(cmd.OutOrStdout(), string(shellcfg.ActionAllow))
			return nil
		}
		context := ""
		if len(args) == 2 {
			context = args[1]
		} else {
			context = shellcfg.CurrentContext()
		}
		d := policy.Evaluate(args[0], context)
		if d.Message != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "%s|%s\n", d.Action, d.Message)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), string(d.Action))
		}
		return nil
	},
}

var guardListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show the active guard rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		policy, custom, err := shellcfg.LoadPolicy()
		if err != nil {
			return err
		}
		path, _ := shellcfg.PolicyPath()
		source := "built-in default policy"
		if custom {
			source = path
		}
		fmt.Println(ui.Header("opsforge guards", source))
		fmt.Println()
		if len(policy.Rules) == 0 {
			fmt.Println(ui.Dim.Render("  no rules defined"))
			return nil
		}
		for _, r := range policy.Rules {
			fmt.Printf("  %s %s\n", actionGlyph(r.Action), ui.Accent.Render(r.Name))
			fmt.Printf("      %s %s\n", ui.Label("command", 8), matchOrAny(r.Match.Command))
			fmt.Printf("      %s %s\n", ui.Label("context", 8), matchOrAny(r.Match.Context))
			fmt.Printf("      %s %s\n", ui.Label("action", 8), string(r.Action))
			if r.Message != "" {
				fmt.Printf("      %s %s\n", ui.Label("message", 8), ui.Dim.Render(r.Message))
			}
			fmt.Println()
		}
		if !custom {
			fmt.Println(ui.Dim.Render("  Run `opsforge guard init` to write a customizable guards.yaml."))
		}
		return nil
	},
}

var guardTestContext string

var guardTestCmd = &cobra.Command{
	Use:   "test <command>",
	Short: "Simulate a command: show which rule matches and the action",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		policy, _, err := shellcfg.LoadPolicy()
		if err != nil {
			return err
		}
		command := joinArgs(args)
		context := guardTestContext
		if context == "" {
			context = shellcfg.CurrentContext()
		}

		// Machine path: emit the decision as JSON so a CI job can assert a
		// command's action against the committed policy (e.g. that
		// "terraform destroy" is denied on prod). Same Evaluate call as the
		// human path, so the two never diverge.
		if output.JSON {
			d := policy.Evaluate(command, context)
			return output.Emit(struct {
				Command     string `json:"command"`
				Context     string `json:"context"`
				MatchedRule string `json:"matched_rule"`
				Action      string `json:"action"`
				Message     string `json:"message"`
			}{command, context, d.Rule, string(d.Action), d.Message})
		}

		fmt.Println(ui.Header("opsforge guard test", ""))
		fmt.Println()
		fmt.Printf("  %s %s\n", ui.Label("command", 8), ui.Accent.Render(command))
		ctxShown := context
		if ctxShown == "" {
			ctxShown = "(none)"
		}
		fmt.Printf("  %s %s\n\n", ui.Label("context", 8), ctxShown)

		d := policy.Evaluate(command, context)
		switch d.Action {
		case shellcfg.ActionAllow:
			fmt.Printf("  %s %s\n", ui.OKMark(), ui.OK.Render("allow — no rule matched, the command would run"))
		case shellcfg.ActionWarn:
			fmt.Printf("  %s matched %q → %s\n", ui.WarnMark(), d.Rule, ui.Warn.Render("warn (runs after showing a message)"))
		case shellcfg.ActionConfirm:
			fmt.Printf("  %s matched %q → %s\n", ui.WarnMark(), d.Rule, ui.Warn.Render("confirm (asks for 'yes' before running)"))
		case shellcfg.ActionDeny:
			fmt.Printf("  %s matched %q → %s\n", ui.ErrMark(), d.Rule, ui.Err.Render("deny (blocked)"))
		}
		if d.Message != "" {
			fmt.Printf("      %s\n", ui.Dim.Render(d.Message))
		}
		return nil
	},
}

// guardLintCmd validates the active guards policy and exits non-zero when
// it is invalid — the piece that makes policy-as-code CI-enforceable. A team
// commits guards.yaml and runs `opsforge guard lint` on every change; a typo
// (bad regex, unknown action, wrong version) fails the job instead of
// silently falling back to the default policy at runtime.
var guardLintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Validate guards.yaml (non-zero exit on error, for CI)",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := shellcfg.PolicyPath()

		// LoadPolicy validates as it loads: a present-but-broken file returns
		// an error, an absent file yields the (always-valid) default. We split
		// those cases so JSON reports a clean errors[] rather than a Go error.
		policy, custom, err := shellcfg.LoadPolicy()
		source := "no guards.yaml — built-in default policy"
		if err != nil {
			// A present-but-broken file: LoadPolicy reports custom=false, but
			// the file exists — name it so the header points at what to fix.
			source = path
		} else if custom {
			source = path
		}

		var errs []string
		valid := true
		rules := 0
		if err != nil {
			valid = false
			errs = append(errs, err.Error())
		} else {
			rules = len(policy.Rules)
			// LoadPolicy already validated, but re-run Validate so lint is
			// self-contained and future-proof against a load that skips it.
			if verr := policy.Validate(); verr != nil {
				valid = false
				errs = append(errs, verr.Error())
			}
		}

		if output.JSON {
			if emitErr := output.Emit(struct {
				Valid  bool     `json:"valid"`
				Rules  int      `json:"rules"`
				Errors []string `json:"errors"`
			}{valid, rules, errs}); emitErr != nil {
				return emitErr
			}
		} else {
			fmt.Println(ui.Header("opsforge guard lint", source))
			fmt.Println()
			if valid {
				fmt.Printf("  %s policy is valid (%s)\n",
					ui.OKMark(), plural(rules, "rule"))
			} else {
				fmt.Printf("  %s policy is invalid\n", ui.ErrMark())
				for _, e := range errs {
					fmt.Printf("      %s %s\n", ui.MarkArrow, ui.Err.Render(e))
				}
			}
		}

		if !valid {
			// Non-zero exit for CI. SilenceErrors/SilenceUsage are set on
			// root, and we've already printed the diagnostics ourselves, so
			// exit directly rather than returning a (re-printed) error.
			os.Exit(1)
		}
		return nil
	},
}

var guardInitForce bool

var guardInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Write a commented example guards.yaml to ~/.config/opsforge/",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := shellcfg.PolicyPath()
		if err != nil {
			return err
		}
		if _, err := os.Stat(path); err == nil && !guardInitForce {
			return fmt.Errorf("%s already exists (use --force to overwrite)", path)
		}
		if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(exampleGuardsYAML), 0o644); err != nil {
			return err
		}
		fmt.Printf("%s wrote %s\n", ui.OKMark(), path)
		fmt.Println(ui.Dim.Render("  Edit it, then check your rules with `opsforge guard list` and"))
		fmt.Println(ui.Dim.Render("  `opsforge guard test \"terraform destroy\" --context prod`."))
		return nil
	},
}

// guardPrefilterCmd emits a zsh extended-glob alternation of the keywords
// that appear in the active policy's command patterns. The shell guard
// module sources this so its cheap in-shell prefilter tracks the REAL
// rules — a custom rule on, say, `terraform import` is then no longer
// silently skipped by a hard-coded verb list. Hidden: it's plumbing.
var guardPrefilterCmd = &cobra.Command{
	Use:    "prefilter",
	Short:  "Print the shell prefilter pattern derived from the active policy",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		policy, _, err := shellcfg.LoadPolicy()
		if err != nil {
			return err
		}
		terms := policy.PrefilterTerms()
		if len(terms) == 0 {
			// No rules → match nothing (empty alternation would match all).
			fmt.Println("")
			return nil
		}
		fmt.Printf("(%s)\n", strings.Join(terms, "|"))
		return nil
	},
}

func actionGlyph(a shellcfg.Action) string {
	switch a {
	case shellcfg.ActionDeny:
		return ui.ErrMark()
	case shellcfg.ActionConfirm, shellcfg.ActionWarn:
		return ui.WarnMark()
	default:
		return ui.OKMark()
	}
}

func matchOrAny(s string) string {
	if s == "" {
		return ui.Dim.Render("(any)")
	}
	return s
}

func joinArgs(args []string) string {
	out := args[0]
	for _, a := range args[1:] {
		out += " " + a
	}
	return out
}

func dirOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}

// exampleGuardsYAML is the seed written by `guard init`. It documents the
// format and reproduces the default policy so users start from working,
// editable rules.
const exampleGuardsYAML = `# opsforge guards — policy-as-code for your interactive shell.
#
# Every command you run is matched against these rules (first match wins).
# match.command and match.context are case-insensitive regexes; leave a
# field out to match anything. The context is read passively from your
# kubeconfig current-context, AWS_PROFILE/AWS_VAULT, and the terraform
# workspace — opsforge never runs kubectl/gcloud to figure it out.
#
# action: allow | warn | confirm | deny
#   allow   — run normally (also the result when nothing matches)
#   warn    — print message, then run
#   confirm — require typing 'yes' before running
#   deny    — block the command
#
# Disable all guards for a session with: OPSFORGE_GUARDS=0
version: 1
rules:
  - name: "confirm destructive kubectl on prod"
    match:
      command: "kubectl (delete|drain|cordon|apply|replace)"
      context: "prod|production"
    action: confirm
    message: "This changes PRODUCTION Kubernetes resources."

  - name: "confirm helm changes on prod"
    match:
      command: "helm (uninstall|delete|rollback)"
      context: "prod|production"
    action: confirm
    message: "This changes a PRODUCTION helm release."

  - name: "confirm terraform destroy/apply on prod"
    match:
      command: "terraform (destroy|apply)"
      context: "prod|production"
    action: confirm
    message: "This changes PRODUCTION infrastructure."

  # Example of a hard block (uncomment to forbid outright):
  # - name: "never delete namespaces on prod"
  #   match:
  #     command: "kubectl delete (ns|namespace)"
  #     context: "prod"
  #   action: deny
  #   message: "Deleting a prod namespace is forbidden by policy."
`

func init() {
	guardTestCmd.Flags().StringVar(&guardTestContext, "context", "",
		"context to simulate against (defaults to the current context)")
	guardInitCmd.Flags().BoolVar(&guardInitForce, "force", false,
		"overwrite an existing guards.yaml")
	guardCmd.AddCommand(guardCheckCmd, guardListCmd, guardTestCmd, guardLintCmd, guardInitCmd, guardPrefilterCmd)
	rootCmd.AddCommand(guardCmd)
}
