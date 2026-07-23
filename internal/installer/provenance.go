package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Mrg77/opsforge/internal/catalog"
)

// This file adds a supply-chain layer on top of checksum verification: when a
// release publishes a cosign KEYLESS signature over its checksums file, and
// cosign is available locally, we verify that signature against the expected
// OIDC identity. A verified signature upgrades ChecksumVerified to
// ChecksumSigned (checksum intact AND provably built by the trusted workflow).
//
// The whole thing is best-effort by design: the vast majority of third-party
// GitHub tools in the catalog publish no signature, and users may not have
// cosign installed. The one release we sign ourselves is opsforge's own
// (Mrg77/opsforge), which is exactly where the self-update path opts in.

// selfCertIdentityRegexp matches the OIDC identity of opsforge's release
// workflow. cosign binds the keyless certificate to the workflow ref, e.g.
// https://github.com/Mrg77/opsforge/.github/workflows/release.yml@refs/tags/v0.5.0
// so we anchor the start and allow any trailing @ref.
const selfCertIdentityRegexp = `^https://github.com/Mrg77/opsforge/\.github/workflows/release\.yml@.*`

// oidcIssuerGitHubActions is the Sigstore OIDC issuer for GitHub Actions
// keyless signing. It is the trust anchor: the certificate must have been
// issued off a token from this issuer, not any other identity provider.
const oidcIssuerGitHubActions = "https://token.actions.githubusercontent.com"

// signatureFetcher retrieves the signature (.sig) and certificate (.pem)
// bytes published for a checksum file, reporting ok=false when either is
// absent. Split out so verifyProvenance's decision logic is unit-testable
// without a network or a real cosign process (see the verifier seam too).
type signatureFetcher func() (sig, cert []byte, ok bool)

// blobVerifier verifies a keyless cosign signature over blob (the raw
// checksum-file bytes) using sig/cert and the given identity/issuer. It
// returns nil on a good signature and an error otherwise. The production
// implementation shells out to cosign; tests inject a fake so the exec is
// never run in CI (per the no-real-cosign constraint).
type blobVerifier func(blob, sig, cert []byte, identityRegexp, issuer string) error

// verifyProvenance decides whether a matched checksum can be UPGRADED to
// ChecksumSigned. It is the pure decision core — all I/O (fetching the .sig
// and .pem, running cosign) is injected — so every branch is unit-testable.
//
// Contract, layered strictly on top of the checksum result:
//   - tooling/signature absent  → (ChecksumVerified, nil): best-effort, the
//     checksum already matched, we simply couldn't do better. Never a failure.
//   - signature present, verifies → (ChecksumSigned, nil): strongest state.
//   - signature present, FAILS    → (ChecksumMismatch, err): a published
//     signature that does not verify is tampering; the caller MUST abort,
//     exactly like a checksum mismatch.
//
// cosignAvailable reports whether the cosign binary is usable locally; when
// false we short-circuit to ChecksumVerified without touching fetch/verify.
func verifyProvenance(
	blob []byte,
	cosignAvailable bool,
	fetch signatureFetcher,
	verify blobVerifier,
	identityRegexp, issuer string,
) (ChecksumStatus, error) {
	if !cosignAvailable {
		return ChecksumVerified, nil
	}
	sig, cert, ok := fetch()
	if !ok {
		// No signature published for this release: best-effort, stay verified.
		return ChecksumVerified, nil
	}
	if err := verify(blob, sig, cert, identityRegexp, issuer); err != nil {
		// A signature WAS published and did not verify — treat as tampering.
		return ChecksumMismatch, fmt.Errorf(
			"signature verification failed for the checksums file: %w (refusing to install)", err)
	}
	return ChecksumSigned, nil
}

// cosignAvailable reports whether a cosign binary is on PATH. Kept as a var so
// tests can observe the seam; production uses exec.LookPath.
var cosignAvailable = func() bool {
	_, err := exec.LookPath("cosign")
	return err == nil
}

// cosignVerifyBlob is the production blobVerifier: it writes blob/sig/cert to
// temp files and runs `cosign verify-blob` in keyless mode against the given
// identity and issuer. cosign v2 needs no COSIGN_EXPERIMENTAL for this.
func cosignVerifyBlob(blob, sig, cert []byte, identityRegexp, issuer string) error {
	dir, err := os.MkdirTemp("", "opsforge-cosign-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	blobPath := filepath.Join(dir, "checksums.txt")
	sigPath := filepath.Join(dir, "checksums.txt.sig")
	certPath := filepath.Join(dir, "checksums.txt.pem")
	for _, w := range []struct {
		path string
		data []byte
	}{{blobPath, blob}, {sigPath, sig}, {certPath, cert}} {
		if err := os.WriteFile(w.path, w.data, 0o600); err != nil {
			return err
		}
	}

	cmd := exec.Command("cosign", "verify-blob",
		"--certificate", certPath,
		"--signature", sigPath,
		"--certificate-identity-regexp", identityRegexp,
		"--certificate-oidc-issuer", issuer,
		blobPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cosign verify-blob: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// verifyChecksumProvenance runs the normal checksum verification and, when it
// passes, opportunistically upgrades the result to ChecksumSigned by verifying
// the cosign keyless signature over the checksum file. It is the entry point
// the self-update path uses (opsforge signs its own checksums.txt).
//
// Behaviour is a strict, non-regressing superset of verifyChecksum:
//   - any non-verified checksum result (unavailable / mismatch / I/O error) is
//     returned unchanged — provenance is only ever attempted on an already
//     matched checksum;
//   - a matched checksum with a good signature returns ChecksumSigned;
//   - a matched checksum with a bad signature returns ChecksumMismatch (hard
//     failure — a published-but-invalid signature means tampering);
//   - a matched checksum with no signature / no cosign stays ChecksumVerified.
//
// checksumName is the release-asset name whose signature to look for
// (checksums.txt for our own releases). identityRegexp/issuer pin the expected
// keyless signer.
func verifyChecksumProvenance(
	gh *catalog.GitHubRelease,
	repo, tag, asset, archivePath, checksumName, identityRegexp, issuer string,
) (ChecksumStatus, error) {
	status, err := verifyChecksum(gh, repo, tag, asset, archivePath)
	if status != ChecksumVerified || err != nil {
		return status, err
	}
	blob, ok := fetchBytes(
		fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, checksumName))
	if !ok {
		// The checksum matched but we can't re-fetch its file to verify a
		// signature over it — stay verified, don't invent a failure.
		return ChecksumVerified, nil
	}
	return verifyProvenance(
		blob,
		cosignAvailable(),
		fetchSignaturePair(repo, tag, checksumName),
		cosignVerifyBlob,
		identityRegexp, issuer,
	)
}

// fetchSignaturePair fetches "<checksumName>.sig" and "<checksumName>.pem"
// from a release, returning ok=false unless BOTH are present (cosign needs the
// certificate to establish the keyless identity). It is the production
// signatureFetcher used by verifyChecksumProvenance.
func fetchSignaturePair(repo, tag, checksumName string) signatureFetcher {
	return func() (sig, cert []byte, ok bool) {
		base := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, checksumName)
		s, sOK := fetchBytes(base + ".sig")
		c, cOK := fetchBytes(base + ".pem")
		if !sOK || !cOK {
			return nil, nil, false
		}
		return s, c, true
	}
}
