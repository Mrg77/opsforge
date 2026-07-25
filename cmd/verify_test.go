package cmd

import (
	"testing"
	"time"

	"github.com/Mrg77/opsforge/internal/credscan"
)

func TestExitCodeFor(t *testing.T) {
	high := credscan.Report{Findings: []credscan.Finding{
		{Severity: credscan.SevHigh, SeverityLabel: "HIGH"},
	}}
	low := credscan.Report{Findings: []credscan.Finding{
		{Severity: credscan.SevLow, SeverityLabel: "LOW"},
	}}
	clean := credscan.Report{}

	// Default gate: HIGH/CRITICAL fails, LOW does not, clean passes.
	if exitCodeFor(high, false) == nil {
		t.Error("a HIGH finding must fail the default gate")
	}
	if exitCodeFor(low, false) != nil {
		t.Error("a LOW finding must NOT fail the default gate")
	}
	if exitCodeFor(clean, false) != nil {
		t.Error("a clean report must pass")
	}

	// --strict: any finding fails.
	if exitCodeFor(low, true) == nil {
		t.Error("--strict must fail on a LOW finding")
	}
	if exitCodeFor(clean, true) != nil {
		t.Error("--strict on a clean report must pass")
	}
}

func TestTallyBySeverity(t *testing.T) {
	fs := []credscan.Finding{
		{Severity: credscan.SevCritical},
		{Severity: credscan.SevHigh},
		{Severity: credscan.SevHigh},
		{Severity: credscan.SevMedium},
		{Severity: credscan.SevLow},
	}
	crit, high, other := tallyBySeverity(fs)
	if crit != 1 || high != 2 || other != 2 {
		t.Errorf("tally = (%d,%d,%d), want (1,2,2)", crit, high, other)
	}
}

// The scan itself must be safe to run anywhere: read-only, no panic, no
// external process. We just assert it returns without error-shaped surprises.
func TestScanRunsCleanly(t *testing.T) {
	r := credscan.Scan(time.Now())
	if r.Scanned < 0 {
		t.Error("Scanned count must not be negative")
	}
}
