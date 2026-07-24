package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/sbom"
	"github.com/Mrg77/opsforge/internal/ui"
)

var (
	sbomWithAudit bool
	sbomSign      bool
	sbomSignOut   string
)

var sbomCmd = &cobra.Command{
	Use:   "sbom",
	Short: "Emit a CycloneDX SBOM of your installed tools (--audit adds CVEs)",
	Long: `Produce a CycloneDX 1.6 Software Bill of Materials of the DevOps tools
installed on this machine — each tool a component with its version and, when
the catalog maps it to a package ecosystem, a PURL.

With --audit, opsforge cross-references the OSV.dev database and embeds the
known CVEs as CycloneDX vulnerabilities, so the SBOM is CVE-correlated out
of the box (feed it to grype, trivy sbom, or a compliance pipeline).

With --sign, opsforge also writes a self-contained Sigstore bundle of the SBOM,
signed key-based with a local opsforge key (offline — no OIDC, no Rekor). It
proves the document's integrity and attribution to your key; verify it with
'cosign verify-blob --key ~/.config/opsforge/signing.pub --bundle …'.

  opsforge sbom            # SBOM to stdout (CycloneDX JSON)
  opsforge sbom --audit    # + embedded CVE findings
  opsforge sbom > bom.json # capture it
  opsforge sbom --sign > bom.json   # + a bom.sigstore.json signature bundle`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		statuses := detect.All(cat.Tools())

		// Optionally scan installed, OSV-mapped tools for CVEs and index the
		// findings by tool name so we can attach them to components.
		vulnsByTool := map[string][]audit.Vuln{}
		if sbomWithAudit {
			targets := CollectOSVTargets(cat)
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()
			for _, f := range audit.ScanTools(ctx, targets) {
				if len(f.Vulns) > 0 {
					vulnsByTool[f.Tool] = f.Vulns
				}
			}
		}

		var inputs []sbom.Input
		for _, t := range cat.Tools() {
			s := statuses[t.Name]
			if !s.Installed {
				continue
			}
			in := sbom.Input{
				Name:        t.Name,
				Version:     s.Version,
				Description: t.Description,
				Vulns:       vulnsByTool[t.Name],
			}
			if t.OSV != nil {
				in.Ecosystem = t.OSV.Ecosystem
				in.Package = t.OSV.Name
			}
			inputs = append(inputs, in)
		}

		// time.Now is unavailable to the pure builder; stamp here in RFC3339.
		doc := sbom.Build(inputs, time.Now().UTC().Format(time.RFC3339))

		// The SBOM itself is the machine artifact — always emit the JSON to
		// stdout. A short summary goes to stderr so `opsforge sbom > bom.json`
		// still gives feedback without polluting the document.
		b, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return err
		}
		// Emit the document with a trailing newline; sign the EXACT same bytes
		// so `cosign verify-blob <the-file>` matches what landed on disk/stdout.
		docBytes := append(b, '\n')
		os.Stdout.Write(docBytes)
		fmt.Fprintln(os.Stderr, ui.Dim.Render("  "+doc.Summary()))

		if sbomSign {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			return signArtifact(ctx, docBytes, sbomSignOut, "SBOM")
		}
		return nil
	},
}

func init() {
	sbomCmd.Flags().BoolVar(&sbomWithAudit, "audit", false,
		"cross-reference OSV.dev and embed known CVEs in the SBOM")
	sbomCmd.Flags().BoolVar(&sbomSign, "sign", false,
		"also write a Sigstore signature bundle (key-based, offline)")
	sbomCmd.Flags().StringVar(&sbomSignOut, "sign-out", "bom.sigstore.json",
		"path for the --sign Sigstore bundle")
	rootCmd.AddCommand(sbomCmd)
}
