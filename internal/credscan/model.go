// Package credscan inventories the credentials stored locally on a DevOps
// workstation and flags hygiene risks: long-lived static keys, secrets in
// clear text, over-broad file permissions, and expired or soon-to-expire
// certificates and tokens.
//
// SECURITY / DESIGN — READ-ONLY, PASSIVE, OFFLINE:
//   - It NEVER executes an external tool. In particular it never runs
//     kubectl/aws/gcloud — reading a kubeconfig must not trigger an OIDC
//     login. Every check parses files on disk and nothing more.
//   - It NEVER prints a secret's value. Findings reference *where* a secret
//     lives and *why* it's risky, never the secret itself.
//   - It needs no network. Certificate expiry is read from the local PEM,
//     token expiry from the local JWT/JSON — no call home.
//
// The scanners live in scan_*.go, one file per credential family (cloud,
// kube, scm, ...). Each returns []Finding; verify() in cmd/verify.go runs
// them and shapes the report.
package credscan

import "time"

// Severity is a coarse, sortable hygiene risk level. It mirrors the shape of
// audit.Severity so the verify command can reuse the same ui styles, but it
// is its own type: a credential-hygiene risk is not a CVE.
type Severity int

const (
	SevOK Severity = iota // informational: a well-formed, temporary/short-lived credential
	SevLow
	SevMedium
	SevHigh
	SevCritical
)

func (s Severity) String() string {
	switch s {
	case SevCritical:
		return "CRITICAL"
	case SevHigh:
		return "HIGH"
	case SevMedium:
		return "MEDIUM"
	case SevLow:
		return "LOW"
	default:
		return "OK"
	}
}

// Finding is one credential-hygiene observation about one location on disk.
// It deliberately carries no secret material — only its location and the
// reason it was flagged.
type Finding struct {
	// Provider groups findings in the report, e.g. "AWS", "Kubernetes",
	// "SSH", "Docker", "npm".
	Provider string `json:"provider"`
	// Kind is the specific credential type, e.g. "static access key",
	// "client certificate", "personal access token".
	Kind string `json:"kind"`
	// Path is the file the finding is about (secret value never included).
	Path string `json:"path"`
	// Severity is the hygiene risk level.
	Severity Severity `json:"-"`
	// SeverityLabel is the string form, emitted in JSON.
	SeverityLabel string `json:"severity"`
	// Title is a short problem statement, e.g. "long-lived static access key".
	Title string `json:"title"`
	// Detail explains the risk and, where relevant, how to fix it. No secret.
	Detail string `json:"detail,omitempty"`
	// Expires is set when the credential carries an expiry (cert/token). A
	// finding may be SevOK yet still report Expires for context.
	Expires *time.Time `json:"expires,omitempty"`
}

// mkFinding is a small constructor that keeps SeverityLabel in sync with
// Severity so callers can't forget to set the JSON label.
func mkFinding(provider, kind, path string, sev Severity, title, detail string) Finding {
	return Finding{
		Provider:      provider,
		Kind:          kind,
		Path:          path,
		Severity:      sev,
		SeverityLabel: sev.String(),
		Title:         title,
		Detail:        detail,
	}
}

// Report is the full result of a scan, ready to shape for human or JSON output.
type Report struct {
	Findings []Finding `json:"findings"`
	// Scanned is the number of credential locations that were inspected
	// (present on disk), so the report can honestly say "looked at N, flagged M"
	// rather than implying silence means safety.
	Scanned int `json:"scanned"`
}

// TopSeverity returns the highest severity among the findings (SevOK if none).
func (r Report) TopSeverity() Severity {
	top := SevOK
	for _, f := range r.Findings {
		if f.Severity > top {
			top = f.Severity
		}
	}
	return top
}

// Risky returns the findings above SevOK, most-severe first, then by provider.
func (r Report) Risky() []Finding {
	out := make([]Finding, 0, len(r.Findings))
	for _, f := range r.Findings {
		if f.Severity > SevOK {
			out = append(out, f)
		}
	}
	sortFindings(out)
	return out
}
