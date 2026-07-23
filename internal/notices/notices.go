// Package notices aggregates everything a DevOps engineer should be nudged
// about on their own machine — CVEs on installed tools, available updates,
// leaked secrets, and a newer opsforge itself — into one cached digest.
//
// Reads are instant (a small JSON cache), so `opsforge notify` and the
// shell startup notice never block on the network; a refresh runs only
// when the cache is stale, in the background.
package notices

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// DefaultTTL — how long a digest is fresh. Long enough to keep the prompt
// instant, short enough to stay useful across a work day.
const DefaultTTL = 6 * time.Hour

// Severity orders notices for display and for the compact one-liner.
type Severity int

const (
	SevInfo Severity = iota
	SevWarn
	SevCritical
)

func (s Severity) String() string {
	switch s {
	case SevCritical:
		return "critical"
	case SevWarn:
		return "warning"
	default:
		return "info"
	}
}

// Digest is the cached, aggregated view of what needs attention.
type Digest struct {
	// RefreshedAt is when the digest was computed (zero = never).
	RefreshedAt time.Time `json:"refreshed_at"`

	// CVEs on installed tools.
	CVETools          int `json:"cve_tools"`
	CVEHighOrCritical int `json:"cve_high_or_critical"`

	// Tools with an available update.
	Updates int `json:"updates"`

	// Leaked-secret findings on the workstation (0 when not scanned).
	Secrets         int  `json:"secrets"`
	SecretsCritical int  `json:"secrets_critical"`
	SecretsScanned  bool `json:"secrets_scanned"`

	// opsforge itself: a newer release than the running binary.
	SelfUpdate        bool   `json:"self_update"`
	SelfLatestVersion string `json:"self_latest_version,omitempty"`
}

// Item is one line in the human digest, produced by Items().
type Item struct {
	Severity Severity
	Text     string
	Fix      string
}

// Items turns the digest into ordered, human-facing lines (most severe
// first). Empty slice means "nothing to report".
func (d Digest) Items() []Item {
	var out []Item
	if d.CVEHighOrCritical > 0 {
		out = append(out, Item{SevCritical,
			plural(d.CVEHighOrCritical, "tool") + " with a HIGH/CRITICAL CVE",
			"opsforge audit"})
	} else if d.CVETools > 0 {
		out = append(out, Item{SevWarn,
			plural(d.CVETools, "tool") + " with a known CVE",
			"opsforge audit"})
	}
	if d.SecretsCritical > 0 {
		out = append(out, Item{SevCritical,
			plural(d.SecretsCritical, "critical secret") + " leaking in your shell/rc/.env",
			"opsforge audit --secrets"})
	} else if d.Secrets > 0 {
		out = append(out, Item{SevWarn,
			plural(d.Secrets, "potential secret leak"),
			"opsforge audit --secrets"})
	}
	if d.Updates > 0 {
		out = append(out, Item{SevWarn,
			plural(d.Updates, "tool") + " can be updated",
			"opsforge upgrade -u"})
	}
	if d.SelfUpdate {
		v := d.SelfLatestVersion
		if v != "" {
			v = " (" + v + ")"
		}
		out = append(out, Item{SevInfo,
			"a newer opsforge is available" + v,
			"opsforge self update"})
	}
	return out
}

// TopSeverity is the worst severity across all items (SevInfo when empty).
func (d Digest) TopSeverity() Severity {
	top := SevInfo
	for _, it := range d.Items() {
		if it.Severity > top {
			top = it.Severity
		}
	}
	return top
}

// OneLine is the compact summary the shell prints on startup, e.g.
// "opsforge: 1 HIGH/CRITICAL CVE · 3 updates — run `opsforge notify`".
// Empty when there's nothing to report.
func (d Digest) OneLine() string {
	items := d.Items()
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, it.Text)
	}
	joined := parts[0]
	for _, p := range parts[1:] {
		joined += " · " + p
	}
	return "opsforge: " + joined + " — run `opsforge notify`"
}

// Stale reports whether the digest is older than ttl (or never computed).
func (d Digest) Stale(ttl time.Duration, now time.Time) bool {
	if d.RefreshedAt.IsZero() {
		return true
	}
	return now.Sub(d.RefreshedAt) > ttl
}

// cachePath is ~/.cache/opsforge/notices.json.
func cachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "opsforge", "notices.json"), nil
}

// Load reads the cached digest. ok=false when there's no cache yet.
func Load() (Digest, bool) {
	p, err := cachePath()
	if err != nil {
		return Digest{}, false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return Digest{}, false
	}
	var d Digest
	if err := json.Unmarshal(data, &d); err != nil {
		return Digest{}, false
	}
	return d, true
}

// Save writes the digest to the cache.
func Save(d Digest) error {
	p, err := cachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return itoa(n) + " " + word + "s"
}

// itoa avoids pulling strconv for one call.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
