package credscan

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strings"
	"time"
)

// certNotAfter parses the first CERTIFICATE block out of a PEM blob and
// returns its NotAfter. Purely local: it decodes the certificate, it never
// contacts a CA or runs a tool. ok is false when the blob holds no parsable
// certificate.
func certNotAfter(pemBytes []byte) (t time.Time, ok bool) {
	rest := pemBytes
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			return time.Time{}, false
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}
		return cert.NotAfter, true
	}
}

// certNotAfterB64 decodes a base64-encoded PEM (the shape kubeconfig uses for
// client-certificate-data) and returns the certificate's NotAfter — without
// ever invoking kubectl.
func certNotAfterB64(b64 string) (time.Time, bool) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return time.Time{}, false
	}
	return certNotAfter(raw)
}

// jwtExpiry reads the `exp` claim from a JWT WITHOUT verifying its signature —
// we only want the expiry for hygiene reporting, not to trust the token. ok is
// false when the string isn't a three-part JWT or has no numeric exp.
func jwtExpiry(token string) (time.Time, bool) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Some encoders pad; try standard base64 as a fallback.
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return time.Time{}, false
		}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.Exp == 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0).UTC(), true
}

// expirySeverity maps a time-to-expiry to a hygiene severity, relative to now.
// Already-expired credentials are the loudest (they're either dead weight or a
// sign something is refreshing them silently); near-expiry is a heads-up.
//
// now is passed in (not time.Now()) so scanners share one clock and tests are
// deterministic.
func expirySeverity(expires, now time.Time) Severity {
	switch {
	case expires.Before(now):
		return SevHigh // expired: stale credential left on disk
	case expires.Before(now.Add(7 * 24 * time.Hour)):
		return SevLow // expires within a week: informational heads-up
	default:
		return SevOK
	}
}
