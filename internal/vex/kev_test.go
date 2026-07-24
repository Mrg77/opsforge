package vex

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeCache drops a kevCache JSON at the path kevCachePath() will read,
// under a HOME redirected to a temp dir (so no real ~/.cache is touched).
func writeCache(t *testing.T, c kevCache) {
	t.Helper()
	p := kevCachePath()
	if p == "" {
		t.Fatal("kevCachePath() empty — HOME not redirected?")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIdsToSetUppercases(t *testing.T) {
	set := idsToSet([]string{"cve-2025-1", "CVE-2025-2"})
	if !set.Has("CVE-2025-1") || !set.Has("cve-2025-2") {
		t.Errorf("idsToSet should normalize case both ways: %+v", set)
	}
	if len(set) != 2 {
		t.Errorf("want 2 entries, got %d", len(set))
	}
}

func TestSaveThenLoadFreshCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	want := KEVSet{"CVE-2025-1234": true, "CVE-2024-9999": true}
	if err := saveKEVCache(want); err != nil {
		t.Fatalf("saveKEVCache: %v", err)
	}
	got := loadKEVCache()
	if got == nil {
		t.Fatal("loadKEVCache returned nil for a fresh cache we just wrote")
	}
	if !got.Has("CVE-2025-1234") || !got.Has("CVE-2024-9999") {
		t.Errorf("round trip lost entries: %+v", got)
	}
}

func TestLoadKEVCacheRejectsStale(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Written well past the TTL → loadKEVCache (fresh-only) must skip it,
	// but readKEVCacheAnyAge must still return it.
	writeCache(t, kevCache{
		FetchedAt: time.Now().Add(-2 * kevTTL),
		IDs:       []string{"CVE-2020-0001"},
	})

	if fresh := loadKEVCache(); fresh != nil {
		t.Errorf("stale cache should not count as fresh, got %+v", fresh)
	}
	if any := readKEVCacheAnyAge(); any == nil || !any.Has("CVE-2020-0001") {
		t.Errorf("readKEVCacheAnyAge should return the stale entry, got %+v", any)
	}
}

func TestReadKEVCacheMissingAndCorrupt(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// No cache file yet.
	if _, ok := readKEVCache(); ok {
		t.Error("readKEVCache should report ok=false when nothing is cached")
	}

	// Corrupt JSON must not panic and must report ok=false.
	p := kevCachePath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := readKEVCache(); ok {
		t.Error("readKEVCache should report ok=false on corrupt JSON")
	}
}

func TestFetchKEVFromParsesCatalog(t *testing.T) {
	// A minimal slice of the real CISA catalog shape.
	const body = `{"title":"CISA KEV","vulnerabilities":[
		{"cveID":"CVE-2025-0001"},
		{"cveID":"CVE-2025-0002"},
		{"cveID":""}
	]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	set := fetchKEVFrom(srv.URL)
	if set == nil {
		t.Fatal("fetchKEVFrom returned nil for a valid catalog")
	}
	if !set.Has("CVE-2025-0001") || !set.Has("CVE-2025-0002") {
		t.Errorf("missing expected CVEs: %+v", set)
	}
	if len(set) != 2 {
		t.Errorf("empty cveID should be skipped; want 2 entries, got %d", len(set))
	}
}

func TestFetchKEVFromHandlesErrors(t *testing.T) {
	// Non-200 → nil.
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	if set := fetchKEVFrom(bad.URL); set != nil {
		t.Errorf("non-200 should yield nil, got %+v", set)
	}

	// 200 but garbage body → nil, no panic.
	junk := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("{not json"))
	}))
	defer junk.Close()
	if set := fetchKEVFrom(junk.URL); set != nil {
		t.Errorf("invalid JSON should yield nil, got %+v", set)
	}

	// Unreachable URL → nil.
	if set := fetchKEVFrom("http://127.0.0.1:0"); set != nil {
		t.Errorf("connection failure should yield nil, got %+v", set)
	}
}

// TestLoadKEVServedFromFreshCache exercises the LoadKEV fast path without any
// network: a fresh cache short-circuits before fetchKEV is ever reached.
func TestLoadKEVServedFromFreshCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := saveKEVCache(KEVSet{"CVE-2025-1234": true}); err != nil {
		t.Fatal(err)
	}
	set := LoadKEV()
	if set == nil || !set.Has("CVE-2025-1234") {
		t.Errorf("LoadKEV should serve the fresh cache without a fetch, got %+v", set)
	}
}
