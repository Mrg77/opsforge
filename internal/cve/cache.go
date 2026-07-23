// Package cve maintains a cached view of the CVEs affecting installed
// tools, so a prompt or `opsforge status` can surface "N tools have a new
// CVE" instantly — reading a small cache file, never blocking on the
// network. A fresh OSV scan runs only when the cache is stale, and the
// caller decides whether to refresh inline or in the background.
package cve

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/Mrg77/opsforge/internal/audit"
)

// DefaultTTL is how long a cached scan is considered fresh. A DevOps
// toolbox doesn't gain CVEs every minute; a few hours keeps the prompt
// instant while staying useful.
const DefaultTTL = 6 * time.Hour

// Summary is the cached result of a workstation CVE scan.
type Summary struct {
	// ScannedAt is when the scan ran (RFC3339). Zero value = never.
	ScannedAt time.Time `json:"scanned_at"`
	// Vulnerable is the number of installed tools with at least one CVE.
	Vulnerable int `json:"vulnerable"`
	// HighOrCritical is how many of those reach HIGH or CRITICAL.
	HighOrCritical int `json:"high_or_critical"`
	// Tools lists the affected tool names with their worst severity.
	Tools []Affected `json:"tools,omitempty"`
}

// Affected is one vulnerable tool in the cached summary.
type Affected struct {
	Name        string `json:"name"`
	TopSeverity string `json:"top_severity"`
}

// cachePath returns ~/.cache/opsforge/cve-cache.json.
func cachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "opsforge", "cve-cache.json"), nil
}

// Load reads the cached summary. ok is false when there is no cache yet
// (never scanned) or it can't be read — callers treat that as "unknown",
// not "clean".
func Load() (Summary, bool) {
	p, err := cachePath()
	if err != nil {
		return Summary{}, false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return Summary{}, false
	}
	var s Summary
	if err := json.Unmarshal(data, &s); err != nil {
		return Summary{}, false
	}
	return s, true
}

// Stale reports whether the cached summary is older than ttl (or absent).
func (s Summary) Stale(ttl time.Duration, now time.Time) bool {
	if s.ScannedAt.IsZero() {
		return true
	}
	return now.Sub(s.ScannedAt) > ttl
}

// Save writes the summary to the cache, creating the directory as needed.
func Save(s Summary) error {
	p, err := cachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// Summarize turns raw audit findings into a cached Summary, stamped now.
func Summarize(findings []audit.Finding, now time.Time) Summary {
	s := Summary{ScannedAt: now}
	for _, f := range findings {
		if len(f.Vulns) == 0 {
			continue
		}
		s.Vulnerable++
		if f.TopSeverity() >= audit.SevHigh {
			s.HighOrCritical++
		}
		s.Tools = append(s.Tools, Affected{
			Name:        f.Tool,
			TopSeverity: f.TopSeverity().String(),
		})
	}
	return s
}
