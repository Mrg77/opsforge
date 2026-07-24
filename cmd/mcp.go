package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	forgemcp "github.com/Mrg77/opsforge/internal/mcp"
	"github.com/Mrg77/opsforge/internal/sbom"
	"github.com/Mrg77/opsforge/internal/shellcfg"
)

// mcp exposes opsforge's read-only capabilities as Model Context Protocol
// tools over stdio, so an AI agent (Claude Desktop/Code, Cursor, …) can
// interrogate the workstation in natural language: "what's installed?",
// "any CVEs I should worry about?", "would running this command trip a
// guard on prod?".
//
// SECURITY — READ-ONLY BY DESIGN. Only inspection tools are exposed. There
// is deliberately NO install / upgrade / apply / self-update / guard-init
// tool: letting an autonomous agent mutate the machine over MCP is a class
// of foot-gun opsforge does not want to hand out. Every tool below is
// annotated ReadOnlyHint:true so a well-behaved client can surface that it
// cannot change anything. The CVE scan is the only tool that touches the
// network (OSV.dev); it runs under a bounded timeout.
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run a read-only MCP server so AI agents can inspect this workstation",
	Long: `Start a Model Context Protocol (MCP) server on stdio that exposes
opsforge's READ-ONLY capabilities as tools an AI agent can call:

  list_installed_tools   — installed tools with version, category, outdated
  audit_vulnerabilities  — CVEs in installed tools (via OSV.dev)
  generate_sbom          — CycloneDX SBOM of installed tools (optional CVEs)
  workstation_status     — a one-glance summary (tools, updates, CVEs, shell)
  check_guard_policy     — what the shell guard would decide for a command

The server speaks newline-delimited JSON-RPC over stdin/stdout and blocks
until the client disconnects or it receives SIGINT/SIGTERM. It is meant to
be launched BY an agent, not run by hand.

No mutating tool is exposed on purpose (no install/upgrade/apply): an agent
can inspect this machine but never change it.

Register it with Claude Code:

  claude mcp add opsforge -- opsforge mcp

or add it to a client config (Claude Desktop, Cursor, …) as the command
"opsforge" with the argument "mcp".`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMCPServer(cmd.Context())
	},
}

// runMCPServer builds the MCP server, registers the read-only tools, and
// serves stdio until the context is cancelled (Ctrl-C / SIGTERM) or the
// client closes stdin.
func runMCPServer(parent context.Context) error {
	// Cancel the server on interrupt so a client detaching (or a Ctrl-C when
	// run by hand) tears the session down cleanly rather than hanging on a
	// half-open stdio pipe.
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := sdk.NewServer(&sdk.Implementation{
		Name:    "opsforge",
		Title:   "opsforge workstation",
		Version: version,
	}, nil)

	registerTools(server)

	// StdioTransport is the standard MCP transport a launching agent speaks;
	// Run blocks until the peer disconnects or ctx is cancelled.
	if err := server.Run(ctx, &sdk.StdioTransport{}); err != nil && ctx.Err() == nil {
		return fmt.Errorf("mcp server: %w", err)
	}
	return nil
}

// readOnly is a shared annotation marking every opsforge MCP tool as
// non-mutating, so clients can treat them as safe to call without an
// approval prompt.
func readOnly(title string) *sdk.ToolAnnotations {
	return &sdk.ToolAnnotations{Title: title, ReadOnlyHint: true}
}

// osvTimeout bounds the network scan against OSV.dev so a slow or
// unreachable database can't hang an agent's tool call indefinitely.
const osvTimeout = 40 * time.Second

// noArgs is the empty input for tools that take no parameters. The SDK
// requires a struct (or map) so it can infer an object input schema.
type noArgs struct{}

