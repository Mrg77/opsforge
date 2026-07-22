package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/installer"
	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/secrets"
	"github.com/Mrg77/opsforge/internal/shellcfg"
	"github.com/Mrg77/opsforge/internal/ui"
	"github.com/Mrg77/opsforge/internal/versions"
)

// ansiRe strips SGR escape sequences so JSON `detail` fields are plain
// text (several checks pass already-styled detail strings to line()).
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

// doctorSkipSecurity disables the network CVE scan (for --quick / offline).
var doctorSkipSecurity bool

// plural returns "N thing" or "N things".
func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// checkResult is one health check outcome.
type checkResult int

const (
	pass checkResult = iota
	warn
	failed
)

// doctorCheck is one health-check result in machine-readable form.
type doctorCheck struct {
	Section string `json:"section"`
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass" | "warn" | "fail"
	Detail  string `json:"detail,omitempty"`
	Fix     string `json:"fix,omitempty"`
}

// doctorReport accumulates checks so we can score, summarize and emit JSON.
type doctorReport struct {
	pass, warn, fail int
	section          string
	checks           []doctorCheck
}

func (r *doctorReport) line(res checkResult, label, detail, fix string) {
	var mark, status string
	switch res {
	case pass:
		mark, status = ui.OKMark(), "pass"
		r.pass++
	case warn:
		mark, status = ui.WarnMark(), "warn"
		r.warn++
	default:
		mark, status = ui.ErrMark(), "fail"
		r.fail++
	}
	r.checks = append(r.checks, doctorCheck{
		Section: r.section, Name: label, Status: status,
		Detail: stripANSI(detail), Fix: fix,
	})
	if output.JSON {
		return
	}
	line := fmt.Sprintf("  %s %s", mark, label)
	if detail != "" {
		line += "  " + ui.Dim.Render(detail)
	}
	fmt.Println(line)
	if fix != "" && res != pass {
		fmt.Printf("      %s %s\n", ui.Dim.Render(ui.MarkArrow), ui.Dim.Render(fix))
	}
}

// section records the current section and prints its header unless we're
// emitting JSON. Commands call this instead of ui.Section directly so the
// JSON path stays quiet and each check knows its section.
func (r *doctorReport) beginSection(name string) {
	r.section = name
	if !output.JSON {
		fmt.Println(ui.Section(name))
	}
}

// jsonReport is the machine-readable shape of a full doctor run.
func (r *doctorReport) jsonReport() any {
	status := "healthy"
	switch {
	case r.fail > 0:
		status = "failing"
	case r.warn > 0:
		status = "warnings"
	}
	return struct {
		Status string        `json:"status"` // healthy | warnings | failing
		Passed int           `json:"passed"`
		Warned int           `json:"warnings"`
		Failed int           `json:"failed"`
		Checks []doctorCheck `json:"checks"`
	}{status, r.pass, r.warn, r.fail, r.checks}
}

