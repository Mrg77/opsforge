// Package project is opsforge's project-level layer: an opsforge.yaml
// committed in a repo declares the tools a project needs, and
// `opsforge sync` installs them — the reproducibility angle mise/devbox
// own, but with opsforge's guard policy and CVE gate available at the
// project level too, which none of them combine.
package project

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
)

// FileName is the conventional project manifest name, discovered by walking
// up from the working directory (like .git, .tool-versions…).
const FileName = "opsforge.yaml"

// Project is a repo's declared tooling needs.
type Project struct {
	// Version pins the manifest format (currently 1).
	Version int `yaml:"version"`
	// Tools are catalog tool names the project requires.
	Tools []string `yaml:"tools"`
	// Profiles are catalog/user profile names to expand into tools.
	Profiles []string `yaml:"profiles"`
	// FailOn optionally gates `sync`: "high" or "critical" makes sync fail
	// when a required tool carries a CVE at or above that severity.
	FailOn string `yaml:"fail_on,omitempty"`
}

// Find walks up from dir looking for opsforge.yaml, returning its path or
// ok=false. This lets `opsforge sync` work from any subdirectory of a repo.
func Find(dir string) (string, bool) {
	for {
		p := filepath.Join(dir, FileName)
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
		parent := filepath.Dir(dir)
		if parent == dir { // reached the filesystem root
			return "", false
		}
		dir = parent
	}
}

// Load reads and validates a project manifest.
func Load(path string) (*Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Project
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // reject typos in the manifest
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if p.Version != 1 {
		return nil, fmt.Errorf("%s: unsupported version %d (want 1)", path, p.Version)
	}
	switch p.FailOn {
	case "", "high", "critical":
	default:
		return nil, fmt.Errorf("%s: fail_on must be high or critical, got %q", path, p.FailOn)
	}
	return &p, nil
}

// ResolveTools expands the manifest's tools + profiles into a de-duplicated,
// order-preserving list of catalog tool names. Unknown profiles/tools are
// returned separately so the caller can report them without failing hard.
func (p *Project) ResolveTools(cat *catalog.Catalog) (tools []string, unknown []string) {
	seen := map[string]bool{}
	add := func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		if _, ok := cat.Tool(name); ok {
			tools = append(tools, name)
		} else {
			unknown = append(unknown, name)
		}
	}
	for _, prof := range p.Profiles {
		if pr, ok := cat.Profile(prof); ok {
			for _, name := range pr.Tools {
				add(name)
			}
		} else {
			unknown = append(unknown, "profile:"+prof)
		}
	}
	for _, name := range p.Tools {
		add(name)
	}
	return tools, unknown
}

// Plan is what `sync` would do: which required tools are missing vs present.
type Plan struct {
	Install []string
	Present []string
	Unknown []string
}

// BuildPlan diffs the resolved project tools against the current machine.
func BuildPlan(p *Project, cat *catalog.Catalog, statuses map[string]detect.Status) Plan {
	tools, unknown := p.ResolveTools(cat)
	plan := Plan{Unknown: unknown}
	for _, name := range tools {
		if statuses[name].Installed {
			plan.Present = append(plan.Present, name)
		} else {
			plan.Install = append(plan.Install, name)
		}
	}
	return plan
}

// ExampleManifest is written by `opsforge sync --init`.
const ExampleManifest = `# opsforge project manifest — commit this in your repo so anyone can
# reproduce the toolchain with a single 'opsforge sync'.
version: 1

# Tools this project needs (catalog names — see 'opsforge list all').
tools:
  - kubectl
  - helm
  - terraform

# Optionally pull in whole stack profiles (see 'opsforge profiles').
profiles:
  - core

# Optionally gate 'opsforge sync' on CVEs in the required tools:
#   fail_on: high      # sync exits non-zero on a HIGH or CRITICAL CVE
# (unset = don't gate)
`
