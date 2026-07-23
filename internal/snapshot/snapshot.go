// Package snapshot implements workstation-as-code: capture everything
// opsforge manages (installed tools, user profiles, shell environment
// state, the active theme, the guard policy, the detected version
// manager) into one shareable YAML file, and compute the plan to rebuild
// — or verify — that exact workstation on another machine.
//
// The file is designed to live in a dotfiles repo or a gist:
//
//	opsforge snapshot -o my-setup.yaml          # on the old machine
//	opsforge apply https://.../my-setup.yaml    # rebuild on the new one
//	opsforge apply --check my-setup.yaml        # verify a machine in CI
//
// Format history:
//
//	v1 — tools, profiles, shell.enabled
//	v2 — adds theme, guards (raw guards.yaml), versions (manager name).
//	     v1 files still load: the new fields are simply zero-valued.
package snapshot

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
)

// CurrentVersion is the snapshot format version this build writes. v2 adds
// theme/guards/versions on top of v1 (tools/profiles/shell).
const CurrentVersion = 2

// Snapshot is the portable description of an opsforge-managed workstation.
type Snapshot struct {
	Version int    `yaml:"version"`
	Created string `yaml:"created,omitempty"`
	// Tools are catalog tool names that were installed on the machine.
	Tools []string `yaml:"tools"`
	// Profiles are the user's saved custom profiles, embedded so the new
	// machine gets them without a separate file.
	Profiles []catalog.Profile `yaml:"profiles,omitempty"`
	// Shell records whether the DevOps shell environment was enabled.
	Shell ShellState `yaml:"shell"`
	// Theme is the persisted opsforge theme (v2). Empty means "auto / not
	// pinned" — apply leaves the theme untouched and --check ignores it.
	Theme ThemeState `yaml:"theme,omitempty"`
	// Guards embeds the user's guard policy so security rules travel with
	// the workstation (v2). Empty means the user relied on the built-in
	// default policy.
	Guards GuardState `yaml:"guards,omitempty"`
	// Versions records the detected tool-version manager (v2). It documents
	// the baseline; opsforge delegates actual pinning to mise/asdf.
	Versions VersionState `yaml:"versions,omitempty"`
}

// ShellState captures the shell-environment side of the workstation.
type ShellState struct {
	Enabled bool `yaml:"enabled"`
}

// ThemeState captures the persisted UI theme. Name is the theme name (e.g.
// "dracula"); Persisted distinguishes "the user explicitly chose this" from
// "auto-resolved", so apply only writes a theme the user actually pinned.
type ThemeState struct {
	Name      string `yaml:"name,omitempty"`
	Persisted bool   `yaml:"persisted,omitempty"`
}

// GuardState embeds the user's guards.yaml verbatim. Storing the raw YAML
// (rather than a re-serialized struct) keeps comments and ordering intact,
// and means apply reproduces exactly the file the user wrote.
type GuardState struct {
	// YAML is the literal content of ~/.config/opsforge/guards.yaml, or ""
	// when the user has no custom policy (built-in defaults were in use).
	YAML string `yaml:"yaml,omitempty"`
}

// VersionState records the version manager present on the source machine.
// Listing individual pins is intentionally out of scope: pins live in
// per-project .mise.toml / .tool-versions files that belong to the repos,
// not to the workstation, so the snapshot captures the manager only.
type VersionState struct {
	// Manager is the detected version manager ("mise", "asdf", or "").
	Manager string `yaml:"manager,omitempty"`
}

// Capture builds a Snapshot from the current machine state. Pure with
// respect to its inputs so it is fully testable — every environment read
// happens in the caller (cmd/snapshot.go), never here.
func Capture(cat *catalog.Catalog, statuses map[string]detect.Status,
	userProfiles []catalog.Profile, shellEnabled bool,
	theme ThemeState, guardsYAML string, versionManager string,
	created time.Time) Snapshot {

	var tools []string
	for _, t := range cat.Tools() {
		if statuses[t.Name].Installed {
			tools = append(tools, t.Name)
		}
	}
	sort.Strings(tools)
	return Snapshot{
		Version:  CurrentVersion,
		Created:  created.UTC().Format(time.RFC3339),
		Tools:    tools,
		Profiles: userProfiles,
		Shell:    ShellState{Enabled: shellEnabled},
		Theme:    theme,
		Guards:   GuardState{YAML: guardsYAML},
		Versions: VersionState{Manager: versionManager},
	}
}

