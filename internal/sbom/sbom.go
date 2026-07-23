// Package sbom builds a CycloneDX 1.6 Software Bill of Materials of the
// installed workstation tools — each tool a component (with a PURL when
// the catalog maps it to an ecosystem), optionally enriched with the CVEs
// found by the audit. It reuses data opsforge already has (detection,
// catalog OSV mapping, audit findings), so no concurrent tool does this:
// a signable, CVE-correlated SBOM of your DevOps workstation.
package sbom

import (
	"fmt"
	"strings"

	"github.com/Mrg77/opsforge/internal/audit"
)

// cycloneDX schema version we emit.
const specVersion = "1.6"

// Doc is the top-level CycloneDX document. Only the fields opsforge
// populates are modeled; the shape follows the CycloneDX 1.6 JSON schema.
type Doc struct {
	BOMFormat       string          `json:"bomFormat"`
	SpecVersion     string          `json:"specVersion"`
	Version         int             `json:"version"`
	Metadata        Metadata        `json:"metadata"`
	Components      []Component     `json:"components"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities,omitempty"`
}

type Metadata struct {
	Timestamp string     `json:"timestamp,omitempty"`
	Tools     []ToolMeta `json:"tools,omitempty"`
	Component *Component `json:"component,omitempty"`
}

type ToolMeta struct {
	Vendor  string `json:"vendor"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// Component is one installed tool. BOMRef ties it to any vulnerabilities.
type Component struct {
	Type    string `json:"type"` // "application"
	BOMRef  string `json:"bom-ref"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	PURL    string `json:"purl,omitempty"`
	Desc    string `json:"description,omitempty"`
}

// Vulnerability follows the CycloneDX vulnerabilities schema (subset).
type Vulnerability struct {
	ID             string          `json:"id"`
	Source         Source          `json:"source"`
	Ratings        []Rating        `json:"ratings,omitempty"`
	Description    string          `json:"description,omitempty"`
	Recommendation string          `json:"recommendation,omitempty"`
	Affects        []AffectsTarget `json:"affects"`
}

type Source struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

type Rating struct {
	Severity string `json:"severity"` // critical|high|medium|low|unknown
	Method   string `json:"method,omitempty"`
}

type AffectsTarget struct {
	Ref string `json:"ref"`
}

// Input describes one installed tool to include in the SBOM.
type Input struct {
	Name        string
	Version     string // raw detected version string
	Description string
	// Ecosystem/Package come from the catalog's OSV mapping when present;
	// they build the PURL. Empty when the tool has no mapping.
	Ecosystem string
	Package   string
	// Findings are the audit results for this tool (may be empty).
	Vulns []audit.Vuln
}

// Build assembles a CycloneDX document from the installed tools. timestamp
// is passed in (the caller stamps it) so Build stays pure and testable.
func Build(inputs []Input, timestamp string) Doc {
	doc := Doc{
		BOMFormat:   "CycloneDX",
		SpecVersion: specVersion,
		Version:     1,
		Metadata: Metadata{
			Timestamp: timestamp,
			Tools: []ToolMeta{
				{Vendor: "opsforge", Name: "opsforge"},
			},
		},
	}
	for _, in := range inputs {
		ref := "tool:" + in.Name
		doc.Components = append(doc.Components, Component{
			Type:    "application",
			BOMRef:  ref,
			Name:    in.Name,
			Version: audit.NormalizeVersion(in.Version),
			PURL:    purl(in),
			Desc:    in.Description,
		})
		for _, v := range in.Vulns {
			doc.Vulnerabilities = append(doc.Vulnerabilities, vuln(ref, v))
		}
	}
	return doc
}

// purl builds a Package URL (purl) for a tool from its OSV ecosystem +
// package, e.g. pkg:golang/helm.sh/helm/v3@3.14.0. Returns "" when the
// tool has no ecosystem mapping — a bare generic purl carries no useful
// coordinates, so we omit it rather than emit a misleading one.
func purl(in Input) string {
	if in.Ecosystem == "" || in.Package == "" {
		return ""
	}
	eco := purlEcosystem(in.Ecosystem)
	ver := audit.NormalizeVersion(in.Version)
	p := "pkg:" + eco + "/" + in.Package
	if ver != "" {
		p += "@" + ver
	}
	return p
}

// purlEcosystem maps an OSV ecosystem to the purl "type" token.
func purlEcosystem(osv string) string {
	switch strings.ToLower(osv) {
	case "go":
		return "golang"
	case "npm":
		return "npm"
	case "pypi":
		return "pypi"
	case "crates.io":
		return "cargo"
	case "rubygems":
		return "gem"
	case "packagist":
		return "composer"
	default:
		return strings.ToLower(osv)
	}
}

func vuln(ref string, v audit.Vuln) Vulnerability {
	out := Vulnerability{
		ID:          v.ID,
		Source:      Source{Name: sourceName(v.ID), URL: advisoryURL(v.ID)},
		Description: v.Summary,
		Affects:     []AffectsTarget{{Ref: ref}},
	}
	if sev := severityToken(v.Severity); sev != "" {
		out.Ratings = []Rating{{Severity: sev, Method: "other"}}
	}
	if v.FixedIn != "" {
		out.Recommendation = "Upgrade to " + v.FixedIn + " or later."
	}
	return out
}

func severityToken(s audit.Severity) string {
	switch s {
	case audit.SevCritical:
		return "critical"
	case audit.SevHigh:
		return "high"
	case audit.SevMedium:
		return "medium"
	case audit.SevLow:
		return "low"
	default:
		return "unknown"
	}
}

func sourceName(id string) string {
	if strings.HasPrefix(id, "CVE-") {
		return "NVD"
	}
	return "OSV"
}

func advisoryURL(id string) string {
	if strings.HasPrefix(id, "CVE-") {
		return "https://nvd.nist.gov/vuln/detail/" + id
	}
	return "https://osv.dev/vulnerability/" + id
}

// Summary is a short human line describing an SBOM (used by the non-JSON
// output path).
func (d Doc) Summary() string {
	return fmt.Sprintf("CycloneDX %s · %d component(s) · %d vulnerabilit(y/ies)",
		d.SpecVersion, len(d.Components), len(d.Vulnerabilities))
}
