// Package audit cross-references installed tool versions against known
// vulnerabilities using the OSV.dev API (Google's Open Source
// Vulnerabilities database — free, no key required).
//
// A tool is auditable when its catalog entry carries an `osv:` mapping
// (ecosystem + package name), because OSV indexes by package, not by
// "CLI installed via brew". Tools without that mapping are simply
// reported as not-auditable rather than silently skipped.
package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const osvEndpoint = "https://api.osv.dev/v1/query"

var client = &http.Client{Timeout: 20 * time.Second}

// Severity is a coarse, sortable risk level derived from OSV data.
type Severity int

const (
	SevUnknown Severity = iota
	SevLow
	SevMedium
	SevHigh
	SevCritical
)

func (s Severity) String() string {
	switch s {
	case SevCritical:
		return "CRITICAL"
	case SevHigh:
		return "HIGH"
	case SevMedium:
		return "MEDIUM"
	case SevLow:
		return "LOW"
	default:
		return "UNKNOWN"
	}
}

// Vuln is one vulnerability affecting an installed tool.
type Vuln struct {
	ID       string // preferred CVE id, else the OSV id
	Summary  string
	Severity Severity
	FixedIn  string // first fixed version, when OSV states one
}

// Finding is the audit result for a single tool.
type Finding struct {
	Tool      string
	Version   string
	Vulns     []Vuln
	Auditable bool // false when the tool has no OSV mapping or no parsable version
}

// TopSeverity returns the highest severity among the tool's vulns.
func (f Finding) TopSeverity() Severity {
	top := SevUnknown
	for _, v := range f.Vulns {
		if v.Severity > top {
			top = v.Severity
		}
	}
	return top
}

// osvRequest / osvResponse model the subset of the OSV query API we use.
type osvRequest struct {
	Version string     `json:"version"`
	Package osvPackage `json:"package"`
}
type osvPackage struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
}

type osvResponse struct {
	Vulns []osvVuln `json:"vulns"`
}
type osvVuln struct {
	ID       string   `json:"id"`
	Summary  string   `json:"summary"`
	Aliases  []string `json:"aliases"`
	Severity []struct {
		Type  string `json:"type"`
		Score string `json:"score"`
	} `json:"severity"`
	Affected []struct {
		Ranges []struct {
			Type   string `json:"type"`
			Events []struct {
				Introduced string `json:"introduced"`
				Fixed      string `json:"fixed"`
			} `json:"events"`
		} `json:"ranges"`
	} `json:"affected"`
	DatabaseSpecific struct {
		Severity string `json:"severity"`
	} `json:"database_specific"`
}

var semverRe = regexp.MustCompile(`\d+\.\d+\.\d+`)

// NormalizeVersion extracts a bare semver (x.y.z) from a tool's messy
// version string, which OSV needs. Returns "" when none is present.
func NormalizeVersion(raw string) string {
	return semverRe.FindString(raw)
}

// Query asks OSV about a specific ecosystem/package/version. Exposed for
// testing with a swappable endpoint via QueryAt.
func Query(ctx context.Context, ecosystem, name, version string) ([]Vuln, error) {
	return QueryAt(ctx, osvEndpoint, ecosystem, name, version)
}

// ToolTarget identifies one installed tool to check against OSV.
type ToolTarget struct {
	Name      string // display name
	Ecosystem string
	Package   string
	Version   string // normalized semver
}

// ScanTools queries OSV concurrently for every target and returns one
// Finding per tool (empty Vulns when clean; query errors yield an empty
// result rather than failing the whole scan). Reused by the audit
// command and the TUI security view.
func ScanTools(ctx context.Context, targets []ToolTarget) []Finding {
	findings := make([]Finding, len(targets))
	done := make(chan int, len(targets))
	for i, tg := range targets {
		go func(i int, tg ToolTarget) {
			vulns, err := Query(ctx, tg.Ecosystem, tg.Package, tg.Version)
			f := Finding{Tool: tg.Name, Version: tg.Version, Auditable: true}
			if err == nil {
				f.Vulns = vulns
			}
			findings[i] = f
			done <- i
		}(i, tg)
	}
	for range targets {
		<-done
	}
	return findings
}

// QueryAt is Query against an explicit endpoint (used in tests).
func QueryAt(ctx context.Context, endpoint, ecosystem, name, version string) ([]Vuln, error) {
	body, _ := json.Marshal(osvRequest{
		Version: version,
		Package: osvPackage{Ecosystem: ecosystem, Name: name},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("osv.dev returned %s", resp.Status)
	}
	var parsed osvResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return toVulns(parsed, version), nil
}

// toVulns keeps only vulnerabilities that actually affect `version` and
// dedupes by preferred ID (OSV often returns the same CVE under both a
// GHSA and a GO id).
func toVulns(r osvResponse, version string) []Vuln {
	var out []Vuln
	seen := map[string]bool{}
	for _, v := range r.Vulns {
		if !affectsVersion(v, version) {
			continue
		}
		id := preferredID(v)
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, Vuln{
			ID:       id,
			Summary:  v.Summary,
			Severity: severityOf(v),
			FixedIn:  fixedForVersion(v, version),
		})
	}
	return out
}

