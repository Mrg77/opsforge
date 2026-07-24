package imagescan

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// Backend is an external SBOM generator opsforge can drive.
type Backend struct {
	Bin  string // executable name
	Args func(image string) []string
}

// backends lists the SBOM generators opsforge knows how to drive, in order of
// preference. Both emit CycloneDX JSON to stdout.
var backends = []Backend{
	{Bin: "syft", Args: func(img string) []string {
		return []string{"scan", img, "-o", "cyclonedx-json", "-q"}
	}},
	{Bin: "trivy", Args: func(img string) []string {
		return []string{"image", "--format", "cyclonedx", "--quiet", img}
	}},
}

// DetectBackend returns the first available SBOM backend on PATH, or ok=false
// with the list of names it looked for.
func DetectBackend() (Backend, bool) {
	for _, b := range backends {
		if _, err := exec.LookPath(b.Bin); err == nil {
			return b, true
		}
	}
	return Backend{}, false
}

// BackendNames lists the backends opsforge can use, for error messages.
func BackendNames() []string {
	names := make([]string, len(backends))
	for i, b := range backends {
		names[i] = b.Bin
	}
	return names
}

// cyclonedxDoc is the subset of a CycloneDX SBOM opsforge reads back.
type cyclonedxDoc struct {
	Components []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		PURL    string `json:"purl"`
	} `json:"components"`
}

// GenerateSBOM runs the backend against an image and returns the OSV-mappable
// components it found (deduped by purl). Components without an OSV-mapped purl
// or a usable version are skipped.
func GenerateSBOM(ctx context.Context, b Backend, image string) ([]Component, error) {
	cmd := exec.CommandContext(ctx, b.Bin, b.Args(image)...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s failed on %q: %w", b.Bin, image, err)
	}
	return parseCycloneDX(out)
}

// parseCycloneDX extracts OSV-mappable components from a CycloneDX SBOM.
func parseCycloneDX(data []byte) ([]Component, error) {
	var doc cyclonedxDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing CycloneDX SBOM: %w", err)
	}
	seen := map[string]bool{}
	var comps []Component
	for _, c := range doc.Components {
		if c.PURL == "" || seen[c.PURL] {
			continue
		}
		seen[c.PURL] = true
		comp, ok := componentFromPURL(c.PURL, c.Name, c.Version)
		if !ok {
			continue
		}
		comps = append(comps, comp)
	}
	return comps, nil
}
