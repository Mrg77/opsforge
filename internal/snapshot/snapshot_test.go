package snapshot

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
)

func testCatalog() *catalog.Catalog {
	return &catalog.Catalog{Categories: []catalog.Category{{
		Name: "Test",
		Tools: []catalog.Tool{
			{Name: "jq", Bin: "jq", Brew: "jq"},
			{Name: "kubectl", Bin: "kubectl", Brew: "kubernetes-cli"},
			{Name: "helm", Bin: "helm", Brew: "helm"},
		},
	}}}
}

func TestCaptureOnlyInstalledSorted(t *testing.T) {
	cat := testCatalog()
	statuses := map[string]detect.Status{
		"kubectl": {Installed: true},
		"jq":      {Installed: true},
		"helm":    {}, // not installed
	}
	s := Capture(cat, statuses, nil, true, time.Unix(0, 0))
	if len(s.Tools) != 2 || s.Tools[0] != "jq" || s.Tools[1] != "kubectl" {
		t.Errorf("Capture tools = %v, want sorted [jq kubectl]", s.Tools)
	}
	if !s.Shell.Enabled {
		t.Error("shell enabled state lost")
	}
	if s.Version != CurrentVersion {
		t.Errorf("version = %d", s.Version)
	}
}

func TestMarshalLoadRoundTrip(t *testing.T) {
	s := Snapshot{
		Version:  CurrentVersion,
		Tools:    []string{"jq", "kubectl"},
		Profiles: []catalog.Profile{{Name: "mine", Tools: []string{"jq"}}},
		Shell:    ShellState{Enabled: true},
	}
	data, err := s.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "snap.yaml")
	os.WriteFile(path, data, 0o644)

	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Tools) != 2 || len(got.Profiles) != 1 || !got.Shell.Enabled {
		t.Errorf("round trip lost data: %+v", got)
	}
}

func TestLoadRejectsGarbageAndFutureVersions(t *testing.T) {
	dir := t.TempDir()

	garbage := filepath.Join(dir, "garbage.yaml")
	os.WriteFile(garbage, []byte("hello: world\n"), 0o644)
	if _, err := Load(garbage); err == nil {
		t.Error("Load accepted a non-snapshot file")
	}

	future := filepath.Join(dir, "future.yaml")
	os.WriteFile(future, []byte("version: 99\ntools: []\n"), 0o644)
	if _, err := Load(future); err == nil {
		t.Error("Load accepted a future snapshot version")
	}
}

func TestLoadFromURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("version: 1\ntools: [jq]\nshell:\n  enabled: false\n"))
	}))
	defer srv.Close()
	s, err := Load(srv.URL + "/snap.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Tools) != 1 || s.Tools[0] != "jq" {
		t.Errorf("URL load = %+v", s)
	}
}

func TestBuildPlan(t *testing.T) {
	cat := testCatalog()
	statuses := map[string]detect.Status{
		"jq": {Installed: true},
	}
	s := Snapshot{Version: 1, Tools: []string{"jq", "kubectl", "not-a-tool"}}
	p := BuildPlan(s, cat, statuses)
	if len(p.Present) != 1 || p.Present[0] != "jq" {
		t.Errorf("Present = %v", p.Present)
	}
	if len(p.Install) != 1 || p.Install[0] != "kubectl" {
		t.Errorf("Install = %v", p.Install)
	}
	if len(p.Unknown) != 1 || p.Unknown[0] != "not-a-tool" {
		t.Errorf("Unknown = %v", p.Unknown)
	}
}
