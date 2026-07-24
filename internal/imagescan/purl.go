// Package imagescan scans a container image for CVEs and correlates the result
// with the workstation's own toolbox.
//
// opsforge does NOT re-implement SBOM extraction — that's syft/trivy's job, and
// they do it well. It shells out to whichever is installed (the same way
// opsforge delegates version pinning to mise/asdf), then runs the image's
// components through opsforge's OWN OSV engine and correlates them with the
// workstation SBOM. The value opsforge adds is the correlation, not another
// image scanner.
package imagescan

import (
	"strings"

	"github.com/Mrg77/opsforge/internal/audit"
)

// Component is one package from an image SBOM, reduced to what the OSV engine
// needs.
type Component struct {
	Name      string // display name
	Ecosystem string // OSV ecosystem (Go, PyPI, npm…), empty if unmapped
	Package   string // OSV package name
	Version   string // normalized semver
	PURL      string // original purl, for reference
}

// purlToOSV turns a Package URL into an OSV (ecosystem, name) pair. It is the
// inverse of sbom.purlEcosystem: purl uses types like "golang"/"cargo"/"gem",
// OSV uses "Go"/"crates.io"/"RubyGems". Returns ("","") for a purl type opsforge
// doesn't map to an OSV ecosystem (those components are skipped, not guessed).
func purlToOSV(purlType, name string) (ecosystem, pkg string) {
	switch strings.ToLower(purlType) {
	case "golang":
		return "Go", name
	case "npm":
		return "npm", name
	case "pypi":
		return "PyPI", name
	case "cargo":
		return "crates.io", name
	case "gem":
		return "RubyGems", name
	case "composer":
		return "Packagist", name
	case "deb":
		return "Debian", name
	case "apk":
		return "Alpine", name
	case "maven":
		return "Maven", name
	default:
		return "", ""
	}
}

// parsePURL splits a Package URL into its type, name and version. It handles the
// common form pkg:TYPE/NAMESPACE/NAME@VERSION?QUALIFIERS#SUBPATH, keeping the
// namespace as part of the name (OSV package names for Go/Maven include it).
// Returns ok=false for anything that isn't a pkg: URL.
func parsePURL(purl string) (typ, name, version string, ok bool) {
	if !strings.HasPrefix(purl, "pkg:") {
		return "", "", "", false
	}
	rest := strings.TrimPrefix(purl, "pkg:")

	// Strip qualifiers (?a=b) and subpath (#...) — not needed for OSV lookup.
	if i := strings.IndexAny(rest, "?#"); i >= 0 {
		rest = rest[:i]
	}
	// Split off the version (@…), which may itself contain no slashes.
	if i := strings.LastIndex(rest, "@"); i >= 0 {
		version = rest[i+1:]
		rest = rest[:i]
	}
	// rest is now TYPE/NAMESPACE.../NAME.
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		return "", "", "", false
	}
	typ = rest[:slash]
	name = rest[slash+1:]
	if typ == "" || name == "" {
		return "", "", "", false
	}
	return typ, name, version, true
}

// componentFromPURL builds a Component (with OSV coordinates) from a purl and a
// fallback display name/version. Returns ok=false when the purl maps to no OSV
// ecosystem or carries no usable version.
func componentFromPURL(purl, fallbackName, fallbackVersion string) (Component, bool) {
	typ, name, version, ok := parsePURL(purl)
	if !ok {
		return Component{}, false
	}
	eco, pkg := purlToOSV(typ, name)
	if eco == "" {
		return Component{}, false
	}
	if version == "" {
		version = fallbackVersion
	}
	nv := audit.NormalizeVersion(version)
	if nv == "" {
		return Component{}, false // OSV needs a concrete version to match
	}
	disp := name
	if fallbackName != "" {
		disp = fallbackName
	}
	return Component{
		Name:      disp,
		Ecosystem: eco,
		Package:   pkg,
		Version:   nv,
		PURL:      purl,
	}, true
}
