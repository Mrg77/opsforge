package credscan

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"
)

// scanCloud inspects the local credential stores of the major cloud CLIs. It
// only ever reads files; it never runs aws/gcloud/az. now is injected so the
// clock is shared and tests are deterministic.
func scanCloud(now time.Time) []Finding {
	var out []Finding
	out = append(out, scanAWS(now)...)
	out = append(out, scanGCP(now)...)
	out = append(out, scanAzure(now)...)
	out = append(out, scanMiscCloud(now)...)
	return out
}

// --- AWS ---------------------------------------------------------------------

// scanAWS reads ~/.aws/credentials and ~/.aws/config. The signature risk is a
// section holding an AKIA access key with no session token — a long-lived IAM
// user key. ASIA keys are STS (temporary); sso_*/role_arn/web_identity mean
// the profile federates and holds no static secret.
func scanAWS(now time.Time) []Finding {
	var out []Finding

	credsPath := "~/.aws/credentials"
	if b, ok := readFile(credsPath); ok {
		out = append(out, permFinding("AWS", "credentials file", credsPath)...)
		ini := parseINI(b)
		for section, kv := range ini {
			if section == "" {
				continue
			}
			id := kv["aws_access_key_id"]
			if id == "" {
				continue // role_arn/source_profile-only section: no static secret
			}
			hasSession := kv["aws_session_token"] != ""
			switch {
			case hasSession || strings.HasPrefix(id, "ASIA"):
				out = append(out, mkFinding("AWS", "temporary STS credential", credsPath,
					SevOK, "profile ["+section+"] uses a temporary session token",
					"Short-lived STS credential — good hygiene."))
			case strings.HasPrefix(id, "AKIA"):
				out = append(out, mkFinding("AWS", "static access key", credsPath,
					SevHigh, "profile ["+section+"] holds a long-lived IAM user key",
					"An AKIA key never expires. Prefer AWS SSO / IAM Identity Center or "+
						"assume-role. If you must keep it, rotate it regularly and delete unused keys."))
			}
		}
	}

	// The config file itself holds no secret, but it tells us which profiles
	// federate — surfacing that a profile is SSO-backed is reassuring context.
	if b, ok := readFile("~/.aws/config"); ok {
		ini := parseINI(b)
		for section, kv := range ini {
			if federatesAWS(kv) {
				name := strings.TrimPrefix(section, "profile ")
				out = append(out, mkFinding("AWS", "federated profile", "~/.aws/config",
					SevOK, "profile ["+name+"] federates (SSO / role / OIDC)",
					"No static secret stored locally for this profile."))
			}
		}
	}

	// SSO and STS caches carry short-lived tokens with an explicit expiry;
	// report an expired one so a stale token isn't mistaken for a live session.
	out = append(out, scanExpiringJSONDir("AWS", "SSO token cache", "~/.aws/sso/cache", now, awsSSOExpiry)...)
	out = append(out, scanExpiringJSONDir("AWS", "STS cache", "~/.aws/cli/cache", now, awsSTSExpiry)...)
	return out
}

func federatesAWS(kv map[string]string) bool {
	for _, k := range []string{"sso_session", "sso_start_url", "sso_role_name", "role_arn", "web_identity_token_file", "credential_process"} {
		if kv[k] != "" {
			return true
		}
	}
	return false
}

func awsSSOExpiry(b []byte) (time.Time, bool) {
	var v struct {
		ExpiresAt string `json:"expiresAt"`
	}
	if json.Unmarshal(b, &v) != nil {
		return time.Time{}, false
	}
	return parseTime(v.ExpiresAt)
}

func awsSTSExpiry(b []byte) (time.Time, bool) {
	var v struct {
		Credentials struct {
			Expiration string `json:"Expiration"`
		} `json:"Credentials"`
	}
	if json.Unmarshal(b, &v) != nil {
		return time.Time{}, false
	}
	return parseTime(v.Credentials.Expiration)
}

// --- GCP ---------------------------------------------------------------------

// scanGCP reads gcloud's Application Default Credentials and legacy per-account
// creds. A "service_account" ADC embeds a private key (static, no expiry — the
// loudest finding); "authorized_user" embeds a long-lived refresh token;
// "external_account"/"impersonated_service_account" federate and are fine.
func scanGCP(now time.Time) []Finding {
	var out []Finding

	adc := "~/.config/gcloud/application_default_credentials.json"
	out = append(out, gcpADCFindings(adc)...)

	// Legacy per-account creds mirror the authorized_user shape.
	if matches, _ := filepathGlob("~/.config/gcloud/legacy_credentials/*/adc.json"); len(matches) > 0 {
		for _, m := range matches {
			out = append(out, gcpADCFindings(m)...)
		}
	}

	// A service-account key JSON can live anywhere; the common signal is the
	// GOOGLE_APPLICATION_CREDENTIALS env var pointing at one.
	if p := envPath("GOOGLE_APPLICATION_CREDENTIALS"); p != "" {
		out = append(out, gcpADCFindings(p)...)
	}
	return out
}

