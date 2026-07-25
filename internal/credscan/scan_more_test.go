package credscan

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestParseTime(t *testing.T) {
	cases := []string{
		"2026-07-25T14:30:00Z",
		"2026-07-25T14:30:00.123456Z",
		"2026-07-25 14:30:00.123456",
		"2026-07-25 14:30:00",
	}
	for _, s := range cases {
		if _, ok := parseTime(s); !ok {
			t.Errorf("parseTime(%q) failed", s)
		}
	}
	if _, ok := parseTime("not a date"); ok {
		t.Error("parseTime must reject garbage")
	}
}

func TestNetrcAndGitCredentials(t *testing.T) {
	dir := withHome(t)
	write(t, filepath.Join(dir, ".netrc"),
		"machine github.com login me password ghp_realtoken\n", 0o600)
	write(t, filepath.Join(dir, ".git-credentials"),
		"https://me:ghp_realtoken@github.com\n", 0o600)

	var netrc, gitcred bool
	for _, f := range scanSCM() {
		if f.Provider == "Git" && f.Kind == "clear-text password" {
			netrc = true
		}
		if f.Provider == "Git" && f.Kind == "stored credential" {
			gitcred = true
		}
	}
	if !netrc {
		t.Error(".netrc with a password must be flagged")
	}
	if !gitcred {
		t.Error(".git-credentials with a token must be flagged")
	}
}

func TestNetrcPlaceholderNotFlagged(t *testing.T) {
	dir := withHome(t)
	write(t, filepath.Join(dir, ".netrc"),
		"machine example.com login me password ${TOKEN}\n", 0o600)
	for _, f := range scanSCM() {
		if f.Kind == "clear-text password" {
			t.Error("an env-interpolated password is not a hard-coded secret")
		}
	}
}

func TestGlabTokenFlagged(t *testing.T) {
	dir := withHome(t)
	write(t, filepath.Join(dir, ".config", "glab-cli", "config.yml"), `
hosts:
    gitlab.com:
        token: glpat-realtokenvalue
`, 0o600)
	var flagged bool
	for _, f := range scanSCM() {
		if f.Provider == "GitLab" && f.Kind == "personal access token" {
			flagged = true
		}
	}
	if !flagged {
		t.Error("a glab token must be flagged")
	}
}

func TestOCIStaticKeyVsSessionToken(t *testing.T) {
	dir := withHome(t)
	write(t, filepath.Join(dir, ".oci", "config"), `
[STATIC]
user = ocid1.user
key_file = ~/.oci/oci_api_key.pem

[SESSION]
security_token_file = ~/.oci/sessions/DEFAULT/token
`, 0o600)
	var static, session bool
	for _, f := range scanMiscCloud(time.Now()) {
		if f.Kind == "static API key" && f.Severity == SevHigh {
			static = true
		}
		if f.Kind == "session token" && f.Severity == SevOK {
			session = true
		}
	}
	if !static {
		t.Error("an OCI key_file profile must be HIGH")
	}
	if !session {
		t.Error("an OCI security_token profile must be OK")
	}
}

func TestVaultTokenPresence(t *testing.T) {
	dir := withHome(t)
	// world-readable vault token => a permission finding.
	write(t, filepath.Join(dir, ".vault-token"), "hvs.exampletoken", 0o644)
	var perm bool
	for _, f := range scanMiscCloud(time.Now()) {
		if f.Provider == "Vault" && f.Severity == SevMedium {
			perm = true
		}
	}
	if !perm {
		t.Error("a world-readable ~/.vault-token must raise a permission finding")
	}
}

func TestAWSSSOExpiredTokenCached(t *testing.T) {
	dir := withHome(t)
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	body, _ := json.Marshal(map[string]string{"accessToken": "x", "expiresAt": past})
	write(t, filepath.Join(dir, ".aws", "sso", "cache", "abc.json"), string(body), 0o600)

	var stale bool
	for _, f := range scanAWS(time.Now()) {
		if f.Kind == "SSO token cache" && f.Expires != nil {
			stale = true
		}
	}
	if !stale {
		t.Error("an expired SSO cache token should be reported")
	}
}

func TestTerraformTokenFlagged(t *testing.T) {
	dir := withHome(t)
	write(t, filepath.Join(dir, ".terraform.d", "credentials.tfrc.json"),
		`{"credentials":{"app.terraform.io":{"token":"realtoken.atlasv1.xxxx"}}}`, 0o600)
	var flagged bool
	for _, f := range scanTerraform() {
		if f.Kind == "API token" {
			flagged = true
		}
	}
	if !flagged {
		t.Error("a Terraform Cloud token must be flagged")
	}
}

func TestCargoTokenFlagged(t *testing.T) {
	dir := withHome(t)
	write(t, filepath.Join(dir, ".cargo", "credentials.toml"),
		"[registry]\ntoken = \"realcargotoken123\"\n", 0o600)
	var flagged bool
	for _, f := range scanPackageRegistries() {
		if f.Provider == "Cargo" && f.Kind == "registry token" {
			flagged = true
		}
	}
	if !flagged {
		t.Error("a Cargo token must be flagged")
	}
}
