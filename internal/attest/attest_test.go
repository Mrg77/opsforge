package attest

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"os"
	"strings"
	"testing"
)

func TestSignBlobProducesBundleAndPersistsKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	res, err := SignBlob(context.Background(), []byte(`{"hello":"world"}`))
	if err != nil {
		t.Fatalf("SignBlob: %v", err)
	}

	// The bundle is valid JSON with a Sigstore media type.
	var bundle map[string]any
	if err := json.Unmarshal(res.Bundle, &bundle); err != nil {
		t.Fatalf("bundle is not valid JSON: %v", err)
	}
	if mt, _ := bundle["mediaType"].(string); !strings.Contains(mt, "sigstore") {
		t.Errorf("bundle mediaType looks wrong: %q", mt)
	}

	// The public key is real PEM and the key files were persisted.
	if !strings.Contains(res.PublicKeyPEM, "BEGIN PUBLIC KEY") {
		t.Errorf("public key PEM malformed: %q", res.PublicKeyPEM)
	}
	priv, pub, _ := KeyPaths()
	if _, err := os.Stat(priv); err != nil {
		t.Errorf("private key not persisted: %v", err)
	}
	if _, err := os.Stat(pub); err != nil {
		t.Errorf("public key not persisted: %v", err)
	}

	// Private key must be written with restrictive perms.
	if fi, err := os.Stat(priv); err == nil && fi.Mode().Perm() != 0o600 {
		t.Errorf("private key perms = %v, want 0600", fi.Mode().Perm())
	}
}

func TestSignBlobReusesPersistedKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	first, err := SignBlob(context.Background(), []byte("a"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := SignBlob(context.Background(), []byte("b"))
	if err != nil {
		t.Fatal(err)
	}
	// Same persistent key → identical public key across signatures (this is
	// the whole point of the local keypair vs an ephemeral one).
	if first.PublicKeyPEM != second.PublicKeyPEM {
		t.Error("public key changed between signatures; the key should persist")
	}
}

// TestBundleVerifiesAgainstSignedBytes is the interop guard: the bundle's
// digest must be the SHA-256 of the exact bytes passed to SignBlob, and the
// signature must verify over that digest with the persisted public key. This
// is what makes `cosign verify-blob <the-file>` succeed — a mismatch (e.g. a
// stray trailing newline between the signed bytes and the written file) would
// silently break verification.
func TestBundleVerifiesAgainstSignedBytes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	payload := []byte(`{"bomFormat":"CycloneDX"}` + "\n")
	res, err := SignBlob(context.Background(), payload)
	if err != nil {
		t.Fatal(err)
	}

	var bundle struct {
		MessageSignature struct {
			MessageDigest struct {
				Digest string `json:"digest"`
			} `json:"messageDigest"`
			Signature string `json:"signature"`
		} `json:"messageSignature"`
	}
	if err := json.Unmarshal(res.Bundle, &bundle); err != nil {
		t.Fatal(err)
	}

	// The bundle digest describes exactly the bytes we signed.
	sum := sha256.Sum256(payload)
	if got := base64.StdEncoding.EncodeToString(sum[:]); got != bundle.MessageSignature.MessageDigest.Digest {
		t.Fatalf("bundle digest %q != digest of signed bytes %q",
			bundle.MessageSignature.MessageDigest.Digest, got)
	}

	// The signature verifies over that digest with the emitted public key.
	block, _ := pem.Decode([]byte(res.PublicKeyPEM))
	if block == nil {
		t.Fatal("public key PEM did not decode")
	}
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	pub, ok := pubAny.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("public key is not ECDSA: %T", pubAny)
	}
	sig, err := base64.StdEncoding.DecodeString(bundle.MessageSignature.Signature)
	if err != nil {
		t.Fatal(err)
	}
	if !ecdsa.VerifyASN1(pub, sum[:], sig) {
		t.Error("signature did not verify against the public key and signed bytes")
	}
}

func TestSignBlobDistinctDataDistinctSignatures(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	a, err := SignBlob(context.Background(), []byte("data-a"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := SignBlob(context.Background(), []byte("data-b"))
	if err != nil {
		t.Fatal(err)
	}
	// Different payloads must yield different bundles (the signature differs).
	if string(a.Bundle) == string(b.Bundle) {
		t.Error("distinct payloads produced identical bundles")
	}
}
