package notices

import (
	"testing"
	"time"
)

func TestItemsOrderAndSeverity(t *testing.T) {
	d := Digest{
		CVEHighOrCritical: 1,
		Updates:           3,
		SecretsCritical:   2,
		SelfUpdate:        true,
		SelfLatestVersion: "v1.2.3",
	}
	items := d.Items()
	if len(items) != 4 {
		t.Fatalf("want 4 items, got %d: %+v", len(items), items)
	}
	// Critical items (CVE, secrets) must come before warn (updates) and info.
	if items[0].Severity != SevCritical || items[len(items)-1].Severity != SevInfo {
		t.Errorf("items not ordered by severity: %+v", items)
	}
	if d.TopSeverity() != SevCritical {
		t.Errorf("TopSeverity = %v, want critical", d.TopSeverity())
	}
}

func TestItemsEmptyWhenClean(t *testing.T) {
	if items := (Digest{}).Items(); len(items) != 0 {
		t.Errorf("clean digest should have no items, got %+v", items)
	}
	if line := (Digest{}).OneLine(); line != "" {
		t.Errorf("clean digest one-liner should be empty, got %q", line)
	}
}

func TestOneLine(t *testing.T) {
	d := Digest{CVETools: 2, Updates: 1}
	line := d.OneLine()
	if line == "" {
		t.Fatal("expected a one-liner")
	}
	// Must mention the command to see details.
	if want := "opsforge notify"; !contains(line, want) {
		t.Errorf("one-liner %q missing %q", line, want)
	}
}

func TestCVEHighDominatesPlainCVE(t *testing.T) {
	// When both a HIGH/CRITICAL and plain CVE count exist, only the worse
	// line shows (not double-counted).
	d := Digest{CVETools: 5, CVEHighOrCritical: 2}
	n := 0
	for _, it := range d.Items() {
		if contains(it.Text, "CVE") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("expected 1 CVE line, got %d", n)
	}
}

func TestStale(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	if !(Digest{}).Stale(DefaultTTL, now) {
		t.Error("never-refreshed digest should be stale")
	}
	fresh := Digest{RefreshedAt: now.Add(-time.Hour)}
	if fresh.Stale(DefaultTTL, now) {
		t.Error("1h-old digest should be fresh")
	}
}

func TestPluralAndItoa(t *testing.T) {
	if plural(1, "tool") != "1 tool" || plural(3, "tool") != "3 tools" {
		t.Errorf("plural wrong: %q %q", plural(1, "tool"), plural(3, "tool"))
	}
	if itoa(0) != "0" || itoa(42) != "42" {
		t.Errorf("itoa wrong: %q %q", itoa(0), itoa(42))
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
