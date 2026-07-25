package credscan

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// kubeconfig is the subset of a kubeconfig we parse. We only read it — we never
// invoke kubectl. On this class of machine an OIDC kubeconfig would trigger a
// Keycloak prod login if kubectl ran, so parsing the YAML directly is the whole
// point: we inspect the stored credentials without ever authenticating.
type kubeconfig struct {
	Users []struct {
		Name string   `yaml:"name"`
		User kubeUser `yaml:"user"`
	} `yaml:"users"`
}

type kubeUser struct {
	ClientCertData string `yaml:"client-certificate-data"`
	ClientKeyData  string `yaml:"client-key-data"`
	Token          string `yaml:"token"`
	TokenFile      string `yaml:"tokenFile"`
	Exec           *struct {
		Command string `yaml:"command"`
	} `yaml:"exec"`
	AuthProvider *struct {
		Name   string            `yaml:"name"`
		Config map[string]string `yaml:"config"`
	} `yaml:"auth-provider"`
}

// scanKube walks every kubeconfig ($KUBECONFIG may list several) and reports,
// per user entry, the credential mechanism and its hygiene. Purely file-based.
func scanKube(now time.Time) []Finding {
	var out []Finding
	for _, path := range kubeconfigPaths() {
		b, ok := readFile(path)
		if !ok {
			continue
		}
		out = append(out, permFinding("Kubernetes", "kubeconfig", path)...)

		var kc kubeconfig
		if yaml.Unmarshal(b, &kc) != nil {
			continue
		}
		for _, u := range kc.Users {
			out = append(out, kubeUserFindings(path, u.Name, u.User, now)...)
		}
	}
	return out
}

// kubeUserFindings classifies one kubeconfig user. Order matters: exec/plugin
// users hold NO local secret (a common false positive), a client cert carries
// a readable expiry, a legacy token has none, and an OIDC auth-provider's risk
// is the long-lived refresh-token/client-secret — never the id-token, which is
// expected to be expired at rest.
func kubeUserFindings(path, name string, u kubeUser, now time.Time) []Finding {
	who := "user \"" + name + "\""

	// exec plugin: the token is minted on demand by an external command; there
	// is no static secret in the kubeconfig. Don't flag it as a leak.
	if u.Exec != nil {
		return []Finding{mkFinding("Kubernetes", "exec credential", path,
			SevOK, who+" mints tokens via an exec plugin ("+u.Exec.Command+")",
			"No static secret stored in the kubeconfig for this user.")}
	}

	// OIDC auth-provider (kubectl's legacy OIDC): the durable risk is the
	// refresh-token + client-secret sitting in clear text, not the id-token.
	if u.AuthProvider != nil && strings.EqualFold(u.AuthProvider.Name, "oidc") {
		cfg := u.AuthProvider.Config
		if cfg["refresh-token"] != "" || cfg["client-secret"] != "" {
			return []Finding{mkFinding("Kubernetes", "OIDC refresh credential", path,
				SevHigh, who+" stores a long-lived OIDC refresh token / client secret in clear text",
				"The id-token being expired at rest is normal; the risk is the refresh-token and "+
					"client-secret next to it. Keep the kubeconfig private (0600) and prefer an "+
					"exec-based OIDC plugin (e.g. kubelogin) that doesn't persist the client secret.")}
		}
		return nil
	}

	var out []Finding

	// Client certificate: the key-data is the secret; the cert-data gives us a
	// readable NotAfter, without ever calling kubectl.
	if u.ClientKeyData != "" || u.ClientCertData != "" {
		f := mkFinding("Kubernetes", "client certificate", path,
			SevMedium, who+" authenticates with a client certificate",
			"A client cert/key grants access until it expires and can't be rotated without "+
				"re-issuing. Keep the kubeconfig private; prefer short-lived, exec-based auth.")
		if exp, ok := certNotAfterB64(u.ClientCertData); ok {
			f.Expires = &exp
			if s := expirySeverity(exp, now); s > f.Severity {
				f.Severity = s
				f.SeverityLabel = s.String()
			}
			if exp.Before(now) {
				f.Title = who + " has an EXPIRED client certificate"
			}
		}
		out = append(out, f)
	}

	// Static token: a JWT with an exp is a short-lived (TokenRequest) token; a
	// JWT with no exp — or an opaque token — is a legacy, long-lived one.
	if tok := strings.TrimSpace(u.Token); tok != "" {
		if exp, ok := jwtExpiry(tok); ok {
			f := mkFinding("Kubernetes", "bearer token", path,
				SevLow, who+" carries a bearer token with an expiry", "Short-lived token.")
			f.Expires = &exp
			if exp.Before(now) {
				f.Severity, f.SeverityLabel = SevLow, SevLow.String()
				f.Title = who + " carries an expired bearer token"
			}
			out = append(out, f)
		} else {
			out = append(out, mkFinding("Kubernetes", "static bearer token", path,
				SevHigh, who+" carries a long-lived bearer token (no expiry)",
				"Legacy service-account tokens don't expire. Prefer bound, short-lived tokens "+
					"(TokenRequest) and keep the kubeconfig private (0600)."))
		}
	}
	return out
}

// kubeconfigPaths returns the kubeconfig files to inspect: every entry of
// $KUBECONFIG (colon-separated on Unix), else the default ~/.kube/config.
func kubeconfigPaths() []string {
	if kc := os.Getenv("KUBECONFIG"); kc != "" {
		var paths []string
		for _, p := range strings.Split(kc, string(os.PathListSeparator)) {
			if p = strings.TrimSpace(p); p != "" {
				paths = append(paths, p)
			}
		}
		if len(paths) > 0 {
			return paths
		}
	}
	return []string{"~/.kube/config"}
}

// filepathGlob expands a ~-prefixed glob and runs filepath.Glob on it.
func filepathGlob(pattern string) ([]string, error) {
	return filepath.Glob(expand(pattern))
}

// envPath returns the value of an env var when it names an existing file,
// else "". Used for GOOGLE_APPLICATION_CREDENTIALS and friends.
func envPath(name string) string {
	p := os.Getenv(name)
	if p == "" {
		return ""
	}
	p = expand(p) // a ~ in the env value should resolve like everywhere else
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}
