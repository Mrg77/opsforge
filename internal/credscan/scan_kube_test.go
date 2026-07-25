package credscan

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func writeKubeconfig(t *testing.T, body string) func() {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KUBECONFIG", p)
	return func() {}
}

// TestKubeOIDCReadPassively is the load-bearing test: an OIDC kubeconfig (the
// shape that would trigger a Keycloak prod login if kubectl ran) is inspected
// by parsing YAML only. We assert the refresh-token/client-secret is the HIGH
// finding — never the id-token — and that scanning it needs no exec.
func TestKubeOIDCReadPassively(t *testing.T) {
	writeKubeconfig(t, `
apiVersion: v1
kind: Config
users:
- name: prod-oidc
  user:
    auth-provider:
      name: oidc
      config:
        idp-issuer-url: https://keycloak.example/realms/prod
        client-id: kubernetes
        client-secret: super-secret
        refresh-token: long-lived-refresh
        id-token: eyJhbGciOiJub25lIn0.e30.sig
`)
	fs := scanKube(time.Now())
	var high bool
	for _, f := range fs {
		if f.Kind == "OIDC refresh credential" && f.Severity == SevHigh {
			high = true
		}
		if f.Kind == "bearer token" || f.Title == "id-token" {
			t.Errorf("the id-token must not be flagged on its own: %+v", f)
		}
	}
	if !high {
		t.Error("an OIDC refresh-token/client-secret must be a HIGH finding")
	}
}

func TestKubeExecIsOK(t *testing.T) {
	writeKubeconfig(t, `
users:
- name: eks
  user:
    exec:
      command: aws
      args: ["eks","get-token","--cluster-name","prod"]
`)
	for _, f := range scanKube(time.Now()) {
		if f.Severity > SevOK {
			t.Errorf("an exec-plugin user holds no static secret, must not be risky: %+v", f)
		}
	}
}

func TestKubeExpiredClientCert(t *testing.T) {
	// A client cert that expired yesterday must be HIGH.
	past := time.Now().Add(-24 * time.Hour)
	certB64 := base64.StdEncoding.EncodeToString(makeCertPEM(t, past))
	writeKubeconfig(t, `
users:
- name: admin
  user:
    client-certificate-data: `+certB64+`
    client-key-data: `+base64.StdEncoding.EncodeToString([]byte("-----BEGIN PRIVATE KEY-----\nx\n-----END PRIVATE KEY-----"))+`
`)
	var expired bool
	for _, f := range scanKube(time.Now()) {
		if f.Kind == "client certificate" && f.Severity == SevHigh && f.Expires != nil {
			expired = true
		}
	}
	if !expired {
		t.Error("an expired client certificate must be HIGH with an Expires timestamp")
	}
}

func TestKubeLegacyStaticTokenIsHigh(t *testing.T) {
	// An opaque (non-JWT) token has no expiry => long-lived => HIGH.
	writeKubeconfig(t, `
users:
- name: legacy
  user:
    token: abcdef.opaque.legacy.sa.token.not.a.jwt.value
`)
	var high bool
	for _, f := range scanKube(time.Now()) {
		if f.Kind == "static bearer token" && f.Severity == SevHigh {
			high = true
		}
	}
	if !high {
		t.Error("a legacy opaque token (no exp) must be HIGH")
	}
}

// --- SSH ---------------------------------------------------------------------

func writeEd25519Key(t *testing.T, path string, encrypted bool, mode os.FileMode) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var block *pem.Block
	if encrypted {
		b, err := ssh.MarshalPrivateKeyWithPassphrase(priv, "", []byte("hunter2"))
		if err != nil {
			t.Fatal(err)
		}
		block = b
	} else {
		b, err := ssh.MarshalPrivateKey(priv, "")
		if err != nil {
			t.Fatal(err)
		}
		block = b
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), mode); err != nil {
		t.Fatal(err)
	}
}

func TestSSHUnencryptedKeyFlagged(t *testing.T) {
	dir := withHome(t)
	writeEd25519Key(t, filepath.Join(dir, ".ssh", "id_ed25519"), false, 0o600)

	var unenc bool
	for _, f := range scanSSH() {
		if f.Kind == "unencrypted private key" && f.Severity == SevHigh {
			unenc = true
		}
	}
	if !unenc {
		t.Error("a passphrase-less private key must be flagged HIGH")
	}
}

func TestSSHEncryptedKeyIsOK(t *testing.T) {
	dir := withHome(t)
	writeEd25519Key(t, filepath.Join(dir, ".ssh", "id_ed25519"), true, 0o600)

	for _, f := range scanSSH() {
		if f.Kind == "unencrypted private key" {
			t.Error("a passphrase-protected key must not be flagged as unencrypted")
		}
	}
}

func TestSSHIgnoresPublicKey(t *testing.T) {
	dir := withHome(t)
	// A stray world-readable .pub must never be flagged.
	write(t, filepath.Join(dir, ".ssh", "id_ed25519.pub"), "ssh-ed25519 AAAA... user@host\n", 0o644)

	if fs := scanSSH(); len(fs) != 0 {
		t.Errorf("a .pub file must produce no findings, got %+v", fs)
	}
}

func TestSSHWeakRSA(t *testing.T) {
	dir := withHome(t)
	key, err := rsa.GenerateKey(rand.Reader, 1024) //nolint:gosec // intentionally weak for the test
	if err != nil {
		t.Fatal(err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der})
	write(t, filepath.Join(dir, ".ssh", "weak"), string(pemBytes), 0o600)

	var weak bool
	for _, f := range scanSSH() {
		if f.Kind == "weak key" && f.Severity == SevMedium {
			weak = true
		}
	}
	if !weak {
		t.Error("a 1024-bit RSA key must be flagged weak")
	}
}
