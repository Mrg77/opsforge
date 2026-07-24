package project

import (
	"path/filepath"
	"testing"

	"github.com/Mrg77/opsforge/internal/detect"
)

func statuses(m map[string]string) map[string]detect.Status {
	out := map[string]detect.Status{}
	for name, ver := range m {
		out[name] = detect.Status{Installed: true, Version: ver}
	}
	return out
}

func TestBuildLockPinsInstalledSortedNormalized(t *testing.T) {
	st := statuses(map[string]string{
		"helm": "v3.14.0",  // leading v should be normalized off
		"jq":   "jq-1.7.1", // prefix should be stripped
	})
	st["kubectl"] = detect.Status{Installed: false} // not installed → not pinned

	l := BuildLock([]string{"kubectl", "jq", "helm"}, st)

	if l.Version != 1 {
		t.Fatalf("lock version = %d, want 1", l.Version)
	}
	if len(l.Tools) != 2 {
		t.Fatalf("pinned %d tools, want 2 (kubectl not installed): %+v", len(l.Tools), l.Tools)
	}
	// Sorted by name: helm before jq.
	if l.Tools[0].Name != "helm" || l.Tools[1].Name != "jq" {
		t.Fatalf("tools not sorted by name: %+v", l.Tools)
	}
	if l.Tools[0].Version != "3.14.0" {
		t.Errorf("helm version = %q, want normalized 3.14.0", l.Tools[0].Version)
	}
	if l.Tools[1].Version != "1.7.1" {
		t.Errorf("jq version = %q, want normalized 1.7.1", l.Tools[1].Version)
	}
}

func TestWriteReadLockRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LockFileName)

	want := Lock{Version: 1, Tools: []LockedTool{
		{Name: "helm", Version: "3.14.0"},
		{Name: "jq", Version: "1.7.1"},
	}}
	if err := WriteLock(path, want); err != nil {
		t.Fatalf("WriteLock: %v", err)
	}
	got, ok, err := ReadLock(path)
	if err != nil {
		t.Fatalf("ReadLock: %v", err)
	}
	if !ok {
		t.Fatal("ReadLock ok=false for a file that exists")
	}
	if len(got.Tools) != 2 || got.Tools[0] != want.Tools[0] || got.Tools[1] != want.Tools[1] {
		t.Errorf("round trip mismatch: got %+v, want %+v", got, want)
	}
}

func TestReadLockMissingIsNotAnError(t *testing.T) {
	_, ok, err := ReadLock(filepath.Join(t.TempDir(), "nope.lock"))
	if err != nil {
		t.Fatalf("ReadLock on missing file errored: %v", err)
	}
	if ok {
		t.Error("ReadLock ok=true for a missing file")
	}
}

func TestCheckLockReportsDrift(t *testing.T) {
	lock := Lock{Version: 1, Tools: []LockedTool{
		{Name: "jq", Version: "1.7.1"},       // matches → no drift
		{Name: "helm", Version: "3.14.0"},    // version mismatch → drift
		{Name: "kubectl", Version: "1.29.0"}, // missing → drift
		{Name: "yq", Version: ""},            // unknown-at-lock-time → never drift
	}}
	st := statuses(map[string]string{
		"jq":   "1.7.1",
		"helm": "3.15.0",
		"yq":   "4.0.0",
	})
	// kubectl absent from st → not installed.

	drift := CheckLock(lock, st)
	if len(drift) != 2 {
		t.Fatalf("got %d drifts, want 2: %+v", len(drift), drift)
	}

	byName := map[string]LockDrift{}
	for _, d := range drift {
		byName[d.Name] = d
	}
	if d, ok := byName["helm"]; !ok || d.Expected != "3.14.0" || d.Got != "3.15.0" {
		t.Errorf("helm drift = %+v, want expected 3.14.0 got 3.15.0", d)
	}
	if d, ok := byName["kubectl"]; !ok || d.Got != "" {
		t.Errorf("kubectl drift = %+v, want got=\"\" (missing)", d)
	}
	if _, ok := byName["jq"]; ok {
		t.Error("jq flagged as drift but matches the lock")
	}
	if _, ok := byName["yq"]; ok {
		t.Error("yq flagged as drift but was pinned with an empty version")
	}
}

func TestCheckLockCleanWhenMachineMatches(t *testing.T) {
	lock := Lock{Version: 1, Tools: []LockedTool{{Name: "jq", Version: "1.7.1"}}}
	if drift := CheckLock(lock, statuses(map[string]string{"jq": "1.7.1"})); len(drift) != 0 {
		t.Errorf("expected no drift, got %+v", drift)
	}
}

func TestLockPathSitsBesideManifest(t *testing.T) {
	got := LockPath(filepath.Join("some", "dir", FileName))
	want := filepath.Join("some", "dir", LockFileName)
	if got != want {
		t.Errorf("LockPath = %q, want %q", got, want)
	}
}