const fileHeader = `# opsforge workstation snapshot — rebuild or verify this exact setup with:
#   opsforge apply <this-file-or-its-url>          # rebuild it here
#   opsforge apply --check <this-file-or-its-url>  # verify a machine (CI baseline)
`

// Marshal renders the snapshot as YAML with an explanatory header.
func (s Snapshot) Marshal() ([]byte, error) {
	body, err := yaml.Marshal(s)
	if err != nil {
		return nil, err
	}
	return append([]byte(fileHeader), body...), nil
}

// Load reads a snapshot from a local path or an http(s) URL and validates
// it. Older (v1) files load cleanly: fields introduced in v2 stay
// zero-valued rather than erroring, so a v1 snapshot keeps working.
func Load(src string) (Snapshot, error) {
	var data []byte
	var err error
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		data, err = fetch(src)
	} else {
		data, err = os.ReadFile(src)
	}
	if err != nil {
		return Snapshot{}, err
	}
	var s Snapshot
	if err := yaml.Unmarshal(data, &s); err != nil {
		return Snapshot{}, fmt.Errorf("parsing snapshot: %w", err)
	}
	if s.Version == 0 {
		return Snapshot{}, fmt.Errorf("not an opsforge snapshot (missing version field)")
	}
	if s.Version > CurrentVersion {
		return Snapshot{}, fmt.Errorf(
			"snapshot version %d is newer than this opsforge understands (%d) — upgrade opsforge",
			s.Version, CurrentVersion)
	}
	return s, nil
}

func fetch(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching snapshot: %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // snapshots are small; 1MB cap
}

// Plan is what applying a snapshot would do on this machine.
type Plan struct {
	Install []string // catalog tools to install
	Present []string // already installed, nothing to do
	Unknown []string // in the snapshot but not in this build's catalog
}

// BuildPlan diffs a snapshot against the current machine.
func BuildPlan(s Snapshot, cat *catalog.Catalog, statuses map[string]detect.Status) Plan {
	var p Plan
	for _, name := range s.Tools {
		if _, ok := cat.Tool(name); !ok {
			p.Unknown = append(p.Unknown, name)
			continue
		}
		if statuses[name].Installed {
			p.Present = append(p.Present, name)
		} else {
			p.Install = append(p.Install, name)
		}
	}
	return p
}

// CurrentState is the live machine state that `apply --check` compares the
// snapshot against. Like Capture's inputs it is read by the caller and
// passed in, so CheckDrift stays pure and testable.
type CurrentState struct {
	// Installed reports, per catalog tool name, whether it is present.
	Installed map[string]bool
	// ShellEnabled mirrors shellcfg.InstalledInZshrc().
	ShellEnabled bool
	// ThemeName is the persisted theme name ("" when not pinned).
	ThemeName string
	// ThemePersisted mirrors ui.ThemePersisted().
	ThemePersisted bool
	// GuardsYAML is the raw content of the machine's guards.yaml ("" when
	// absent).
	GuardsYAML string
	// VersionManager is the detected manager ("mise"/"asdf"/"").
	VersionManager string
}

// DriftKind categorizes a single deviation, so machine consumers can filter.
type DriftKind string

const (
	DriftShell   DriftKind = "shell"
	DriftTheme   DriftKind = "theme"
	DriftGuards  DriftKind = "guards"
	DriftVersion DriftKind = "version_manager"
)

// Drift is one non-tool deviation from the baseline (tool drift is reported
// separately as MissingTools, which is the common CI failure).
type Drift struct {
	Kind     DriftKind `json:"kind"`
	Detail   string    `json:"detail"`   // human-readable "what differs"
	Expected string    `json:"expected"` // value the snapshot recorded
	Actual   string    `json:"actual"`   // value found on the machine
}

