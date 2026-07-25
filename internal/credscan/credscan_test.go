package credscan

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// withHome points the package's home at a temp dir so scanners read fixtures,
// not the real machine. It resets the memoized home for the test's duration.
func withHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old := home
	home = dir
	t.Cleanup(func() { home = old })
	return dir
}

func write(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

// --- expiry / crypto helpers -------------------------------------------------

// makeCertPEM builds a self-signed cert expiring at notAfter and returns its
// PEM bytes. Test-only; deterministic serial.
func makeCertPEM(t *testing.T, notAfter time.Time) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    notAfter.Add(-24 * time.Hour),
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestCertNotAfterB64(t *testing.T) {
	want := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	b64 := base64.StdEncoding.EncodeToString(makeCertPEM(t, want))
	got, ok := certNotAfterB64(b64)
	if !ok {
		t.Fatal("expected to parse the base64 PEM cert")
	}
	if !got.Equal(want) {
		t.Errorf("NotAfter = %v, want %v", got, want)
	}
}

func TestJWTExpiry(t *testing.T) {
	// Hand-build an unsigned JWT with exp; we never verify the signature.
	exp := time.Date(2031, 6, 1, 0, 0, 0, 0, time.UTC)
	payload, _ := json.Marshal(map[string]int64{"exp": exp.Unix()})
	tok := "eyJhbGciOiJub25lIn0." +
		base64.RawURLEncoding.EncodeToString(payload) + ".sig"
	got, ok := jwtExpiry(tok)
	if !ok {
		t.Fatal("expected to read exp from the JWT")
	}
	if !got.Equal(exp) {
		t.Errorf("exp = %v, want %v", got, exp)
	}

	if _, ok := jwtExpiry("not-a-jwt"); ok {
		t.Error("a non-JWT must not parse")
	}
}

func TestExpirySeverity(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := expirySeverity(now.Add(-time.Hour), now); got != SevHigh {
		t.Errorf("expired => %v, want HIGH", got)
	}
	if got := expirySeverity(now.Add(48*time.Hour), now); got != SevLow {
		t.Errorf("expiring soon => %v, want LOW", got)
	}
	if got := expirySeverity(now.Add(365*24*time.Hour), now); got != SevOK {
		t.Errorf("far future => %v, want OK", got)
	}
}

// --- AWS ---------------------------------------------------------------------

func TestAWSStaticVsTemporary(t *testing.T) {
	dir := withHome(t)
	write(t, filepath.Join(dir, ".aws", "credentials"), `
[static]
aws_access_key_id = AKIAEXAMPLE1234567890
aws_secret_access_key = deadbeefdeadbeefdeadbeefdeadbeefdeadbeef

[sso-session]
aws_access_key_id = ASIAEXAMPLE1234567890
aws_secret_access_key = whatever
aws_session_token = short-lived-token
`, 0o600)

	fs := scanAWS(time.Now())
	var static, temp bool
	for _, f := range fs {
		if f.Kind == "static access key" && f.Severity == SevHigh {
			static = true
		}
		if f.Kind == "temporary STS credential" && f.Severity == SevOK {
			temp = true
		}
	}
	if !static {
		t.Error("an AKIA key with no session token must be flagged HIGH static")
	}
	if !temp {
		t.Error("an ASIA key with a session token must be OK (temporary)")
	}
}

func TestAWSPermissionFinding(t *testing.T) {
	dir := withHome(t)
	// World-readable credentials file.
	write(t, filepath.Join(dir, ".aws", "credentials"),
		"[x]\naws_access_key_id = AKIA000\naws_secret_access_key = s\n", 0o644)

	var perm bool
	for _, f := range scanAWS(time.Now()) {
		if f.Kind == "credentials file" && f.Severity == SevMedium {
			perm = true
		}
	}
	if !perm {
		t.Error("a 0644 credentials file must raise a permission finding")
	}
}

// --- GCP ---------------------------------------------------------------------

func TestGCPServiceAccountKeyIsCritical(t *testing.T) {
	dir := withHome(t)
	adc := filepath.Join(dir, ".config", "gcloud", "application_default_credentials.json")
	write(t, adc, `{"type":"service_account","private_key":"-----BEGIN PRIVATE KEY-----\nx\n-----END PRIVATE KEY-----\n","client_email":"svc@proj.iam"}`, 0o600)

	var crit bool
	for _, f := range scanGCP(time.Now()) {
		if f.Kind == "service-account key" && f.Severity == SevCritical {
			crit = true
		}
	}
	if !crit {
		t.Error("a service_account ADC with a private_key must be CRITICAL")
	}
}

func TestGCPExternalAccountIsOK(t *testing.T) {
	dir := withHome(t)
	adc := filepath.Join(dir, ".config", "gcloud", "application_default_credentials.json")
	write(t, adc, `{"type":"external_account","audience":"//iam.../x"}`, 0o600)

	for _, f := range scanGCP(time.Now()) {
		if f.Severity > SevOK && f.Kind != "credential file" {
			t.Errorf("external_account (federation) must not be risky, got %+v", f)
		}
	}
}
