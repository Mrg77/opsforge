package detect

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Mrg77/opsforge/internal/catalog"
)

// Version probes must run with a neutralized KUBECONFIG: on machines
// where kubectl is a cloud-SDK dispatcher wired to an OIDC exec plugin,
// a probe with the real kubeconfig pops a browser login (see README).
func TestSafeProbeEnvNeutralizesKubeconfig(t *testing.T) {
	t.Setenv("KUBECONFIG", "/home/user/.kube/config")
	env := SafeProbeEnv()
	last := ""
	for _, kv := range env {
		if strings.HasPrefix(kv, "KUBECONFIG=") {
			last = kv
		}
	}
	if last != "KUBECONFIG="+os.DevNull {
		t.Errorf("effective KUBECONFIG entry = %q, want %q", last, "KUBECONFIG="+os.DevNull)
	}
}

func TestFirstVersionLine(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"simple", "jq-1.7.1\n", "jq-1.7.1"},
		{"skips banner", "  ____\n |logo|\n\nv0.32.5\n", "v0.32.5"},
		{"no digits", "no version here\n", ""},
		{"empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := FirstVersionLine(c.in); got != c.want {
				t.Errorf("FirstVersionLine(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestToolDetectsStubInPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub not portable to windows")
	}
	dir := t.TempDir()
	stub := filepath.Join(dir, "faketool")
	script := "#!/bin/sh\necho 'faketool 9.9.9'\n"
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	s := Tool(catalog.Tool{Name: "faketool", Bin: "faketool"})
	if !s.Installed {
		t.Fatal("stub tool not detected as installed")
	}
	if s.Version != "faketool 9.9.9" {
		t.Errorf("version = %q, want %q", s.Version, "faketool 9.9.9")
	}

	missing := Tool(catalog.Tool{Name: "nope", Bin: "definitely-not-a-real-binary"})
	if missing.Installed {
		t.Error("missing tool reported as installed")
	}
}
