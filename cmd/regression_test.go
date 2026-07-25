package cmd

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/imagescan"
	"github.com/Mrg77/opsforge/internal/output"
)

// captureOut runs fn with stdout redirected and returns what it wrote.
func captureOut(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	done := make(chan string)
	go func() { b, _ := io.ReadAll(r); done <- string(b) }()
	fn()
	_ = w.Close()
	os.Stdout = orig
	return <-done
}

// TestScanJSONSchemaMatchesAudit guards two fixes at once: scan's CVE keys must
// be snake_case (id/severity/fixed_in) like `audit --json`, and with --diff the
// workstation_drift array must always be present (even empty) so a pipeline can
// tell "correlation ran, no drift" from "not requested".
func TestScanJSONSchemaMatchesAudit(t *testing.T) {
	output.JSON = true
	defer func() { output.JSON = false }()

	findings := []imagescan.ImageFinding{{
		Name: "x", Ecosystem: "npm", Version: "1.0.0", PURL: "pkg:npm/x@1.0.0",
		TopSeverity: "HIGH",
		Vulns:       []audit.Vuln{{ID: "CVE-1", Severity: audit.SevHigh, Summary: "s", FixedIn: "1.2.0"}},
	}}

	// With --diff and no drift: workstation_drift must be present as [].
	out := captureOut(t, func() {
		_ = emitScanJSON("img", "syft", nil, findings, nil, true)
	})
	var withDiff map[string]any
	if err := json.Unmarshal([]byte(out), &withDiff); err != nil {
		t.Fatalf("scan --diff --json not valid JSON: %v\n%s", err, out)
	}
	if _, ok := withDiff["workstation_drift"]; !ok {
		t.Error("--diff must always include workstation_drift (even empty)")
	}
	// CVE keys must be snake_case, severity a string.
	f := withDiff["findings"].([]any)[0].(map[string]any)
	v := f["vulnerabilities"].([]any)[0].(map[string]any)
	for _, k := range []string{"id", "severity", "fixed_in"} {
		if _, ok := v[k]; !ok {
			t.Errorf("scan CVE missing snake_case key %q (got %v)", k, v)
		}
	}
	if _, isStr := v["severity"].(string); !isStr {
		t.Errorf("scan CVE severity must be a string like audit, got %T", v["severity"])
	}

	// Without --diff: workstation_drift must be absent.
	out2 := captureOut(t, func() {
		_ = emitScanJSON("img", "syft", nil, findings, nil, false)
	})
	var noDiff map[string]any
	json.Unmarshal([]byte(out2), &noDiff)
	if _, ok := noDiff["workstation_drift"]; ok {
		t.Error("without --diff, workstation_drift should be absent")
	}
}

// TestListAllInheritsOutdatedFlag guards that `list all -u` works: the
// --outdated flag must be persistent so the `all` subcommand sees it.
func TestListAllInheritsOutdatedFlag(t *testing.T) {
	if listCmd.PersistentFlags().Lookup("outdated") == nil {
		t.Error("--outdated must be a persistent flag so `list all -u` works")
	}
}

// TestAnsibleMappedToCore guards the fix for bogus cross-version CVEs: ansible
// must map to ansible-core (matches `ansible --version`), not the "ansible"
// meta-package (collection numbering 3.x-12.x).
func TestAnsibleMappedToCore(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	tool, ok := cat.Tool("ansible")
	if !ok {
		t.Fatal("ansible not in catalog")
	}
	if tool.OSV == nil || tool.OSV.Name != "ansible-core" {
		t.Errorf("ansible OSV mapping = %v, want name=ansible-core", tool.OSV)
	}
}

// TestNoDebianEcosystemMappings guards that no tool maps to the Debian OSV
// ecosystem — those package versions don't match Homebrew/upstream installs
// and produced false-positive CVEs (jq, nmap).
func TestNoDebianEcosystemMappings(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, tl := range cat.Tools() {
		if tl.OSV != nil && tl.OSV.Ecosystem == "Debian" {
			t.Errorf("%s maps to the Debian ecosystem — remove it (false positives)", tl.Name)
		}
	}
}
