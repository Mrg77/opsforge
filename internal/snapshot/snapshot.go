// Package snapshot implements workstation-as-code: capture everything
// opsforge manages (installed tools, user profiles, shell environment
// state) into one shareable YAML file, and compute the plan to rebuild
// that exact workstation on another machine.
//
// The file is designed to live in a dotfiles repo or a gist:
//
//	opsforge snapshot -o my-setup.yaml     # on the old machine
//	opsforge apply https://.../my-setup.yaml   # on the new one
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

// CurrentVersion is the snapshot format version this build writes.
const CurrentVersion = 1

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
}

// ShellState captures the shell-environment side of the workstation.
type ShellState struct {
	Enabled bool `yaml:"enabled"`
}

// Capture builds a Snapshot from the current machine state. Pure with
// respect to its inputs so it is fully testable.
func Capture(cat *catalog.Catalog, statuses map[string]detect.Status,
	userProfiles []catalog.Profile, shellEnabled bool, created time.Time) Snapshot {

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
	}
}

const fileHeader = `# opsforge workstation snapshot — rebuild this exact setup anywhere with:
#   opsforge apply <this-file-or-its-url>
`

// Marshal renders the snapshot as YAML with an explanatory header.
func (s Snapshot) Marshal() ([]byte, error) {
	body, err := yaml.Marshal(s)
	if err != nil {
		return nil, err
	}
	return append([]byte(fileHeader), body...), nil
}

// Load reads a snapshot from a local path or an http(s) URL and
// validates it.
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