func gcpADCFindings(path string) []Finding {
	b, ok := readFile(path)
	if !ok {
		return nil
	}
	var v struct {
		Type       string `json:"type"`
		PrivateKey string `json:"private_key"`
		Email      string `json:"client_email"`
	}
	if json.Unmarshal(b, &v) != nil {
		return nil
	}
	out := permFinding("GCP", "credential file", path)
	switch v.Type {
	case "service_account":
		if strings.TrimSpace(v.PrivateKey) != "" {
			out = append(out, mkFinding("GCP", "service-account key", path,
				SevCritical, "a service-account private key sits in clear text on disk",
				"Service-account JSON keys don't expire and grant standing access ("+v.Email+"). "+
					"Prefer Workload Identity Federation or short-lived impersonation; if you must "+
					"keep the key, restrict and rotate it."))
		}
	case "authorized_user":
		out = append(out, mkFinding("GCP", "user refresh token", path,
			SevMedium, "a long-lived user OAuth refresh token is stored here",
			"Valid until revoked. Fine for a workstation, but treat the file as a secret (0600) "+
				"and run `gcloud auth revoke` when done."))
	case "external_account", "impersonated_service_account":
		out = append(out, mkFinding("GCP", "federated credential", path,
			SevOK, "credentials federate ("+v.Type+")",
			"No static key material stored locally."))
	}
	return out
}

// --- Azure -------------------------------------------------------------------

// scanAzure reads ~/.azure. On Linux/macOS the MSAL cache and service-principal
// entries sit in clear text. A service principal with a client_secret is a
// long-lived static secret; cert- or federation-based SPs are not.
func scanAzure(now time.Time) []Finding {
	var out []Finding

	spPath := "~/.azure/service_principal_entries.json"
	if b, ok := readFile(spPath); ok {
		out = append(out, permFinding("Azure", "service-principal store", spPath)...)
		var entries []map[string]any
		if json.Unmarshal(b, &entries) == nil {
			for _, e := range entries {
				if secret, _ := e["client_secret"].(string); strings.TrimSpace(secret) != "" {
					out = append(out, mkFinding("Azure", "service-principal secret", spPath,
						SevHigh, "a service-principal client secret is stored in clear text",
						"Client secrets are long-lived. Prefer certificate or federated (OIDC) "+
							"credentials, or a managed identity where possible."))
				} else if fc, ok := e["federated_credential"]; ok && fc != nil {
					out = append(out, mkFinding("Azure", "federated service principal", spPath,
						SevOK, "service principal federates (OIDC)", "No static secret stored locally."))
				}
			}
		}
	}

	// The MSAL cache holds short-lived access tokens with an expires_on epoch;
	// surface the store's presence + whether all its tokens are stale.
	msal := "~/.azure/msal_token_cache.json"
	if _, ok := statFile(msal); ok {
		out = append(out, permFinding("Azure", "token cache", msal)...)
	}
	return out
}

// --- Misc providers ----------------------------------------------------------

