package credscan

import (
	"encoding/json"
	"strings"
)

// scanInfra covers infra/registry credential stores: Docker, Terraform Cloud,
// and the language package registries (npm, PyPI, Cargo, RubyGems). The common
// trap the report flags: base64 is NOT encryption — a Docker `auth` or npm
// `_auth` is clear text. Purely file-based.
func scanInfra() []Finding {
	var out []Finding
	out = append(out, scanDocker()...)
	out = append(out, scanTerraform()...)
	out = append(out, scanPackageRegistries()...)
	return out
}

// scanDocker reads ~/.docker/config.json. An `auths.<reg>.auth` is base64 of
// user:password — clear text. A registry delegated to credsStore/credHelpers
// keeps NO secret in the file (good hygiene) — don't flag it.
func scanDocker() []Finding {
	b, ok := readFile("~/.docker/config.json")
	if !ok {
		return nil
	}
	out := permFinding("Docker", "docker config", "~/.docker/config.json")

	var cfg struct {
		Auths       map[string]struct{ Auth string } `json:"auths"`
		CredsStore  string                           `json:"credsStore"`
		CredHelpers map[string]string                `json:"credHelpers"`
	}
	if json.Unmarshal(b, &cfg) != nil {
		return out
	}
	for reg, a := range cfg.Auths {
		if strings.TrimSpace(a.Auth) == "" {
			continue // handled by a helper, or empty
		}
		// A registry covered by a helper keeps auth around redundantly.
		delegated := cfg.CredsStore != ""
		if _, ok := cfg.CredHelpers[reg]; ok {
			delegated = true
		}
		detail := "Docker `auth` is base64(user:password), not encryption. Use a credential " +
			"store (`credsStore`) so the secret lives in your OS keychain."
		if delegated {
			detail = reg + " also has a credential helper, yet a base64 login lingers here — remove it."
		}
		out = append(out, mkFinding("Docker", "registry login", "~/.docker/config.json",
			SevMedium, reg+": a base64 registry login is stored in clear text", detail))
	}
	return out
}

// scanTerraform reads Terraform Cloud/Enterprise credentials. The token is
// long-lived with no local expiry.
func scanTerraform() []Finding {
	p := "~/.terraform.d/credentials.tfrc.json"
	b, ok := readFile(p)
	if !ok {
		return nil
	}
	out := permFinding("Terraform", "TFC credentials", p)
	var v struct {
		Credentials map[string]struct{ Token string } `json:"credentials"`
	}
	if json.Unmarshal(b, &v) == nil {
		for host, c := range v.Credentials {
			if strings.TrimSpace(c.Token) != "" {
				out = append(out, mkFinding("Terraform", "API token", p,
					SevMedium, host+": a long-lived Terraform token is stored in clear text",
					"Revoke it in Terraform Cloud when unused; keep the file 0600."))
			}
		}
	}
	return out
}

// scanPackageRegistries covers npm, PyPI, Cargo, RubyGems. Each keeps a token
// or a base64 auth in a config file. Env interpolation (${VAR}) is not a
// hard-coded secret; a keyring-backed absence is fine.
func scanPackageRegistries() []Finding {
	var out []Finding

	// npm ~/.npmrc — _authToken / _auth / _password lines (base64 for the last
	// two, still clear text).
	if b, ok := readFile("~/.npmrc"); ok {
		out = append(out, permFinding("npm", ".npmrc", "~/.npmrc")...)
		if npmrcHasSecret(b) {
			out = append(out, mkFinding("npm", "registry token", "~/.npmrc",
				SevMedium, "a registry token/auth is hard-coded in ~/.npmrc",
				"Prefer `${NPM_TOKEN}` env interpolation over an inline token, and keep the file 0600."))
		}
	}

	// PyPI ~/.pypirc — [server] password (token prefixed pypi-).
	if b, ok := readFile("~/.pypirc"); ok {
		out = append(out, permFinding("PyPI", ".pypirc", "~/.pypirc")...)
		ini := parseINI(b)
		for section, kv := range ini {
			if section == "" {
				continue
			}
			if pw := kv["password"]; pw != "" && !isPlaceholder(pw) {
				out = append(out, mkFinding("PyPI", "upload token", "~/.pypirc",
					SevMedium, "["+section+"] stores an upload token/password in clear text",
					"Scope the token narrowly and keep the file 0600."))
			}
		}
	}

	// Cargo ~/.cargo/credentials.toml — token under [registry]/[registries.*].
	for _, p := range []string{"~/.cargo/credentials.toml", "~/.cargo/credentials"} {
		if b, ok := readFile(p); ok {
			out = append(out, permFinding("Cargo", "cargo credentials", p)...)
			if tomlHasToken(b) {
				out = append(out, mkFinding("Cargo", "registry token", p,
					SevMedium, "a crates.io/registry token is stored in clear text",
					"Keep the file 0600; revoke unused tokens on crates.io."))
			}
		}
	}

	// RubyGems ~/.gem/credentials — :rubygems_api_key.
	if b, ok := readFile("~/.gem/credentials"); ok {
		out = append(out, permFinding("RubyGems", "gem credentials", "~/.gem/credentials")...)
		if strings.Contains(string(b), "rubygems_api_key") {
			out = append(out, mkFinding("RubyGems", "API key", "~/.gem/credentials",
				SevMedium, "a RubyGems API key is stored in clear text",
				"Keep the file 0600; scope and rotate the key."))
		}
	}
	return out
}

// npmrcHasSecret reports an inline (non env-interpolated) _authToken/_auth/
// _password value.
func npmrcHasSecret(b []byte) bool {
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		for _, key := range []string{"_authToken", "_auth", "_password"} {
			if i := strings.Index(line, key+"="); i >= 0 {
				v := strings.TrimSpace(line[i+len(key)+1:])
				if v != "" && !isPlaceholder(v) {
					return true
				}
			}
		}
	}
	return false
}

// tomlHasToken reports a non-empty `token = "..."` (dependency-free — the file
// is tiny and we never write it).
func tomlHasToken(b []byte) bool {
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "token") {
			if _, v, ok := strings.Cut(line, "="); ok {
				v = strings.TrimSpace(strings.Trim(strings.TrimSpace(v), `"'`))
				if v != "" && !isPlaceholder(v) {
					return true
				}
			}
		}
	}
	return false
}
