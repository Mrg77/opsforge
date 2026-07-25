package credscan

import "time"

// Scan runs every credential-hygiene scanner and returns a consolidated,
// sorted report. It is passive and offline: no external tool is ever executed
// (in particular never kubectl/aws/gcloud), no secret value is included in the
// result, and the network is never touched. now is injected so all expiry
// checks share one clock; pass time.Now().
func Scan(now time.Time) Report {
	findings := []Finding{} // non-nil so JSON emits [] not null on a clean scan
	findings = append(findings, scanCloud(now)...)
	findings = append(findings, scanKube(now)...)
	findings = append(findings, scanSSH()...)
	findings = append(findings, scanSCM()...)
	findings = append(findings, scanInfra()...)

	sortFindings(findings)
	return Report{
		Findings: findings,
		Scanned:  countScanned(),
	}
}

// scannedLocations lists the credential files/dirs whose presence we count as
// "inspected", so the report can honestly say how much of the workstation it
// looked at (present locations only — absent ones aren't counted).
var scannedLocations = []string{
	"~/.aws/credentials", "~/.aws/config",
	"~/.config/gcloud/application_default_credentials.json",
	"~/.azure/service_principal_entries.json", "~/.azure/msal_token_cache.json",
	"~/.oci/config", "~/.config/doctl/config.yaml", "~/.config/scw/config.yaml",
	"~/.ovh.conf", "~/.aliyun/config.json", "~/.vault-token",
	"~/.kube/config",
	"~/.ssh",
	"~/.netrc", "~/.git-credentials", "~/.config/git/credentials",
	"~/.config/gh/hosts.yml", "~/.config/glab-cli/config.yml",
	"~/.docker/config.json", "~/.terraform.d/credentials.tfrc.json",
	"~/.npmrc", "~/.pypirc", "~/.cargo/credentials.toml", "~/.gem/credentials",
}

// countScanned counts how many of the known credential locations actually
// exist on this machine.
func countScanned() int {
	n := 0
	for _, p := range scannedLocations {
		if _, ok := statFile(p); ok {
			n++
		}
	}
	return n
}
