package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeVersion(t *testing.T) {
	cases := map[string]string{
		"Client Version: v1.32.4-dispatcher": "1.32.4",
		"argocd: v2.11.0+d3f33c0":            "2.11.0",
		"OpenTofu v1.10.5":                   "1.10.5",
		"no version here":                    "",
		"v3.19.0+g3d8990f":                   "3.19.0",
	}
	for in, want := range cases {
		if got := NormalizeVersion(in); got != want {
			t.Errorf("NormalizeVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

// mkVuln builds an osvVuln with a single affected range.
func mkVuln(id, introduced, fixed, sev string) osvVuln {
	v := osvVuln{ID: id}
	rng := struct {
		Type   string `json:"type"`
		Events []struct {
			Introduced string `json:"introduced"`
			Fixed      string `json:"fixed"`
		} `json:"events"`
	}{Type: "SEMVER"}
	ev := struct {
		Introduced string `json:"introduced"`
		Fixed      string `json:"fixed"`
	}{Introduced: introduced, Fixed: fixed}
	rng.Events = append(rng.Events, ev)
	aff := struct {
		Ranges []struct {
			Type   string `json:"type"`
			Events []struct {
				Introduced string `json:"introduced"`
				Fixed      string `json:"fixed"`
			} `json:"events"`
		} `json:"ranges"`
	}{}
	aff.Ranges = append(aff.Ranges, rng)
	v.Affected = append(v.Affected, aff)
	v.DatabaseSpecific.Severity = sev
	return v
}

func TestAffectsVersion(t *testing.T) {
	cases := []struct {
		name                       string
		version, introduced, fixed string
		want                       bool
	}{
		{"in range", "2.11.0", "2.0.0", "2.13.0", true},
		{"at introduced", "2.0.0", "2.0.0", "2.13.0", true},
		{"at fixed (not affected)", "2.13.0", "2.0.0", "2.13.0", false},
		{"before introduced", "1.9.0", "2.0.0", "2.13.0", false},
		{"after fixed", "3.0.0", "2.0.0", "2.13.0", false},
		{"open range affected", "5.0.0", "1.0.0", "", true},
		{"open range before intro", "0.9.0", "1.0.0", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := mkVuln("X", c.introduced, c.fixed, "HIGH")
			if got := affectsVersion(v, c.version); got != c.want {
				t.Errorf("affectsVersion(%s in [%s,%s)) = %v, want %v",
					c.version, c.introduced, c.fixed, got, c.want)
			}
		})
	}
}

func TestToVulnsFiltersAndDedupes(t *testing.T) {
	resp := osvResponse{Vulns: []osvVuln{
		mkVuln("GHSA-aaa", "2.0.0", "2.13.0", "CRITICAL"), // affects 2.11.0
		mkVuln("GHSA-bbb", "3.0.0", "3.2.0", "HIGH"),      // does NOT affect 2.11.0
	}}
	// Give the first one a CVE alias so preferredID resolves to it, and a
	// duplicate entry to prove dedupe.
	resp.Vulns[0].Aliases = []string{"CVE-2025-0001"}
	dup := mkVuln("GO-2025-0001", "2.0.0", "2.13.0", "CRITICAL")
	dup.Aliases = []string{"CVE-2025-0001"}
	resp.Vulns = append(resp.Vulns, dup)

	got := toVulns(resp, "2.11.0")
	if len(got) != 1 {
		t.Fatalf("expected 1 vuln (filtered + deduped), got %d: %+v", len(got), got)
	}
	if got[0].ID != "CVE-2025-0001" {
		t.Errorf("preferred ID = %q, want CVE-2025-0001", got[0].ID)
	}
	if got[0].Severity != SevCritical {
		t.Errorf("severity = %v, want CRITICAL", got[0].Severity)
	}
	if got[0].FixedIn != "2.13.0" {
		t.Errorf("fixedIn = %q, want 2.13.0", got[0].FixedIn)
	}
}

func TestQueryAtParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(osvResponse{Vulns: []osvVuln{
			mkVuln("GHSA-z", "1.0.0", "1.5.0", "HIGH"),
		}})
	}))
	defer srv.Close()

	vulns, err := QueryAt(context.Background(), srv.URL, "Go", "example.com/x", "1.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(vulns) != 1 || vulns[0].Severity != SevHigh {
		t.Errorf("QueryAt returned %+v", vulns)
	}
}
