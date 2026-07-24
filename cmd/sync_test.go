package cmd

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/project"
)

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what
// it wrote. syncReport prints through both fmt (human) and output.Emit (JSON),
// both of which target os.Stdout, so this captures either path.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	fn()
	_ = w.Close()
	os.Stdout = orig
	return <-done
}

func status(installed bool, version string) detect.Status {
	return detect.Status{Installed: installed, Version: version}
}

// writeLockFile drops an opsforge.lock next to a manifest path in dir.
func writeLockFile(t *testing.T, dir string, l project.Lock) string {
	t.Helper()
	manifest := filepath.Join(dir, project.FileName)
	if err := project.WriteLock(project.LockPath(manifest), l); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func TestSyncReportCompliantNoDrift(t *testing.T) {
	dir := t.TempDir()
	// Lock pins jq@1.7.1; the machine has exactly that → compliant.
	manifest := writeLockFile(t, dir, project.Lock{Version: 1, Tools: []project.LockedTool{
		{Name: "jq", Version: "1.7.1"},
	}})
	statuses := map[string]detect.Status{"jq": status(true, "1.7.1")}
	plan := project.Plan{Present: []string{"jq"}}
	proj := &project.Project{} // FailOn == "" → no CVE gate, cat unused

	var reportErr error
	out := captureStdout(t, func() {
		reportErr = syncReport(proj, nil, plan, manifest, statuses)
	})
	if reportErr != nil {
		t.Errorf("compliant machine should not error, got %v", reportErr)
	}
	if !strings.Contains(out, "Compliant") {
		t.Errorf("expected a Compliant message, got:\n%s", out)
	}
}

func TestSyncReportVersionDriftHuman(t *testing.T) {
	dir := t.TempDir()
	manifest := writeLockFile(t, dir, project.Lock{Version: 1, Tools: []project.LockedTool{
		{Name: "helm", Version: "3.14.0"},
	}})
	// Machine has a different helm → version drift.
	statuses := map[string]detect.Status{"helm": status(true, "3.15.1")}
	plan := project.Plan{Present: []string{"helm"}}
	proj := &project.Project{}

	var reportErr error
	out := captureStdout(t, func() {
		reportErr = syncReport(proj, nil, plan, manifest, statuses)
	})
	if reportErr == nil {
		t.Error("version drift should return a non-nil error (non-zero exit)")
	}
	if !strings.Contains(out, "helm") || !strings.Contains(out, "3.14.0") || !strings.Contains(out, "3.15.1") {
		t.Errorf("drift line should name the tool and both versions, got:\n%s", out)
	}
}

func TestSyncReportVersionDriftJSON(t *testing.T) {
	dir := t.TempDir()
	manifest := writeLockFile(t, dir, project.Lock{Version: 1, Tools: []project.LockedTool{
		{Name: "helm", Version: "3.14.0"},
	}})
	statuses := map[string]detect.Status{"helm": status(true, "3.15.1")}
	plan := project.Plan{Present: []string{"helm"}}
	proj := &project.Project{}

	output.JSON = true
	defer func() { output.JSON = false }()

	var reportErr error
	out := captureStdout(t, func() {
		reportErr = syncReport(proj, nil, plan, manifest, statuses)
	})
	if reportErr == nil {
		t.Error("version drift should return a non-nil error even in JSON mode")
	}

	var got struct {
		Compliant    bool                `json:"compliant"`
		VersionDrift []project.LockDrift `json:"version_drift"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output was not valid JSON: %v\n%s", err, out)
	}
	if got.Compliant {
		t.Error("compliant should be false on drift")
	}
	if len(got.VersionDrift) != 1 || got.VersionDrift[0].Name != "helm" ||
		got.VersionDrift[0].Expected != "3.14.0" || got.VersionDrift[0].Got != "3.15.1" {
		t.Errorf("version_drift payload wrong: %+v", got.VersionDrift)
	}
}

func TestSyncReportCompliantJSON(t *testing.T) {
	dir := t.TempDir()
	manifest := writeLockFile(t, dir, project.Lock{Version: 1, Tools: []project.LockedTool{
		{Name: "jq", Version: "1.7.1"},
	}})
	statuses := map[string]detect.Status{"jq": status(true, "1.7.1")}
	plan := project.Plan{Present: []string{"jq"}, Unknown: []string{"acme-cli"}}
	proj := &project.Project{}

	output.JSON = true
	defer func() { output.JSON = false }()

	var reportErr error
	out := captureStdout(t, func() {
		reportErr = syncReport(proj, nil, plan, manifest, statuses)
	})
	if reportErr != nil {
		t.Errorf("compliant machine should not error, got %v", reportErr)
	}
	var got struct {
		Compliant bool     `json:"compliant"`
		Unknown   []string `json:"unknown"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output was not valid JSON: %v\n%s", err, out)
	}
	if !got.Compliant {
		t.Error("expected compliant=true")
	}
	// Unknown (not-in-catalog) tools are reported but never fail the check.
	if len(got.Unknown) != 1 || got.Unknown[0] != "acme-cli" {
		t.Errorf("unknown tools not surfaced: %+v", got.Unknown)
	}
}

func TestSyncReportUnknownDoesNotFail(t *testing.T) {
	// A tool that isn't in the catalog is a warning, not a drift: with an
	// empty lock and nothing missing, --check stays compliant.
	dir := t.TempDir()
	manifest := writeLockFile(t, dir, project.Lock{Version: 1})
	plan := project.Plan{Unknown: []string{"acme-cli"}}
	proj := &project.Project{}

	var reportErr error
	out := captureStdout(t, func() {
		reportErr = syncReport(proj, nil, plan, manifest, map[string]detect.Status{})
	})
	if reportErr != nil {
		t.Errorf("an unknown tool must not fail the check, got %v", reportErr)
	}
	if !strings.Contains(out, "acme-cli") {
		t.Errorf("unknown tool should still be listed, got:\n%s", out)
	}
}

func TestSyncReportMissingToolNoLock(t *testing.T) {
	// No lockfile at all: --check must still report a missing required tool
	// (plan.Install non-empty) and behave exactly as before the lock feature.
	dir := t.TempDir()
	manifest := filepath.Join(dir, project.FileName) // never written
	plan := project.Plan{Install: []string{"kubectl"}}
	proj := &project.Project{}

	var reportErr error
	out := captureStdout(t, func() {
		reportErr = syncReport(proj, nil, plan, manifest, map[string]detect.Status{})
	})
	if reportErr == nil {
		t.Error("a missing required tool should return a non-nil error")
	}
	if !strings.Contains(out, "kubectl") || !strings.Contains(out, "missing") {
		t.Errorf("expected kubectl reported missing, got:\n%s", out)
	}
}
