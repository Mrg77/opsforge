package secrets

import (
	"bufio"
	"strings"
	"testing"
)

func scan(t *testing.T, text string) []Finding {
	t.Helper()
	return ScanReader(bufio.NewScanner(strings.NewReader(text)), "test")
}

func TestDetectsKnownLeaks(t *testing.T) {
	cases := []struct {
		name, line, wantRule string
	}{
		{"aws key", "export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE", "aws-access-key"},
		{"github pat", "git clone https://ghp_abcdefghijklmnopqrstuvwxyz0123456789@github.com/x/y", "github-token"},
		{"gitlab pat", "glab auth login --token glpat-abcdefghij1234567890", "gitlab-token"},
		{"kubectl literal", "kubectl create secret generic db --from-literal=pw=s3cret", "kubectl-literal"},
		{"export secret", "export MY_API_KEY=abc123def", "env-export-secret"},
		{"docker login", "docker login -p hunter2 registry.io", "docker-password"},
		{"private key", "-----BEGIN RSA PRIVATE KEY-----", "private-key"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fs := scan(t, c.line)
			found := false
			for _, f := range fs {
				if f.Rule.ID == c.wantRule {
					found = true
				}
			}
			if !found {
				t.Errorf("rule %s not triggered by %q (got %d findings)", c.wantRule, c.line, len(fs))
			}
		})
	}
}

func TestIgnoresCleanLines(t *testing.T) {
	clean := `ls -la
git status
kubectl get pods -n prod
export PATH=$HOME/bin:$PATH
echo "no secrets here"`
	if fs := scan(t, clean); len(fs) != 0 {
		t.Errorf("clean input produced %d findings: %+v", len(fs), fs)
	}
}

func TestZshExtendedHistoryPrefixStripped(t *testing.T) {
	fs := scan(t, ": 1690000000:0;export DB_PASSWORD=oops")
	if len(fs) == 0 {
		t.Fatal("zsh extended-history line not scanned")
	}
	if fs[0].Line != 1 {
		t.Errorf("line = %d, want 1", fs[0].Line)
	}
}

func TestMaskNeverRevealsFullSecret(t *testing.T) {
	m := Mask("ghp_abcdefghijklmnopqrstuvwxyz0123456789")
	if strings.Contains(m, "ijklmnop") {
		t.Errorf("mask leaked the middle of the secret: %s", m)
	}
	if !strings.HasPrefix(m, "ghp_abcd") {
		t.Errorf("mask should keep a recognizable prefix: %s", m)
	}
	if Mask("short") != "*****" {
		t.Errorf("short values should be fully masked")
	}
}
