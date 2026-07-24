// Package vex produces OpenVEX documents from opsforge's CVE audit.
//
// VEX (Vulnerability Exploitability eXchange) answers the question a raw
// CVE list can't: "does this vulnerability actually affect me?" — with a
// machine-readable status (affected / not_affected / fixed / under
// investigation) per (product, vulnerability). Since the NVD stopped
// enriching most CVEs in 2026 and only ~2% of high-CVSS CVEs are ever
// exploited, a VEX statement is what lets a downstream scanner (or a
// human) triage instead of drowning.
//
// opsforge already has, per finding, the tool's PURL, the CVE id and the
// fixed version — everything an OpenVEX statement needs. This turns the
// audit into a first-class supply-chain artifact you emit, not just read.
package vex

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Mrg77/opsforge/internal/audit"
)

// OpenVEX context/version per the spec.
const (
	contextURL = "https://openvex.dev/ns/v0.2.0"
	docVersion = 1
)

// Status is an OpenVEX vulnerability status.
type Status string

const (
	StatusNotAffected        Status = "not_affected"
	StatusAffected           Status = "affected"
	StatusFixed              Status = "fixed"
	StatusUnderInvestigation Status = "under_investigation"
)

// Doc is an OpenVEX document.
type Doc struct {
	Context    string      `json:"@context"`
	ID         string      `json:"@id"`
	Author     string      `json:"author"`
	Timestamp  string      `json:"timestamp"`
	Version    int         `json:"version"`
	Statements []Statement `json:"statements"`
}

// Statement asserts the status of one vulnerability for one product.
type Statement struct {
	Vulnerability Vulnerability `json:"vulnerability"`
	Products      []Product     `json:"products"`
	Status        Status        `json:"status"`
	// ActionStatement is required by the spec when status is affected or
	// under_investigation — it says what to do about it.
	ActionStatement string `json:"action_statement,omitempty"`
}

type Vulnerability struct {
	Name string `json:"name"` // the CVE / advisory id
}

type Product struct {
	ID string `json:"@id"` // a PURL identifying the affected component
}

// Input is one audited tool: its PURL (built from the catalog OSV mapping)
// and the vulnerabilities the scan found.
type Input struct {
	PURL  string
	Vulns []audit.Vuln
}

// Build turns audited tools into an OpenVEX document. id/timestamp are
// passed in so Build stays pure and testable (the caller stamps them).
//
// Semantics: every CVE the scan reported is "affected" for the tool at its
// current version, with an action to upgrade to FixedIn when known. (A
// tool the scan cleared simply has no statement — VEX asserts what's known,
// not the absence of every possible CVE.)
func Build(inputs []Input, id, timestamp string) Doc {
	doc := Doc{
		Context:   contextURL,
		ID:        id,
		Author:    "opsforge",
		Timestamp: timestamp,
		Version:   docVersion,
	}
	for _, in := range inputs {
		if in.PURL == "" {
			continue // no coordinates → can't identify the product
		}
		for _, v := range in.Vulns {
			st := Statement{
				Vulnerability: Vulnerability{Name: v.ID},
				Products:      []Product{{ID: in.PURL}},
				Status:        StatusAffected,
			}
			if v.FixedIn != "" {
				st.ActionStatement = "Upgrade to " + v.FixedIn + " or later."
			} else {
				st.ActionStatement = "No fixed version is published yet; monitor the advisory."
			}
			doc.Statements = append(doc.Statements, st)
		}
	}
	// Deterministic order (by vuln id then product) so the same scan yields
	// the same document — important for diffing and signing.
	sort.Slice(doc.Statements, func(a, b int) bool {
		if doc.Statements[a].Vulnerability.Name != doc.Statements[b].Vulnerability.Name {
			return doc.Statements[a].Vulnerability.Name < doc.Statements[b].Vulnerability.Name
		}
		return doc.Statements[a].Products[0].ID < doc.Statements[b].Products[0].ID
	})
	return doc
}

// Summary is a one-line human description of a VEX document.
func (d Doc) Summary() string {
	return fmt.Sprintf("OpenVEX · %d statement(s) · %d affected component(s)",
		len(d.Statements), countProducts(d))
}

func countProducts(d Doc) int {
	seen := map[string]bool{}
	for _, s := range d.Statements {
		for _, p := range s.Products {
			seen[p.ID] = true
		}
	}
	return len(seen)
}

// KEVSet is the set of CVE ids known to be actively exploited (CISA KEV).
// Injected by the caller so this package stays offline/pure.
type KEVSet map[string]bool

// Has reports whether a CVE id is in the KEV set (case-insensitive on the
// "CVE-" prefix, exact otherwise).
func (k KEVSet) Has(id string) bool {
	if k == nil {
		return false
	}
	return k[strings.ToUpper(id)]
}
