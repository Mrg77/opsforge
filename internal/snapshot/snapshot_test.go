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
	theme := ThemeState{Name: "dracula", Persisted: true}
	s := Capture(cat, statuses, nil, true, theme, "version: 1\nrules: []\n", "mise", time.Unix(0, 0))
	if len(s.Tools) != 2 || s.Tools[0] != "jq" || s.Tools[1] != "kubectl" {
		t.Errorf("Capture tools = %v, want sorted [jq kubectl]", s.Tools)
	}
	if !s.Shell.Enabled {
		t.Error("shell enabled state lost")
	}
	if s.Version != CurrentVersion {
		t.Errorf("version = %d", s.Version)
	}
	if s.Theme.Name != "dracula" || !s.Theme.Persisted {
		t.Errorf("theme not captured: %+v", s.Theme)
	}
	if s.Guards.YAML == "" {
		t.Error("guards YAML not captured")
	}
	if s.Versions.Manager != "mise" {
		t.Errorf("version manager = %q", s.Versions.Manager)
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

// TestMarshalLoadRoundTripV2 exercises every v2 field through
// Capture-shaped values, marshal, and load.
func TestMarshalLoadRoundTripV2(t *testing.T) {
	s := Snapshot{
		Version:  CurrentVersion,
		Tools:    []string{"jq", "kubectl"},
		Profiles: []catalog.Profile{{Name: "mine", Tools: []string{"jq"}}},
		Shell:    ShellState{Enabled: true},
		Theme:    ThemeState{Name: "dracula", Persisted: true},
		Guards:   GuardState{YAML: "version: 1\nrules: []\n"},
		Versions: VersionState{Manager: "mise"},
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
	if got.Theme.Name != "dracula" || !got.Theme.Persisted {
		t.Errorf("theme round trip lost: %+v", got.Theme)
	}
	if got.Guards.YAML != "version: 1\nrules: []\n" {
		t.Errorf("guards round trip lost: %q", got.Guards.YAML)
	}
	if got.Versions.Manager != "mise" {
		t.Errorf("versions round trip lost: %+v", got.Versions)
	}
}

// TestLoadV1Compat proves an old v1 file (no theme/guards/versions) still
// loads: the new fields are simply zero-valued, no error.
func TestLoadV1Compat(t *testing.T) {
	v1 := "version: 1\n" +
		"tools:\n  - jq\n  - kubectl\n" +
		"profiles:\n  - name: mine\n    tools: [jq]\n" +
		"shell:\n  enabled: true\n"
	path := filepath.Join(t.TempDir(), "v1.yaml")
	os.WriteFile(path, []byte(v1), 0o644)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("v1 snapshot failed to load: %v", err)
	}
	if got.Version != 1 {
		t.Errorf("version = %d, want 1", got.Version)
	}
	if len(got.Tools) != 2 || !got.Shell.Enabled || len(got.Profiles) != 1 {
		t.Errorf("v1 core fields lost: %+v", got)
	}
	// v2 fields must be zero-valued, not an error.
	if got.Theme.Name != "" || got.Theme.Persisted {
		t.Errorf("v1 theme should be zero: %+v", got.Theme)
	}
	if got.Guards.YAML != "" {
		t.Errorf("v1 guards should be empty: %q", got.Guards.YAML)
	}
	if got.Versions.Manager != "" {
		t.Errorf("v1 versions should be empty: %+v", got.Versions)
	}
}

func TestCheckDriftCompliant(t *testing.T) {
	cat := testCatalog()
	s := Snapshot{
		Version:  CurrentVersion,
		Tools:    []string{"jq", "kubectl"},
		Shell:    ShellState{Enabled: true},
		Theme:    ThemeState{Name: "dracula", Persisted: true},
		Guards:   GuardState{YAML: "version: 1\nrules: []\n"},
		Versions: VersionState{Manager: "mise"},
	}
	cur := CurrentState{
		Installed:      map[string]bool{"jq": true, "kubectl": true},
		ShellEnabled:   true,
		ThemeName:      "dracula",
		ThemePersisted: true,
		GuardsYAML:     "version: 1\nrules: []\n",
		VersionManager: "mise",
	}
	r := CheckDrift(s, cat, cur)
	if !r.Compliant {
		t.Errorf("expected compliant, got drift: %+v", r)
	}
	if len(r.MissingTools) != 0 || len(r.Drift) != 0 {
		t.Errorf("compliant report should be empty: %+v", r)
	}
}

func TestCheckDriftDetectsEverything(t *testing.T) {
	cat := testCatalog()
	s := Snapshot{
		Version:  CurrentVersion,
		Tools:    []string{"jq", "kubectl", "not-a-tool"},
		Shell:    ShellState{Enabled: true},
		Theme:    ThemeState{Name: "dracula", Persisted: true},
		Guards:   GuardState{YAML: "version: 1\nrules: []\n"},
		Versions: VersionState{Manager: "mise"},
	}
	cur := CurrentState{
		Installed:      map[string]bool{"jq": true, "kubectl": false},
		ShellEnabled:   false,
		ThemeName:      "forge",
		ThemePersisted: true,
		GuardsYAML:     "", // absent
		VersionManager: "asdf",
	}
	r := CheckDrift(s, cat, cur)
	if r.Compliant {
		t.Fatal("expected drift, got compliant")
	}
	if len(r.MissingTools) != 1 || r.MissingTools[0] != "kubectl" {
		t.Errorf("MissingTools = %v", r.MissingTools)
	}
	if len(r.UnknownTools) != 1 || r.UnknownTools[0] != "not-a-tool" {
		t.Errorf("UnknownTools = %v", r.UnknownTools)
	}
	kinds := map[DriftKind]Drift{}
	for _, d := range r.Drift {
		kinds[d.Kind] = d
	}
	for _, want := range []DriftKind{DriftShell, DriftTheme, DriftGuards, DriftVersion} {
		if _, ok := kinds[want]; !ok {
			t.Errorf("missing drift kind %q; got %+v", want, r.Drift)
		}
	}
	if kinds[DriftGuards].Actual != "absent" {
		t.Errorf("guards drift should report absent, got %q", kinds[DriftGuards].Actual)
	}
}

// TestCheckDriftIgnoresUnpinnedThemeAndDefaultGuards proves the baseline
// only enforces facts the source machine actually pinned: an auto theme and
// an empty guards field are not drift.
func TestCheckDriftIgnoresUnpinnedThemeAndDefaultGuards(t *testing.T) {
	cat := testCatalog()
	s := Snapshot{
		Version: CurrentVersion,
		Tools:   []string{"jq"},
		Theme:   ThemeState{Name: "forge", Persisted: false}, // auto-resolved
		Guards:  GuardState{YAML: ""},                        // default policy
	}
	cur := CurrentState{
		Installed:      map[string]bool{"jq": true},
		ThemeName:      "nord", // different, but snapshot didn't pin → no drift
		ThemePersisted: true,
		GuardsYAML:     "version: 1\nrules: []\n", // present, but snapshot empty → no drift
	}
	r := CheckDrift(s, cat, cur)
	if !r.Compliant {
		t.Errorf("expected compliant (nothing pinned), got: %+v", r.Drift)
	}
}

func TestCheckDriftGuardsWhitespaceInsensitive(t *testing.T) {
	cat := testCatalog()
	s := Snapshot{
		Version: CurrentVersion,
		Guards:  GuardState{YAML: "version: 1\nrules: []\n"},
	}
	cur := CurrentState{
		Installed:  map[string]bool{},
		GuardsYAML: "\nversion: 1\nrules: []\r\n\n", // same content, only surrounding whitespace/CRLF differ
	}
	r := CheckDrift(s, cat, cur)
	for _, d := range r.Drift {
		if d.Kind == DriftGuards {
			t.Errorf("cosmetic guard whitespace should not be drift: %+v", d)
		}
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
