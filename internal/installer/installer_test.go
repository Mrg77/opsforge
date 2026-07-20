package installer

import "testing"

func TestIsLinkConflict(t *testing.T) {
	// The real brew output that broke a docker upgrade for a user.
	dockerConflict := `To list all files that would be deleted:
  brew link --overwrite docker --dry-run

Possible conflicting files are:
/opt/homebrew/etc/bash_completion.d/docker -> /opt/homebrew/Cellar/docker-completion/28.3.3/etc/bash_completion.d/docker`

	cases := []struct {
		name string
		out  string
		want bool
	}{
		{"docker link --overwrite hint", dockerConflict, true},
		{"conflicting files phrase", "Error: conflicting files installed", true},
		{"already linked", "Warning: foo is already linked", true},
		{"normal success", "==> Upgrading foo\n==> Pouring foo.bottle", false},
		{"unrelated error", "Error: Download failed: connection reset", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isLinkConflict(c.out); got != c.want {
				t.Errorf("isLinkConflict(%q) = %v, want %v", c.name, got, c.want)
			}
		})
	}
}

func TestLeafFormula(t *testing.T) {
	cases := map[string]string{
		"hashicorp/tap/vault": "vault",
		"fluxcd/tap/flux":     "flux",
		"xo/xo/usql":          "usql",
		"jq":                  "jq",
	}
	for in, want := range cases {
		if got := leafFormula(in); got != want {
			t.Errorf("leafFormula(%q) = %q, want %q", in, got, want)
		}
	}
}
