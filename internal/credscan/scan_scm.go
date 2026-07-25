package credscan

import (
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

// scanSCM inspects git and SCM-CLI credential stores: ~/.netrc, the git
// credential-store file, and the gh/glab config files. These all keep tokens
// in clear text by design, so their mere presence (with a real value) is the
// finding. Purely file-based.
func scanSCM() []Finding {
	var out []Finding

	// ~/.netrc — machine/login/password lines in clear text. Its .gpg variant
	// is encrypted and is NOT a clear-text risk.
	if b, ok := readFile("~/.netrc"); ok {
		out = append(out, permFinding("Git", ".netrc", "~/.netrc")...)
		if netrcHasPassword(b) {
			out = append(out, mkFinding("Git", "clear-text password", "~/.netrc",
				SevMedium, "~/.netrc stores a login password/token in clear text",
				"Prefer a credential helper backed by your OS keychain. Keep the file 0600."))
		}
	}

	// git credential-store: full URLs with user:token@host, plaintext by design.
	for _, p := range []string{"~/.git-credentials", "~/.config/git/credentials"} {
		if b, ok := readFile(p); ok {
			out = append(out, permFinding("Git", "credential store", p)...)
			if gitCredentialsHasSecret(b) {
				out = append(out, mkFinding("Git", "stored credential", p,
					SevMedium, "git stores credentials in clear text (helper `store`)",
					"The `store` helper writes tokens unencrypted. Switch to `osxkeychain`, "+
						"`libsecret`, or `manager` (`git config --global credential.helper ...`)."))
			}
		}
	}

	// GitHub CLI — oauth_token per host. Absent token often means the OS keyring
	// holds it, which is GOOD hygiene, not "nothing found".
	if b, ok := readFile("~/.config/gh/hosts.yml"); ok {
		out = append(out, permFinding("GitHub", "gh hosts", "~/.config/gh/hosts.yml")...)
		out = append(out, ghHostsFindings(b)...)
	}

	// GitLab CLI — hosts.<host>.token.
	if b, ok := readFile("~/.config/glab-cli/config.yml"); ok {
		out = append(out, permFinding("GitLab", "glab config", "~/.config/glab-cli/config.yml")...)
		if glabHasToken(b) {
			out = append(out, mkFinding("GitLab", "personal access token", "~/.config/glab-cli/config.yml",
				SevMedium, "a GitLab token is stored in clear text",
				"Consider `glab auth login --use-keyring` to keep the token in your OS keychain."))
		}
	}
	return out
}

func netrcHasPassword(b []byte) bool {
	fields := strings.Fields(string(b))
	for i := 0; i+1 < len(fields); i++ {
		if fields[i] == "password" && !isPlaceholder(fields[i+1]) {
			return true
		}
	}
	return false
}

func gitCredentialsHasSecret(b []byte) bool {
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if u, err := url.Parse(line); err == nil {
			if pw, ok := u.User.Password(); ok && !isPlaceholder(pw) {
				return true
			}
		}
	}
	return false
}

// ghHostsFindings reports a clear-text token per host, and treats a host with
// no oauth_token as keyring-backed (good hygiene) rather than a miss.
func ghHostsFindings(b []byte) []Finding {
	var hosts map[string]struct {
		OAuthToken string `yaml:"oauth_token"`
	}
	if yaml.Unmarshal(b, &hosts) != nil {
		return nil
	}
	var out []Finding
	for host, h := range hosts {
		if strings.TrimSpace(h.OAuthToken) == "" {
			out = append(out, mkFinding("GitHub", "keyring-backed token", "~/.config/gh/hosts.yml",
				SevOK, host+": token is not in the file (OS keyring)", "Good hygiene."))
			continue
		}
		out = append(out, mkFinding("GitHub", "OAuth token", "~/.config/gh/hosts.yml",
			SevMedium, host+": an OAuth token is stored in clear text",
			"Store it in your OS keychain instead: `gh auth login` with keyring support, "+
				"or `gh auth logout` when unused. Keep the file 0600."))
	}
	return out
}

func glabHasToken(b []byte) bool {
	var cfg struct {
		Hosts map[string]struct {
			Token string `yaml:"token"`
		} `yaml:"hosts"`
	}
	if yaml.Unmarshal(b, &cfg) != nil {
		return false
	}
	for _, h := range cfg.Hosts {
		if strings.TrimSpace(h.Token) != "" {
			return true
		}
	}
	return false
}

// isPlaceholder recognizes obvious non-secrets (empty, env interpolation, or a
// stock placeholder) so we don't flag a template as a leak.
func isPlaceholder(v string) bool {
	v = strings.TrimSpace(strings.Trim(v, `"'`))
	if v == "" {
		return true
	}
	if strings.HasPrefix(v, "${") || strings.HasPrefix(v, "$(") || strings.HasPrefix(v, "%") {
		return true
	}
	switch strings.ToLower(v) {
	case "null", "~", "changeme", "your-token-here", "xxx", "todo", "__token__", "oauth2":
		return true
	}
	return false
}