// affectsVersion reports whether `version` falls inside any affected
// range: at or after an `introduced` and strictly before its `fixed`.
// OSV SEMVER ranges are evaluated with semver ordering; unparsable
// versions fall back to "affected" (safer to over-report than miss).
func affectsVersion(v osvVuln, version string) bool {
	cv := canonical(version)
	if cv == "" {
		return true
	}
	for _, a := range v.Affected {
		for _, r := range a.Ranges {
			if r.Type != "SEMVER" && r.Type != "ECOSYSTEM" && r.Type != "" {
				continue
			}
			introduced := "0"
			for _, e := range r.Events {
				if e.Introduced != "" {
					introduced = e.Introduced
				}
				if e.Fixed != "" {
					if inRange(cv, introduced, e.Fixed) {
						return true
					}
					introduced = "0" // reset for the next introduced/fixed pair
				}
			}
			// An introduced with no matching fixed => affected onwards.
			if hasOpenRange(r.Events) && semverGE(cv, canonical(introduced)) {
				return true
			}
		}
	}
	return false
}

func hasOpenRange(events []struct {
	Introduced string `json:"introduced"`
	Fixed      string `json:"fixed"`
}) bool {
	var lastFixed string
	for _, e := range events {
		if e.Fixed != "" {
			lastFixed = e.Fixed
		}
	}
	return lastFixed == ""
}

// inRange reports introduced <= version < fixed using semver ordering.
func inRange(cv, introduced, fixed string) bool {
	ci, cf := canonical(introduced), canonical(fixed)
	if !semverGE(cv, ci) {
		return false
	}
	if cf == "" {
		return true
	}
	return semver.Compare(cv, cf) < 0
}

func semverGE(a, b string) bool {
	if b == "" || b == "v0" {
		return true
	}
	return semver.Compare(a, b) >= 0
}

// canonical turns a bare version into a semver.Compare-able string
// ("2.11.0" -> "v2.11.0"), stripping any pre-release qualifier noise.
func canonical(v string) string {
	if v == "" || v == "0" {
		return "v0"
	}
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if semver.IsValid(v) {
		return v
	}
	// Strip trailing pre-release/build junk down to a valid core.
	if m := semverRe.FindString(v); m != "" {
		return "v" + m
	}
	return ""
}

// fixedForVersion returns the fix version on the branch that applies to
// `version`, so the advice is actionable (e.g. "fixed in 2.11.13" for a
// 2.11.x install rather than a 3.x fix).
func fixedForVersion(v osvVuln, version string) string {
	cv := canonical(version)
	best := ""
	for _, a := range v.Affected {
		for _, r := range a.Ranges {
			introduced := "0"
			for _, e := range r.Events {
				if e.Introduced != "" {
					introduced = e.Introduced
				}
				if e.Fixed != "" && inRange(cv, introduced, e.Fixed) {
					return e.Fixed // exact branch match
				}
				if e.Fixed != "" && best == "" {
					best = e.Fixed
				}
			}
		}
	}
	return best
}

// preferredID picks the CVE alias when present (users recognize CVEs),
// falling back to the OSV/GHSA id.
func preferredID(v osvVuln) string {
	for _, a := range v.Aliases {
		if strings.HasPrefix(a, "CVE-") {
			return a
		}
	}
	return v.ID
}

// severityOf maps OSV's CVSS or qualitative severity to our coarse scale.
func severityOf(v osvVuln) Severity {
	if s := qualitative(v.DatabaseSpecific.Severity); s != SevUnknown {
		return s
	}
	// Fall back to a CVSS base score when present.
	for _, s := range v.Severity {
		if score := cvssBase(s.Score); score >= 0 {
			switch {
			case score >= 9.0:
				return SevCritical
			case score >= 7.0:
				return SevHigh
			case score >= 4.0:
				return SevMedium
			case score > 0:
				return SevLow
			}
		}
	}
	return SevUnknown
}

func qualitative(s string) Severity {
	switch strings.ToUpper(s) {
	case "CRITICAL":
		return SevCritical
	case "HIGH":
		return SevHigh
	case "MODERATE", "MEDIUM":
		return SevMedium
	case "LOW":
		return SevLow
	default:
		return SevUnknown
	}
}

// cvssBase pulls the base score out of a CVSS vector string. OSV stores
// the full vector (e.g. "CVSS:3.1/AV:N/.../C:H") but not always the
// numeric score, so we only handle the numeric-score case and return -1
// otherwise (callers then fall through to UNKNOWN).
func cvssBase(vector string) float64 {
	// Some OSV entries put a bare number here; try that first.
	var f float64
	if _, err := fmt.Sscanf(vector, "%g", &f); err == nil && f > 0 {
		return f
	}
	return -1
}
