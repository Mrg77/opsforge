package vex

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// kevURL is CISA's Known Exploited Vulnerabilities catalog (public JSON).
// A CVE listed here is being actively exploited in the wild — the single
// strongest signal to prioritize, now that CVSS alone is unreliable.
const kevURL = "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json"

// kevTTL is how long a cached KEV catalog is considered fresh (it updates
// roughly daily).
const kevTTL = 24 * time.Hour

// LoadKEV returns the set of actively-exploited CVE ids, from a local cache
// when fresh, else fetched from CISA and cached. Never fails hard: on any
// error it returns an empty set (KEV enrichment is best-effort — its
// absence must not break an audit).
func LoadKEV() KEVSet {
	if set := loadKEVCache(); set != nil {
		return set
	}
	set := fetchKEV()
	if set != nil {
		_ = saveKEVCache(set)
		return set
	}
	// Fetch failed but a stale cache is better than nothing.
	if stale := readKEVCacheAnyAge(); stale != nil {
		return stale
	}
	return KEVSet{}
}

func kevCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", "opsforge", "kev.json")
}

type kevCache struct {
	FetchedAt time.Time `json:"fetched_at"`
	IDs       []string  `json:"ids"`
}

// loadKEVCache returns the cached set only if it's still fresh.
func loadKEVCache() KEVSet {
	c, ok := readKEVCache()
	if !ok || time.Since(c.FetchedAt) > kevTTL {
		return nil
	}
	return idsToSet(c.IDs)
}

func readKEVCacheAnyAge() KEVSet {
	c, ok := readKEVCache()
	if !ok {
		return nil
	}
	return idsToSet(c.IDs)
}

func readKEVCache() (kevCache, bool) {
	p := kevCachePath()
	if p == "" {
		return kevCache{}, false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return kevCache{}, false
	}
	var c kevCache
	if err := json.Unmarshal(data, &c); err != nil {
		return kevCache{}, false
	}
	return c, true
}

func saveKEVCache(set KEVSet) error {
	p := kevCachePath()
	if p == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	data, err := json.Marshal(kevCache{FetchedAt: time.Now().UTC(), IDs: ids})
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// fetchKEV downloads the CISA catalog and extracts the CVE ids. Returns nil
// on any failure.
func fetchKEV() KEVSet {
	return fetchKEVFrom(kevURL)
}

// fetchKEVFrom is fetchKEV against an explicit URL, so the parsing path can be
// exercised in tests against a local server without hitting cisa.gov.
func fetchKEVFrom(url string) KEVSet {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20)) // 16 MiB cap
	if err != nil {
		return nil
	}
	var doc struct {
		Vulnerabilities []struct {
			CveID string `json:"cveID"`
		} `json:"vulnerabilities"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil
	}
	set := make(KEVSet, len(doc.Vulnerabilities))
	for _, v := range doc.Vulnerabilities {
		if v.CveID != "" {
			set[strings.ToUpper(v.CveID)] = true
		}
	}
	return set
}

func idsToSet(ids []string) KEVSet {
	set := make(KEVSet, len(ids))
	for _, id := range ids {
		set[strings.ToUpper(id)] = true
	}
	return set
}
