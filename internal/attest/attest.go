package attest

import (
	"context"
	"fmt"

	"github.com/sigstore/sigstore-go/pkg/sign"
	"google.golang.org/protobuf/encoding/protojson"
)

// Result is a signed artifact: the Sigstore bundle (self-contained JSON with
// signature + public key) plus the PEM public key the user shares so others
// can verify.
type Result struct {
	Bundle       []byte // Sigstore bundle JSON (.sigstore.json)
	PublicKeyPEM string // the opsforge signing key's public half
}

// SignBlob signs raw bytes with the persistent local opsforge key and returns
// a self-contained Sigstore bundle. Fully offline: no Fulcio certificate and
// no Rekor transparency-log entry (empty BundleOptions), so nothing about the
// signer is published anywhere. It proves integrity + attribution to this
// key — not provenance. Verify with:
//
//	cosign verify-blob --key ~/.config/opsforge/signing.pub \
//	  --bundle <artifact>.sigstore.json <artifact>
func SignBlob(ctx context.Context, data []byte) (Result, error) {
	kp, err := loadOrCreateKeypair()
	if err != nil {
		return Result{}, fmt.Errorf("loading signing key: %w", err)
	}
	pubPEM, err := kp.GetPublicKeyPem()
	if err != nil {
		return Result{}, err
	}

	content := &sign.PlainData{Data: data}
	// Empty options → key-based, offline: public key in the bundle, no cert,
	// no Rekor. This is the deliberate local-signing mode (see package doc).
	bundle, err := sign.Bundle(content, kp, sign.BundleOptions{Context: ctx})
	if err != nil {
		return Result{}, fmt.Errorf("signing: %w", err)
	}

	// Marshal to the canonical Sigstore bundle JSON (what cosign consumes).
	bundleJSON, err := protojson.Marshal(bundle)
	if err != nil {
		return Result{}, fmt.Errorf("encoding bundle: %w", err)
	}

	return Result{Bundle: bundleJSON, PublicKeyPEM: pubPEM}, nil
}
