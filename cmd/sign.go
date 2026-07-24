package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/Mrg77/opsforge/internal/attest"
	"github.com/Mrg77/opsforge/internal/ui"
)

// signArtifact signs the bytes of a supply-chain artifact (an SBOM or VEX
// document) with the local opsforge key and writes a self-contained Sigstore
// bundle next to where the user is capturing the document. bundlePath is the
// output file (e.g. "bom.sigstore.json"); label names the artifact in the
// human feedback. All feedback goes to stderr so a piped document on stdout
// stays clean.
func signArtifact(ctx context.Context, data []byte, bundlePath, label string) error {
	res, err := attest.SignBlob(ctx, data)
	if err != nil {
		return err
	}
	if err := os.WriteFile(bundlePath, res.Bundle, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", bundlePath, err)
	}

	_, pubPath, _ := attest.KeyPaths()
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, ui.OKMark()+" "+ui.OKBold.Render(fmt.Sprintf("Signed the %s", label)))
	fmt.Fprintln(os.Stderr, ui.Dim.Render("  bundle:     "+bundlePath))
	fmt.Fprintln(os.Stderr, ui.Dim.Render("  public key: "+pubPath))
	fmt.Fprintln(os.Stderr, ui.Dim.Render("  key-based & offline — no OIDC, no Rekor entry. Proves integrity"))
	fmt.Fprintln(os.Stderr, ui.Dim.Render("  + attribution to this key, not provenance."))
	fmt.Fprintln(os.Stderr, ui.Faint.Render(fmt.Sprintf(
		"  verify:  cosign verify-blob --key %s --bundle %s <the-document>", pubPath, bundlePath)))
	return nil
}
