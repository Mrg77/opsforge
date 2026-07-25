package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/credscan"
	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/ui"
)

var verifyStrict bool

// verify inventories the credentials stored locally on the workstation and
// flags hygiene risks: long-lived static keys, secrets in clear text,
// over-broad file permissions, and expired certificates/tokens.
//
// It is strictly read-only, passive, and offline — it NEVER runs an external
// tool (in particular never kubectl/aws/gcloud, so reading an OIDC kubeconfig
// can't trigger a login), never prints a secret's value, and never touches the
// network. It only tells you *where* a risky credential lives and *why*.
var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Audit the credentials on this workstation for hygiene risks (read-only)",
	Long: `Inventory the credentials stored locally on this machine and flag hygiene risks:

  • long-lived static keys (AWS AKIA, GCP service-account JSON, SSH without a
    passphrase, static kube tokens) that never expire
  • secrets in clear text (git credential store, .netrc, docker/npm base64
    logins, gh/glab tokens)
  • files readable beyond their owner (a credential that should be 0600)
  • expired or soon-to-expire client certificates and tokens

It is READ-ONLY and OFFLINE: it never runs an external tool (never kubectl —
so an OIDC kubeconfig can't trigger a login), never prints a secret's value,
and never touches the network. It reports *where* a risky credential lives and
*why*, not the secret itself.

  opsforge verify            # human-readable report
  opsforge verify --json     # machine-readable, for scripts
  opsforge verify --strict   # exit non-zero on ANY finding (CI gate)

Without --strict the exit code is non-zero only on HIGH/CRITICAL findings, so
it can gate CI without failing on every minor heads-up.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		report := credscan.Scan(time.Now())

		if output.JSON {
			if err := output.Emit(report); err != nil {
				return err
			}
			return exitCodeFor(report, verifyStrict)
		}

		renderVerify(report)
		return exitCodeFor(report, verifyStrict)
	},
}

// renderVerify prints the human report: a header, findings grouped
// most-severe-first, and an honest summary of what was inspected.
func renderVerify(r credscan.Report) {
	fmt.Println(ui.Header("opsforge verify", "credential hygiene of this workstation — read-only"))
	fmt.Println()

	risky := r.Risky()
	if len(risky) == 0 {
		fmt.Println(ui.OK.Render("  No credential-hygiene risks found."))
		fmt.Println()
		fmt.Println(ui.Faint.Render(fmt.Sprintf("  Inspected %d credential location(s). "+
			"Absence of a finding isn't proof of safety — some stores (OS keychains) can't be read.", r.Scanned)))
		return
	}

	for _, f := range risky {
		fmt.Printf("  %s  %s\n", sevBadge(f.Severity), f.Title)
		fmt.Printf("      %s\n", ui.Faint.Render(f.Provider+" · "+f.Path))
		if f.Expires != nil {
			fmt.Printf("      %s\n", ui.Faint.Render("expires: "+f.Expires.Local().Format("2006-01-02 15:04")))
		}
		if f.Detail != "" {
			fmt.Printf("      %s\n", ui.Dim.Render(f.Detail))
		}
		fmt.Println()
	}

	crit, high, other := tallyBySeverity(risky)
	fmt.Println(ui.Faint.Render(fmt.Sprintf(
		"  %d finding(s) across %d inspected location(s): %d critical, %d high, %d lower.",
		len(risky), r.Scanned, crit, high, other)))
	fmt.Println(ui.Faint.Render("  No secret values are shown or stored. This is a safety net, not a guarantee."))
}

// exitCodeFor returns a non-zero-exit error when the report should fail a CI
// gate: any finding under --strict, else only HIGH/CRITICAL.
func exitCodeFor(r credscan.Report, strict bool) error {
	risky := r.Risky()
	if len(risky) == 0 {
		return nil
	}
	if strict {
		return fmt.Errorf("%d credential-hygiene finding(s)", len(risky))
	}
	crit, high, _ := tallyBySeverity(risky)
	if crit+high > 0 {
		return fmt.Errorf("%d HIGH/CRITICAL credential-hygiene finding(s)", crit+high)
	}
	return nil
}

func tallyBySeverity(fs []credscan.Finding) (crit, high, other int) {
	for _, f := range fs {
		switch f.Severity {
		case credscan.SevCritical:
			crit++
		case credscan.SevHigh:
			high++
		default:
			other++
		}
	}
	return
}

// sevBadge renders a severity label in its themed style.
func sevBadge(s credscan.Severity) string {
	switch s {
	case credscan.SevCritical:
		return ui.SevCritical.Render("CRITICAL")
	case credscan.SevHigh:
		return ui.SevHigh.Render("HIGH    ")
	case credscan.SevMedium:
		return ui.SevMedium.Render("MEDIUM  ")
	default:
		return ui.SevLow.Render("LOW     ")
	}
}

func init() {
	verifyCmd.Flags().BoolVar(&verifyStrict, "strict", false,
		"exit non-zero on any finding, not just HIGH/CRITICAL (CI gate)")
	rootCmd.AddCommand(verifyCmd)
}