func boolRes(ok bool) checkResult {
	if ok {
		return pass
	}
	return failed
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Full health check of your DevOps workstation",
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		r := &doctorReport{}

		if !output.JSON {
			fmt.Println(ui.Header("opsforge doctor", "a full health check of your DevOps workstation"))
			fmt.Println()
		}

		// --- System ---------------------------------------------------------
		r.beginSection("System")
		brew := installer.BrewAvailable()
		r.line(boolRes(brew), "Homebrew", brewDetail(brew),
			"install from https://brew.sh (opsforge can also install via GitHub releases)")
		inPath := strings.Contains(os.Getenv("PATH"), "/opt/homebrew/bin") ||
			strings.Contains(os.Getenv("PATH"), "/usr/local/bin")
		r.line(boolRes(inPath), "Homebrew bin on PATH", "",
			"add `eval \"$(/opt/homebrew/bin/brew shellenv)\"` to ~/.zprofile")
		localBin := strings.Contains(os.Getenv("PATH"), ".local/bin")
		r.line(boolRes(localBin), "~/.local/bin on PATH", ui.Dim.Render("(GitHub-installed tools land here)"),
			"add `export PATH=\"$HOME/.local/bin:$PATH\"` to ~/.zshrc")
		if mgr := versions.Detect(); mgr != versions.None {
			r.line(pass, "Version manager", string(mgr)+" — `opsforge use <tool>@<ver>` works", "")
		} else {
			// Optional feature — a note, not a warning.
			r.line(pass, "Version manager",
				ui.Dim.Render("not installed (optional — `opsforge install mise` enables `opsforge use`)"), "")
		}
		doctorBlank()

		// --- Shell environment ---------------------------------------------
		r.beginSection("Shell environment")
		shellOn := shellcfg.InstalledInZshrc()
		r.line(boolRes(shellOn), "opsforge shell layer", shellStateDetail(shellOn),
			"run `opsforge shell install`")
		if complDir, e := shellcfg.CompletionsDir(); e == nil {
			entries, _ := os.ReadDir(complDir)
			res := pass
			if len(entries) == 0 {
				res = warn
			}
			r.line(res, "Cached completions", fmt.Sprintf("%d tool(s)", len(entries)),
				"run `opsforge shell sync`")
		}
		for _, p := range shellcfg.InteractivePluginStatus() {
			res := pass
			if !p.Installed {
				res = warn
			}
			r.line(res, p.Name, "", "installed by `opsforge shell install`")
		}
		doctorBlank()

		// --- Toolbox --------------------------------------------------------
		r.beginSection("Toolbox")
		statuses := detect.AllWithOutdated(cat.Tools())
		installed := 0
		var outdatedTools []string
		var broken []string
		for _, t := range cat.Tools() {
			s := statuses[t.Name]
			if !s.Installed {
				continue
			}
			installed++
			if s.Outdated {
				v := audit.NormalizeVersion(s.Version)
				if v != "" {
					outdatedTools = append(outdatedTools, fmt.Sprintf("%s (%s)", t.Name, v))
				} else {
					outdatedTools = append(outdatedTools, t.Name)
				}
			}
			if s.Version == "" {
				broken = append(broken, t.Name)
			}
		}
		r.line(pass, "Installed tools",
			fmt.Sprintf("%d of %d catalog tools", installed, len(cat.Tools())), "")
		if len(outdatedTools) > 0 {
			r.line(warn, "Updates available",
				fmt.Sprintf("%d tool(s): %s", len(outdatedTools), strings.Join(outdatedTools, ", ")),
				"run `opsforge upgrade -u` to update them all")
		} else {
			r.line(pass, "Updates", "everything up to date", "")
		}
		if len(broken) > 0 {
			// krew and similar report no --version; it's cosmetic, not a fault.
			r.line(pass, "Version probe",
				ui.Dim.Render(fmt.Sprintf("%s report no version (cosmetic): %s",
					plural(len(broken), "tool"), strings.Join(broken, ", "))), "")
		}
		doctorBlank()

		// --- Security -------------------------------------------------------
		r.beginSection("Security")
		checkCVEs(r, cat)
		checkSecrets(r)
		doctorBlank()

		// --- Summary --------------------------------------------------------
		if output.JSON {
			if err := output.Emit(r.jsonReport()); err != nil {
				return err
			}
		} else {
			printDoctorSummary(r)
		}
		if r.fail > 0 {
			return fmt.Errorf("%d check(s) failed", r.fail)
		}
		return nil
	},
}

// doctorBlank prints a blank separator line in human mode only.
func doctorBlank() {
	if !output.JSON {
		fmt.Println()
	}
}

