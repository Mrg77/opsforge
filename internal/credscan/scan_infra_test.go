package credscan

import (
	"path/filepath"
	"testing"
	"time"
)

func TestGHKeyringAbsenceIsOK(t *testing.T) {
	dir := withHome(t)
	// A host with no oauth_token means the OS keyring holds it — good hygiene,
	// not "nothing found" and not a leak.
	write(t, filepath.Join(dir, ".config", "gh", "hosts.yml"), `
github.com:
    user: someone
    git_protocol: https
`, 0o600)

	fs := scanSCM()
	for _, f := range fs {
		if f.Severity > SevOK {
			t.Errorf("a keyring-backed gh host must not be risky: %+v", f)
		}
	}
	var okSeen bool
	for _, f := range fs {
		if f.Kind == "keyring-backed token" && f.Severity == SevOK {
			okSeen = true
		}
	}
	if !okSeen {
		t.Error("a tokenless gh host should be reported as keyring-backed (OK)")
	}
}

func TestGHClearTextTokenFlagged(t *testing.T) {
	dir := withHome(t)
	write(t, filepath.Join(dir, ".config", "gh", "hosts.yml"), `
github.com:
    user: someone
    oauth_token: gho_deadbeefdeadbeefdeadbeefdeadbeef
`, 0o600)

	var flagged bool
	for _, f := range scanSCM() {
		if f.Kind == "OAuth token" && f.Severity == SevMedium {
			flagged = true
		}
	}
	if !flagged {
		t.Error("a clear-text gh oauth_token must be flagged")
	}
}

func TestDockerCredStoreIsOK(t *testing.T) {
	dir := withHome(t)
	// A registry delegated to a credential store keeps no secret in the file.
	write(t, filepath.Join(dir, ".docker", "config.json"),
		`{"auths":{"registry.example.com":{}},"credsStore":"desktop"}`, 0o600)

	for _, f := range scanDocker() {
		if f.Severity > SevOK {
			t.Errorf("credsStore delegation must not be flagged: %+v", f)
		}
	}
}

func TestDockerBase64AuthFlagged(t *testing.T) {
	dir := withHome(t)
	// base64("user:password") — base64 is NOT encryption; must be flagged.
	write(t, filepath.Join(dir, ".docker", "config.json"),
		`{"auths":{"registry.example.com":{"auth":"dXNlcjpwYXNzd29yZA=="}}}`, 0o600)

	var flagged bool
	for _, f := range scanDocker() {
		if f.Kind == "registry login" && f.Severity == SevMedium {
			flagged = true
		}
	}
	if !flagged {
		t.Error("a base64 docker login must be flagged as clear text")
	}
}

func TestNpmEnvInterpolationNotFlagged(t *testing.T) {
	dir := withHome(t)
	write(t, filepath.Join(dir, ".npmrc"),
		"//registry.npmjs.org/:_authToken=${NPM_TOKEN}\n", 0o600)

	for _, f := range scanPackageRegistries() {
		if f.Kind == "registry token" {
			t.Error("an ${NPM_TOKEN} interpolation is not a hard-coded secret")
		}
	}
}

func TestNpmInlineTokenFlagged(t *testing.T) {
	dir := withHome(t)
	write(t, filepath.Join(dir, ".npmrc"),
		"//registry.npmjs.org/:_authToken=npm_realtokenvalue1234567890\n", 0o600)

	var flagged bool
	for _, f := range scanPackageRegistries() {
		if f.Kind == "registry token" {
			flagged = true
		}
	}
	if !flagged {
		t.Error("an inline npm token must be flagged")
	}
}

func TestScanIsDeterministicAndSorted(t *testing.T) {
	withHome(t) // empty home: no credentials anywhere
	r := Scan(time.Now())
	if len(r.Risky()) != 0 {
		t.Errorf("an empty home must yield no risky findings, got %+v", r.Risky())
	}
	if r.TopSeverity() != SevOK {
		t.Errorf("empty scan TopSeverity = %v, want OK", r.TopSeverity())
	}
}