// registerTools wires the five read-only tools onto the server. Each
// handler reuses opsforge's existing business functions (detect, audit,
// sbom, shellcfg) and defers the shaping to internal/mcp so the payloads
// stay unit-testable.
func registerTools(server *sdk.Server) {
	// 1. list_installed_tools -------------------------------------------------
	sdk.AddTool(server, &sdk.Tool{
		Name:        "list_installed_tools",
		Description: "List the DevOps tools installed on this workstation, with each tool's version, catalog category, and whether an update is available.",
		Annotations: readOnly("List installed tools"),
	}, func(ctx context.Context, _ *sdk.CallToolRequest, _ noArgs) (*sdk.CallToolResult, forgemcp.InstalledToolsResult, error) {
		cat, err := catalog.Load()
		if err != nil {
			return nil, forgemcp.InstalledToolsResult{}, err
		}
		statuses := detect.AllWithOutdated(cat.Tools())
		return nil, forgemcp.BuildInstalledTools(cat, statuses), nil
	})

	// 2. audit_vulnerabilities ------------------------------------------------
	sdk.AddTool(server, &sdk.Tool{
		Name:        "audit_vulnerabilities",
		Description: "Scan the installed tools for known CVEs by cross-referencing their versions against the OSV.dev database. Returns, per vulnerable tool, its top severity and the list of CVEs (id, severity, fixed-in version). Requires network access to OSV.dev.",
		Annotations: readOnly("Audit tools for CVEs"),
	}, func(ctx context.Context, _ *sdk.CallToolRequest, _ noArgs) (*sdk.CallToolResult, forgemcp.AuditResult, error) {
		cat, err := catalog.Load()
		if err != nil {
			return nil, forgemcp.AuditResult{}, err
		}
		scanCtx, cancel := context.WithTimeout(ctx, osvTimeout)
		defer cancel()
		findings := audit.ScanTools(scanCtx, CollectOSVTargets(cat))
		return nil, forgemcp.BuildAudit(findings), nil
	})

	// 3. generate_sbom --------------------------------------------------------
	type sbomArgs struct {
		WithCVEs bool `json:"with_cves" jsonschema:"also cross-reference OSV.dev and embed the known CVEs as CycloneDX vulnerabilities (requires network access)"`
	}
	sdk.AddTool(server, &sdk.Tool{
		Name:        "generate_sbom",
		Description: "Produce a CycloneDX 1.6 Software Bill of Materials of the installed tools — each tool a component with its version and, when mapped, a PURL. Set with_cves to embed known CVEs from OSV.dev.",
		Annotations: readOnly("Generate SBOM"),
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in sbomArgs) (*sdk.CallToolResult, sbom.Doc, error) {
		cat, err := catalog.Load()
		if err != nil {
			return nil, sbom.Doc{}, err
		}
		statuses := detect.All(cat.Tools())

		// Optionally scan installed, OSV-mapped tools for CVEs and index the
		// findings by tool name — mirrors `opsforge sbom --audit`.
		vulnsByTool := map[string][]audit.Vuln{}
		if in.WithCVEs {
			scanCtx, cancel := context.WithTimeout(ctx, osvTimeout)
			defer cancel()
			for _, f := range audit.ScanTools(scanCtx, CollectOSVTargets(cat)) {
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
		doc := sbom.Build(inputs, time.Now().UTC().Format(time.RFC3339))
		return nil, doc, nil
	})

	// 4. workstation_status ---------------------------------------------------
	type statusArgs struct {
		IncludeCVEs bool `json:"include_cves" jsonschema:"also scan OSV.dev to count vulnerable tools (slower, needs network); when false the summary is instant and offline"`
	}
	sdk.AddTool(server, &sdk.Tool{
		Name:        "workstation_status",
		Description: "A one-glance summary of the workstation: how many catalog tools are installed, how many have updates, whether the opsforge shell layer is active, the current kube/cloud/terraform context, and (when include_cves is set) how many tools carry CVEs.",
		Annotations: readOnly("Workstation status"),
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in statusArgs) (*sdk.CallToolResult, forgemcp.StatusResult, error) {
		cat, err := catalog.Load()
		if err != nil {
			return nil, forgemcp.StatusResult{}, err
		}
		statuses := detect.AllWithOutdated(cat.Tools())

		var findings []audit.Finding
		if in.IncludeCVEs {
			scanCtx, cancel := context.WithTimeout(ctx, osvTimeout)
			defer cancel()
			findings = audit.ScanTools(scanCtx, CollectOSVTargets(cat))
		}
		// CurrentContext reads kube/cloud/tf passively — it never execs
		// kubectl, so this cannot trigger an OIDC login.
		result := forgemcp.BuildStatus(statuses, len(cat.Tools()), findings,
			shellcfg.InstalledInZshrc(), shellcfg.CurrentContext())
		return nil, result, nil
	})

	// 5. check_guard_policy ---------------------------------------------------
	type guardArgs struct {
		Command string `json:"command" jsonschema:"the shell command to evaluate, e.g. 'kubectl delete deploy api'"`
		Context string `json:"context" jsonschema:"optional context to match against (kube/cloud/tf), e.g. 'prod'; when empty the current workstation context is read passively"`
	}
	sdk.AddTool(server, &sdk.Tool{
		Name:        "check_guard_policy",
		Description: "Evaluate a shell command against opsforge's guard policy (policy-as-code) and return the decision: allow, warn, confirm, or deny — plus the matched rule and message. Read-only: it never runs the command, and reading the context never invokes kubectl/gcloud.",
		Annotations: readOnly("Check guard policy"),
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in guardArgs) (*sdk.CallToolResult, forgemcp.GuardResult, error) {
		policy, _, err := shellcfg.LoadPolicy()
		if err != nil {
			return nil, forgemcp.GuardResult{}, err
		}
		context := in.Context
		if context == "" {
			context = shellcfg.CurrentContext()
		}
		return nil, forgemcp.BuildGuard(policy, in.Command, context), nil
	})
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
