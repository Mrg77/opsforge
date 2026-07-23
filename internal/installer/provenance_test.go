package installer

import (
	"errors"
	"testing"
)

// TestVerifyProvenance exercises the pure decision core: no network, no real
// cosign — the fetcher and verifier are injected fakes (per the constraint
// that cosign is never actually run in tests).
func TestVerifyProvenance(t *testing.T) {
	blob := []byte("checksums")
	sig := []byte("sig")
	cert := []byte("cert")

	presentFetch := func() ([]byte, []byte, bool) { return sig, cert, true }
	absentFetch := func() ([]byte, []byte, bool) { return nil, nil, false }

	okVerify := func(_, _, _ []byte, _, _ string) error { return nil }
	failVerify := func(_, _, _ []byte, _, _ string) error { return errors.New("bad sig") }

	t.Run("cosign absent stays verified (best-effort)", func(t *testing.T) {
		// verify must never be called when cosign is unavailable.
		poison := func(_, _, _ []byte, _, _ string) error {
			t.Fatal("verify called though cosign is unavailable")
			return nil
		}
		st, err := verifyProvenance(blob, false, presentFetch, poison, "id", "iss")
		if st != ChecksumVerified || err != nil {
			t.Fatalf("got (%v,%v), want (Verified,nil)", st, err)
		}
	})

	t.Run("no signature published stays verified", func(t *testing.T) {
		st, err := verifyProvenance(blob, true, absentFetch, okVerify, "id", "iss")
		if st != ChecksumVerified || err != nil {
			t.Fatalf("got (%v,%v), want (Verified,nil)", st, err)
		}
	})

	t.Run("signature present and valid upgrades to signed", func(t *testing.T) {
		st, err := verifyProvenance(blob, true, presentFetch, okVerify, "id", "iss")
		if st != ChecksumSigned || err != nil {
			t.Fatalf("got (%v,%v), want (Signed,nil)", st, err)
		}
	})

	t.Run("signature present but invalid is a hard failure", func(t *testing.T) {
		st, err := verifyProvenance(blob, true, presentFetch, failVerify, "id", "iss")
		if st != ChecksumMismatch || err == nil {
			t.Fatalf("got (%v,%v), want (Mismatch, error)", st, err)
		}
	})

	t.Run("identity and issuer are passed through to the verifier", func(t *testing.T) {
		var gotID, gotIss string
		spy := func(_, _, _ []byte, id, iss string) error {
			gotID, gotIss = id, iss
			return nil
		}
		if _, err := verifyProvenance(blob, true, presentFetch, spy, selfCertIdentityRegexp, oidcIssuerGitHubActions); err != nil {
			t.Fatal(err)
		}
		if gotID != selfCertIdentityRegexp || gotIss != oidcIssuerGitHubActions {
			t.Fatalf("verifier got id=%q issuer=%q, want the pinned self identity/issuer", gotID, gotIss)
		}
	})
}

// TestProvenanceConstants guards the OIDC trust anchors against accidental
// edits: the identity regexp must anchor opsforge's own release workflow and
// the issuer must be GitHub Actions' Sigstore endpoint.
func TestProvenanceConstants(t *testing.T) {
	if oidcIssuerGitHubActions != "https://token.actions.githubusercontent.com" {
		t.Errorf("unexpected OIDC issuer %q", oidcIssuerGitHubActions)
	}
	for _, want := range []string{"Mrg77/opsforge", "release\\.yml"} {
		if !contains(selfCertIdentityRegexp, want) {
			t.Errorf("identity regexp %q missing %q", selfCertIdentityRegexp, want)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
