package history

import (
	"os"
	"path/filepath"
	"testing"
)

func writeHist(t *testing.T, lines string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "hist")
	if err := os.WriteFile(p, []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestQueryFiltersByFamilyAndOrdersByRecency(t *testing.T) {
	p := writeHist(t, `git status
kubectl get pods
terraform plan
kubectl logs -f api
`)
	got, err := Query(p, []string{"kubectl", "helm"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 kube entries, got %d: %+v", len(got), got)
	}
	// Most recent first.
	if got[0].Command != "kubectl logs -f api" {
		t.Errorf("recency order wrong, first = %q", got[0].Command)
	}
}

func TestQueryDedupesWithCount(t *testing.T) {
	p := writeHist(t, "kubectl get pods\ngit status\nkubectl get pods\n")
	got, err := Query(p, []string{"kubectl"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Count != 2 {
		t.Fatalf("want 1 entry ×2, got %+v", got)
	}
}

func TestQueryHandlesExtendedHistory(t *testing.T) {
	p := writeHist(t, ": 1700000001:0;kubectl get ns\n: 1700000002:5;helm list\n")
	got, err := Query(p, []string{"kubectl", "helm"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("extended-history parse failed: %+v", got)
	}
	if got[0].Command != "helm list" {
		t.Errorf("timestamp prefix not stripped or order wrong: %q", got[0].Command)
	}
}

func TestFirstWordSkipsEnvAndSudo(t *testing.T) {
	cases := map[string]string{
		"AWS_PROFILE=prod aws s3 ls": "aws",
		"sudo kubectl top nodes":     "kubectl",
		"FOO=1 BAR=2 terraform plan": "terraform",
		"git status":                 "git",
		"":                           "",
	}
	for in, want := range cases {
		if got := firstWord(in); got != want {
			t.Errorf("firstWord(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestQueryLimit(t *testing.T) {
	p := writeHist(t, "kubectl a\nkubectl b\nkubectl c\n")
	got, err := Query(p, []string{"kubectl"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("limit not applied: %d", len(got))
	}
}
