package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// OSV's batch API identifies affected packages in one round-trip, then each
// distinct vulnerability is fetched once. This replaces the old "one /v1/query
// per tool" fan-out: for a toolbox of N tools carrying M distinct CVEs it is
// 1 + M requests (M deduped across tools) instead of N. On the common case —
// a healthy machine where M is near zero — that collapses to a single call;
// only a heavily-vulnerable box (M > N) makes more, and those detail fetches
// are bounded and concurrent. The point is fewer calls on the normal path and
// using OSV's rate-limit-friendly batch endpoint; the version-matching engine
// (toVulns) is unchanged — only the transport differs.
const (
	osvQueryBatch = "https://api.osv.dev/v1/querybatch" // id-only batch
	osvVulnByID   = "https://api.osv.dev/v1/vulns/"     // full detail per id
)

// batchRequest / batchResponse model /v1/querybatch. The batch endpoint
// returns ids only (id + modified), in the same order as the queries.
type batchRequest struct {
	Queries []osvRequest `json:"queries"`
}
type batchResponse struct {
	Results []struct {
		Vulns []struct {
			ID string `json:"id"`
		} `json:"vulns"`
	} `json:"results"`
}

// scanBatch runs the whole toolbox through OSV using the batch endpoint and
// per-id detail fetches, returning one Finding per target (order preserved).
// endpoints are injectable for tests. It returns ok=false when the batch call
// itself fails, so ScanTools can fall back to the per-tool path.
func scanBatch(ctx context.Context, targets []ToolTarget, batchURL, vulnBaseURL string) ([]Finding, bool) {
	findings := make([]Finding, len(targets))
	for i, tg := range targets {
		findings[i] = Finding{Tool: tg.Name, Version: tg.Version, Auditable: true}
	}

	// 1. One request: which (tool → vuln ids)?
	idsPerTarget, err := queryBatch(ctx, targets, batchURL)
	if err != nil {
		return nil, false
	}

	// 2. Fetch each distinct id once (deduped across every tool).
	distinct := map[string]bool{}
	for _, ids := range idsPerTarget {
		for _, id := range ids {
			distinct[id] = true
		}
	}
	details := fetchVulnDetails(ctx, distinct, vulnBaseURL)

	// 3. Reuse the existing matcher per tool against its detailed vulns.
	for i, ids := range idsPerTarget {
		resp := osvResponse{}
		for _, id := range ids {
			if v, ok := details[id]; ok {
				resp.Vulns = append(resp.Vulns, v)
			}
		}
		findings[i].Vulns = toVulns(resp, targets[i].Version)
	}
	return findings, true
}

// queryBatch POSTs all targets in one call and returns, per target index, the
// list of vulnerability ids OSV reported (order matches targets).
func queryBatch(ctx context.Context, targets []ToolTarget, endpoint string) ([][]string, error) {
	reqBody := batchRequest{Queries: make([]osvRequest, len(targets))}
	for i, tg := range targets {
		reqBody.Queries[i] = osvRequest{
			Version: tg.Version,
			Package: osvPackage{Ecosystem: tg.Ecosystem, Name: tg.Package},
		}
	}
	body, _ := json.Marshal(reqBody)
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
		return nil, fmt.Errorf("osv.dev querybatch returned %s", resp.Status)
	}
	var parsed batchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if len(parsed.Results) != len(targets) {
		return nil, fmt.Errorf("osv.dev querybatch returned %d results for %d queries",
			len(parsed.Results), len(targets))
	}
	out := make([][]string, len(targets))
	for i, r := range parsed.Results {
		for _, v := range r.Vulns {
			out[i] = append(out[i], v.ID)
		}
	}
	return out, nil
}

// fetchVulnDetails fetches full OSV records for a set of ids concurrently
// (bounded), returning id → record. Ids that fail to fetch are simply absent
// (best-effort — a detail we can't fetch just isn't reported).
func fetchVulnDetails(ctx context.Context, ids map[string]bool, baseURL string) map[string]osvVuln {
	out := make(map[string]osvVuln, len(ids))
	if len(ids) == 0 {
		return out
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 12) // same bound the old per-tool fan-out used
	for id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			v, err := fetchVuln(ctx, baseURL+id)
			if err != nil {
				return
			}
			mu.Lock()
			out[id] = v
			mu.Unlock()
		}(id)
	}
	wg.Wait()
	return out
}

func fetchVuln(ctx context.Context, url string) (osvVuln, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return osvVuln{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return osvVuln{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return osvVuln{}, fmt.Errorf("osv.dev vulns returned %s", resp.Status)
	}
	var v osvVuln
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return osvVuln{}, err
	}
	return v, nil
}