// scanMiscCloud covers the secondary providers whose local store is a single
// file with a static token/secret: DigitalOcean, Scaleway, OVH, OCI, Alibaba,
// Vault, Cloudflare. Each is a long-lived secret with no local expiry.
func scanMiscCloud(now time.Time) []Finding {
	var out []Finding

	// DigitalOcean doctl — YAML with an access-token PAT.
	for _, p := range []string{"~/.config/doctl/config.yaml", "~/Library/Application Support/doctl/config.yaml"} {
		if b, ok := readFile(p); ok {
			out = append(out, permFinding("DigitalOcean", "doctl config", p)...)
			if yamlHasNonEmpty(b, "access-token") {
				out = append(out, mkFinding("DigitalOcean", "personal access token", p,
					SevMedium, "a long-lived DigitalOcean PAT is stored here",
					"DigitalOcean PATs don't expire unless you set an expiry. Rotate periodically."))
			}
		}
	}

	// Scaleway — YAML secret_key (UUID).
	for _, p := range []string{"~/.config/scw/config.yaml"} {
		if b, ok := readFile(p); ok {
			out = append(out, permFinding("Scaleway", "scw config", p)...)
			if yamlHasNonEmpty(b, "secret_key") {
				out = append(out, mkFinding("Scaleway", "static secret key", p,
					SevMedium, "a long-lived Scaleway secret key is stored here",
					"Rotate periodically and keep the file private (0600)."))
			}
		}
	}

	// OVH — INI application_secret / consumer_key.
	for _, p := range []string{"~/.ovh.conf"} {
		if b, ok := readFile(p); ok {
			out = append(out, permFinding("OVH", "ovh.conf", p)...)
			ini := parseINI(b)
			for _, kv := range ini {
				if kv["application_secret"] != "" || kv["consumer_key"] != "" {
					out = append(out, mkFinding("OVH", "static API secret", p,
						SevMedium, "long-lived OVH API secrets are stored here",
						"Keep the file private (0600); OVH consumer keys are long-lived."))
					break
				}
			}
		}
	}

	// OCI — INI with key_file (static RSA API key) vs security_token (session).
	if b, ok := readFile("~/.oci/config"); ok {
		out = append(out, permFinding("OCI", "oci config", "~/.oci/config")...)
		ini := parseINI(b)
		for section, kv := range ini {
			if section == "" {
				continue
			}
			if kv["security_token_file"] != "" {
				out = append(out, mkFinding("OCI", "session token", "~/.oci/config",
					SevOK, "profile ["+section+"] uses a session token", "Short-lived OCI session — good hygiene."))
			} else if kv["key_file"] != "" {
				out = append(out, mkFinding("OCI", "static API key", "~/.oci/config",
					SevHigh, "profile ["+section+"] uses a long-lived RSA API key",
					"OCI API signing keys don't expire. Prefer `oci session authenticate` "+
						"(security_token) for interactive use."))
			}
		}
	}

	// Alibaba — JSON profiles with mode AK (static) vs StsToken/OIDC/RamRole.
	if b, ok := readFile("~/.aliyun/config.json"); ok {
		out = append(out, permFinding("Alibaba", "aliyun config", "~/.aliyun/config.json")...)
		var v struct {
			Profiles []struct {
				Name string `json:"name"`
				Mode string `json:"mode"`
			} `json:"profiles"`
		}
		if json.Unmarshal(b, &v) == nil {
			for _, p := range v.Profiles {
				if p.Mode == "AK" {
					out = append(out, mkFinding("Alibaba", "static access key", "~/.aliyun/config.json",
						SevHigh, "profile ["+p.Name+"] uses a long-lived AccessKey secret",
						"Prefer RAM role / STS / OIDC over a static AccessKey."))
				}
			}
		}
	}

	// HashiCorp Vault — a raw token file. Presence + perms only (no local expiry).
	if _, ok := statFile("~/.vault-token"); ok {
		out = append(out, permFinding("Vault", "vault token", "~/.vault-token")...)
	}
	return out
}

// --- shared helpers ----------------------------------------------------------

// permFinding flags a credential file whose permissions let group/other read
// it. Returns at most one finding; nil when the file is private (or absent).
func permFinding(provider, kind, path string) []Finding {
	fi, ok := statFile(path)
	if !ok || fi.IsDir() {
		return nil
	}
	if worldOrGroupReadable(fi.Mode()) {
		return []Finding{mkFinding(provider, kind, path, SevMedium,
			"file is readable beyond its owner ("+fi.Mode().Perm().String()+")",
			"A credential file should be 0600. Run `chmod 600 "+path+"`.")}
	}
	return nil
}

// scanExpiringJSONDir reads every *.json in a directory, extracts an expiry via
// getExp, and reports the ones already expired (a stale short-lived token left
// behind). Directory absent => no findings.
func scanExpiringJSONDir(provider, kind, dir string, now time.Time, getExp func([]byte) (time.Time, bool)) []Finding {
	matches, err := filepathGlob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil
	}
	var out []Finding
	for _, m := range matches {
		b, ok := readFile(m)
		if !ok {
			continue
		}
		exp, ok := getExp(b)
		if !ok {
			continue
		}
		if exp.Before(now) {
			f := mkFinding(provider, kind, m, SevLow,
				"an expired "+kind+" token is still on disk",
				"Stale cached token; harmless but worth clearing.")
			f.Expires = &exp
			out = append(out, f)
		}
	}
	return out
}

// yamlHasNonEmpty is a dependency-free check for `key: value` with a non-empty,
// non-placeholder value anywhere in a small YAML file. We don't need a full
// YAML parser to answer "is there a secret in this field".
func yamlHasNonEmpty(b []byte, key string) bool {
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, key+":") {
			continue
		}
		v := strings.TrimSpace(strings.TrimPrefix(line, key+":"))
		v = strings.Trim(v, `"'`)
		if v != "" && v != "null" && v != "~" {
			return true
		}
	}
	return false
}