// checkCVEs scans installed tools against OSV.dev and reports known
// vulnerabilities as a doctor check. It's network-bound, so a failed or
// slow query degrades to a note rather than failing the whole doctor.
func checkCVEs(r *doctorReport, cat *catalog.Catalog) {
	if doctorSkipSecurity {
		r.line(pass, "Known CVEs", ui.Dim.Render("skipped (--skip-security)"), "")
		return
	}
	targets := CollectOSVTargets(cat)
	if len(targets) == 0 {
		r.line(pass, "Known CVEs", ui.Dim.Render("no auditable tool installed"), "")
		return
	}

	// Hint on stderr that we're waiting on the network, then clear it so it
	// leaves no residue before the result line prints on stdout.
	fmt.Fprint(os.Stderr, ui.Dim.Render("  scanning OSV.dev for CVEs…"))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	findings := audit.ScanTools(ctx, targets)
	fmt.Fprint(os.Stderr, "\r\033[K")

	// Collect the tools that actually carry vulnerabilities, most severe first.
	var vuln []audit.Finding
	for _, f := range findings {
		if len(f.Vulns) > 0 {
			vuln = append(vuln, f)
		}
	}
	if len(vuln) == 0 {
		r.line(pass, "Known CVEs",
			fmt.Sprintf("%d tool(s) checked, none vulnerable", len(targets)), "")
		return
	}
	sort.Slice(vuln, func(a, b int) bool {
		return vuln[a].TopSeverity() > vuln[b].TopSeverity()
	})

	// Any CRITICAL/HIGH is a failure; only MEDIUM/LOW is a warning.
	res := warn
	for _, f := range vuln {
		if f.TopSeverity() >= audit.SevHigh {
			res = failed
			break
		}
	}
	var names []string
	for _, f := range vuln {
		names = append(names, fmt.Sprintf("%s (%s)", f.Tool, f.TopSeverity()))
	}
	r.line(res, "Known CVEs",
		fmt.Sprintf("%s affected: %s", plural(len(vuln), "tool"), strings.Join(names, ", ")),
		"run `opsforge audit` for details, then `opsforge upgrade` the affected tools")
}

// checkSecrets scans the workstation for leaked credentials and reports
// them as a doctor check (any critical leak is a failure).
func checkSecrets(r *doctorReport) {
	findings := secrets.ScanWorkstation()
	if len(findings) == 0 {
		r.line(pass, "Leaked secrets", "none in history, rc files or local .env", "")
		return
	}
	critical := 0
	for _, f := range findings {
		if f.Rule.Severity == secrets.SevCritical {
			critical++
		}
	}
	res := warn
	if critical > 0 {
		res = failed
	}
	detail := fmt.Sprintf("%s found", plural(len(findings), "potential leak"))
	if critical > 0 {
		detail += fmt.Sprintf(" (%d critical)", critical)
	}
	r.line(res, "Leaked secrets", detail,
		"run `opsforge audit --secrets`, then rotate and remove the exposed credentials")
}

func brewDetail(ok bool) string {
	if ok {
		return "available"
	}
	return "not found"
}

func shellStateDetail(on bool) string {
	if on {
		return "active in ~/.zshrc"
	}
	return "not installed"
}

func printDoctorSummary(r *doctorReport) {
	total := r.pass + r.warn + r.fail
	fmt.Println(ui.Section("Health"))
	fmt.Printf("  %s  %s  %s\n",
		ui.OK.Render(fmt.Sprintf("%s %d passed", ui.MarkOK, r.pass)),
		ui.Warn.Render(fmt.Sprintf("%s %d warnings", ui.MarkWarn, r.warn)),
		ui.Err.Render(fmt.Sprintf("%s %d failed", ui.MarkErr, r.fail)))
	fmt.Printf("  %s\n", ui.Bar(r.pass, total, 24))
	fmt.Println()
	switch {
	case r.fail > 0:
		fmt.Println(ui.Err.Render("Some checks failed — address the → hints above."))
	case r.warn > 0:
		fmt.Println(ui.Warn.Render("Healthy, with a few optional improvements above."))
	default:
		fmt.Println(ui.OKBold.Render("All good. Happy shipping! 🔥"))
	}
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorSkipSecurity, "skip-security", false,
		"skip the online CVE scan (offline / faster)")
	rootCmd.AddCommand(doctorCmd)
}