// DriftReport is the structured result of `apply --check`. It is emitted
// verbatim as JSON (--json) and drives the human rendering.
type DriftReport struct {
	Compliant    bool     `json:"compliant"`
	MissingTools []string `json:"missing_tools"`
	// UnknownTools are in the snapshot but not in this build's catalog —
	// surfaced but not counted as drift (this opsforge simply can't judge
	// them).
	UnknownTools []string `json:"unknown_tools,omitempty"`
	Drift        []Drift  `json:"drift"`
}

// CheckDrift compares a snapshot to the live machine state WITHOUT changing
// anything, returning a structured drift report. A machine is compliant
// when no baseline tool is missing and none of the captured
// theme/guards/shell/version facts differ. Pure: all inputs are provided.
func CheckDrift(s Snapshot, cat *catalog.Catalog, cur CurrentState) DriftReport {
	// Initialize slices non-nil so the JSON report always has [] arrays
	// (stable shape for CI consumers), never null.
	r := DriftReport{Compliant: true, MissingTools: []string{}, Drift: []Drift{}}

	// --- tools ---------------------------------------------------------
	for _, name := range s.Tools {
		if _, ok := cat.Tool(name); !ok {
			r.UnknownTools = append(r.UnknownTools, name)
			continue
		}
		if !cur.Installed[name] {
			r.MissingTools = append(r.MissingTools, name)
		}
	}
	sort.Strings(r.MissingTools)
	if len(r.MissingTools) > 0 {
		r.Compliant = false
	}

	// --- shell environment ---------------------------------------------
	if s.Shell.Enabled && !cur.ShellEnabled {
		r.Drift = append(r.Drift, Drift{
			Kind:     DriftShell,
			Detail:   "shell environment expected but not installed",
			Expected: "enabled",
			Actual:   "disabled",
		})
	}

	// --- theme ---------------------------------------------------------
	// Only enforce a theme the source machine actually pinned; an
	// auto-resolved theme is not part of the baseline.
	if s.Theme.Persisted && s.Theme.Name != "" {
		if !cur.ThemePersisted || cur.ThemeName != s.Theme.Name {
			actual := cur.ThemeName
			if !cur.ThemePersisted {
				actual = "not pinned"
			}
			r.Drift = append(r.Drift, Drift{
				Kind:     DriftTheme,
				Detail:   "theme differs from baseline",
				Expected: s.Theme.Name,
				Actual:   actual,
			})
		}
	}

	// --- guards --------------------------------------------------------
	// Compare on normalized content so trailing-whitespace noise doesn't
	// register as drift. An empty snapshot guards field means "no custom
	// policy captured" and is not enforced.
	if strings.TrimSpace(s.Guards.YAML) != "" {
		if normalizeYAML(cur.GuardsYAML) != normalizeYAML(s.Guards.YAML) {
			detail := "guards.yaml differs from baseline"
			actual := "present but different"
			if strings.TrimSpace(cur.GuardsYAML) == "" {
				detail = "guards.yaml missing (baseline defines security rules)"
				actual = "absent"
			}
			r.Drift = append(r.Drift, Drift{
				Kind:     DriftGuards,
				Detail:   detail,
				Expected: "present",
				Actual:   actual,
			})
		}
	}

	// --- version manager -----------------------------------------------
	if s.Versions.Manager != "" && cur.VersionManager != s.Versions.Manager {
		actual := cur.VersionManager
		if actual == "" {
			actual = "none"
		}
		r.Drift = append(r.Drift, Drift{
			Kind:     DriftVersion,
			Detail:   "version manager differs from baseline",
			Expected: s.Versions.Manager,
			Actual:   actual,
		})
	}

	if len(r.Drift) > 0 {
		r.Compliant = false
	}
	return r
}

// normalizeYAML trims surrounding whitespace and normalizes line endings so
// cosmetic differences don't count as guard drift.
func normalizeYAML(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}
