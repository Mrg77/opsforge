package imagescan

import (
	"context"

	"github.com/Mrg77/opsforge/internal/audit"
)

// ImageFinding is one image component with the CVEs opsforge's OSV engine found
// for it.
type ImageFinding struct {
	Name        string       `json:"name"`
	Ecosystem   string       `json:"ecosystem"`
	Version     string       `json:"version"`
	PURL        string       `json:"purl"`
	Vulns       []audit.Vuln `json:"vulnerabilities,omitempty"`
	TopSeverity string       `json:"top_severity,omitempty"`
}

// ScanComponents runs image components through opsforge's OWN OSV engine (the
// same audit.ScanTools the workstation audit uses) and returns one finding per
// component that carries at least one CVE.
func ScanComponents(ctx context.Context, comps []Component) []ImageFinding {
	targets := make([]audit.ToolTarget, len(comps))
	for i, c := range comps {
		targets[i] = audit.ToolTarget{
			Name:      c.Name,
			Ecosystem: c.Ecosystem,
			Package:   c.Package,
			Version:   c.Version,
		}
	}
	findings := audit.ScanTools(ctx, targets)

	var out []ImageFinding
	for i, f := range findings {
		if len(f.Vulns) == 0 {
			continue
		}
		out = append(out, ImageFinding{
			Name:        comps[i].Name,
			Ecosystem:   comps[i].Ecosystem,
			Version:     comps[i].Version,
			PURL:        comps[i].PURL,
			Vulns:       f.Vulns,
			TopSeverity: f.TopSeverity().String(),
		})
	}
	return out
}

// Drift is one workstation↔image discrepancy for a tool present in both.
type Drift struct {
	Name          string `json:"name"`
	WorkstationV  string `json:"workstation_version"`
	ImageV        string `json:"image_version"`
	VersionDiffer bool   `json:"version_differs"`
}

// WorkstationTool is the minimal view of an installed workstation tool needed
// to correlate with an image (name + normalized version).
type WorkstationTool struct {
	Name    string
	Version string
}

// Correlate compares the image's components against the workstation's tools,
// returning the drift for tools that appear in BOTH — the question trivy can't
// answer: "is a tool I run locally shipped at a different version in this
// image?" Matching is by name (case-insensitive); versions are compared as the
// normalized strings the audit already produces.
func Correlate(image []Component, workstation []WorkstationTool) []Drift {
	wsByName := map[string]string{}
	for _, w := range workstation {
		wsByName[lower(w.Name)] = w.Version
	}
	seen := map[string]bool{}
	var drift []Drift
	for _, c := range image {
		wv, ok := wsByName[lower(c.Name)]
		if !ok || seen[lower(c.Name)] {
			continue
		}
		seen[lower(c.Name)] = true
		drift = append(drift, Drift{
			Name:          c.Name,
			WorkstationV:  wv,
			ImageV:        c.Version,
			VersionDiffer: wv != "" && c.Version != "" && wv != c.Version,
		})
	}
	return drift
}

func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
