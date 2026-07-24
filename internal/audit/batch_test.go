package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// batchServer stands in for OSV: /v1/querybatch maps each query's package name
// to a fixed set of vuln ids, and /v1/vulns/{id} serves details from a table.
// It counts detail fetches so tests can assert dedup.
func batchServer(t *testing.T, idsByPkg map[string][]string, details map[string]osvVuln) (batchURL, vulnBase string, detailHits *int32) {
	t.Helper()
	var hits int32

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/querybatch", func(w http.ResponseWriter, r *http.Request) {
		var req batchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var resp batchResponse
		for _, q := range req.Queries {
			entry := struct {
				Vulns []struct {
					ID string `json:"id"`
				} `json:"vulns"`
				NextPageToken string `json:"next_page_token,omitempty"`
			}{}
			for _, id := range idsByPkg[q.Package.Name] {
				entry.Vulns = append(entry.Vulns, struct {
					ID string `json:"id"`
				}{ID: id})
			}
			// Re-encode via the exported response shape.
			resp.Results = append(resp.Results, struct {
				Vulns []struct {
					ID string `json:"id"`
				} `json:"vulns"`
			}{Vulns: entry.Vulns})
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/v1/vulns/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		id := strings.TrimPrefix(r.URL.Path, "/v1/vulns/")
		v, ok := details[id]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(v)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL + "/v1/querybatch", srv.URL + "/v1/vulns/", &hits
}

func TestScanBatchMatchesOrderAndVersions(t *testing.T) {
	// argocd@2.11.0 is affected by CVE-A (fixed 2.13.0); helm@3.14.0 is clean;
	// kubectl@1.29.0 is affected by CVE-B (fixed 1.30.0).
	idsByPkg := map[string][]string{
		"argocd-pkg":  {"CVE-A"},
		"helm-pkg":    {},
		"kubectl-pkg": {"CVE-B"},
	}
	details := map[string]osvVuln{
		"CVE-A": mkVuln("CVE-A", "2.0.0", "2.13.0", "HIGH"),
		"CVE-B": mkVuln("CVE-B", "1.0.0", "1.30.0", "CRITICAL"),
	}
	batchURL, vulnBase, _ := batchServer(t, idsByPkg, details)

	targets := []ToolTarget{
		{Name: "argocd", Ecosystem: "Go", Package: "argocd-pkg", Version: "2.11.0"},
		{Name: "helm", Ecosystem: "Go", Package: "helm-pkg", Version: "3.14.0"},
		{Name: "kubectl", Ecosystem: "Go", Package: "kubectl-pkg", Version: "1.29.0"},
	}

	findings, ok := scanBatch(context.Background(), targets, batchURL, vulnBase)
	if !ok {
		t.Fatal("scanBatch reported failure on a healthy server")
	}
	if len(findings) != 3 {
		t.Fatalf("want 3 findings, got %d", len(findings))
	}
	// Order preserved.
	if findings[0].Tool != "argocd" || findings[1].Tool != "helm" || findings[2].Tool != "kubectl" {
		t.Fatalf("order not preserved: %+v", findings)
	}
	if len(findings[0].Vulns) != 1 || findings[0].Vulns[0].ID != "CVE-A" || findings[0].Vulns[0].FixedIn != "2.13.0" {
		t.Errorf("argocd finding wrong: %+v", findings[0].Vulns)
	}
	if len(findings[1].Vulns) != 0 {
		t.Errorf("helm should be clean, got %+v", findings[1].Vulns)
	}
	if len(findings[2].Vulns) != 1 || findings[2].Vulns[0].Severity != SevCritical {
		t.Errorf("kubectl finding wrong: %+v", findings[2].Vulns)
	}
}

func TestScanBatchFiltersOutOfRangeVersion(t *testing.T) {
	// The tool is at 3.0.0 but the CVE only affects <2.13.0 → not reported,
	// proving the version matcher still runs on the batch-fetched details.
	idsByPkg := map[string][]string{"x": {"CVE-A"}}
	details := map[string]osvVuln{"CVE-A": mkVuln("CVE-A", "2.0.0", "2.13.0", "HIGH")}
	batchURL, vulnBase, _ := batchServer(t, idsByPkg, details)

	targets := []ToolTarget{{Name: "x", Ecosystem: "Go", Package: "x", Version: "3.0.0"}}
	findings, ok := scanBatch(context.Background(), targets, batchURL, vulnBase)
	if !ok {
		t.Fatal("scanBatch failed")
	}
	if len(findings[0].Vulns) != 0 {
		t.Errorf("a CVE outside the installed version's range must not be reported: %+v", findings[0].Vulns)
	}
}

func TestScanBatchDedupesDetailFetches(t *testing.T) {
	// Two tools share CVE-SHARED → it must be fetched exactly once.
	idsByPkg := map[string][]string{
		"a": {"CVE-SHARED"},
		"b": {"CVE-SHARED"},
	}
	details := map[string]osvVuln{"CVE-SHARED": mkVuln("CVE-SHARED", "0", "", "HIGH")}
	batchURL, vulnBase, hits := batchServer(t, idsByPkg, details)

	targets := []ToolTarget{
		{Name: "a", Ecosystem: "Go", Package: "a", Version: "1.0.0"},
		{Name: "b", Ecosystem: "Go", Package: "b", Version: "1.0.0"},
	}
	if _, ok := scanBatch(context.Background(), targets, batchURL, vulnBase); !ok {
		t.Fatal("scanBatch failed")
	}
	if got := atomic.LoadInt32(hits); got != 1 {
		t.Errorf("shared CVE should be fetched once, was fetched %d times", got)
	}
}

func TestScanBatchFailsWhenBatchEndpointDown(t *testing.T) {
	// Batch endpoint 500s → scanBatch reports ok=false so ScanTools can fall
	// back to the per-tool path.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	targets := []ToolTarget{{Name: "x", Ecosystem: "Go", Package: "x", Version: "1.0.0"}}
	if _, ok := scanBatch(context.Background(), targets, srv.URL, srv.URL+"/v1/vulns/"); ok {
		t.Error("scanBatch should report failure when the batch endpoint errors")
	}
}
