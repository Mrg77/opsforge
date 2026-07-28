package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/i18n"
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
	Short: i18n.T("doctor.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		r := &doctorReport{}

		if !output.JSON {
			fmt.Println(ui.Header("opsforge doctor", i18n.T("doctor.header.tag")))
			fmt.Println()
		}

		// --- System ---------------------------------------------------------
		r.beginSection(i18n.T("doctor.section.system"))
		brew := installer.BrewAvailable()
		r.line(boolRes(brew), i18n.T("doctor.system.homebrew"), brewDetail(brew),
			i18n.T("doctor.system.homebrew.fix"))
		inPath := strings.Contains(os.Getenv("PATH"), "/opt/homebrew/bin") ||
			strings.Contains(os.Getenv("PATH"), "/usr/local/bin")
		r.line(boolRes(inPath), i18n.T("doctor.system.brewpath"), "",
			i18n.T("doctor.system.brewpath.fix"))
		localBin := strings.Contains(os.Getenv("PATH"), ".local/bin")
		r.line(boolRes(localBin), i18n.T("doctor.system.localbin"), ui.Dim.Render(i18n.T("doctor.system.localbin.detail")),
			i18n.T("doctor.system.localbin.fix"))
		if mgr := versions.Detect(); mgr != versions.None {
			r.line(pass, i18n.T("doctor.system.vm"), i18n.T("doctor.system.vm.works", i18n.V("mgr", string(mgr))), "")
		} else {
			// Optional feature — a note, not a warning.
			r.line(pass, i18n.T("doctor.system.vm"),
				ui.Dim.Render(i18n.T("doctor.system.vm.none")), "")
		}
		doctorBlank()

		// --- Shell environment ---------------------------------------------
		r.beginSection(i18n.T("doctor.section.shell"))
		shellOn := shellcfg.InstalledInZshrc()
		r.line(boolRes(shellOn), i18n.T("doctor.shell.layer"), shellStateDetail(shellOn),
			i18n.T("doctor.shell.layer.fix"))
		if complDir, e := shellcfg.CompletionsDir(); e == nil {
			entries, _ := os.ReadDir(complDir)
			res := pass
			if len(entries) == 0 {
				res = warn
			}
			r.line(res, i18n.T("doctor.shell.completions"), i18n.T("doctor.shell.completions.n", i18n.V("n", strconv.Itoa(len(entries)))),
				i18n.T("doctor.shell.completions.fix"))
		}
		for _, p := range shellcfg.InteractivePluginStatus() {
			res := pass
			if !p.Installed {
				res = warn
			}
			r.line(res, p.Name, "", i18n.T("doctor.shell.plugin.fix"))
		}
		doctorBlank()

		// --- Toolbox --------------------------------------------------------
		r.beginSection(i18n.T("doctor.section.toolbox"))
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
		r.line(pass, i18n.T("doctor.toolbox.installed"),
			i18n.T("doctor.toolbox.installed.n", i18n.V("n", strconv.Itoa(installed), "total", strconv.Itoa(len(cat.Tools())))), "")
		if len(outdatedTools) > 0 {
			r.line(warn, i18n.T("doctor.toolbox.updates.avail"),
				i18n.T("doctor.toolbox.updates.detail", i18n.V("n", strconv.Itoa(len(outdatedTools)), "list", strings.Join(outdatedTools, ", "))),
				i18n.T("doctor.toolbox.updates.fix"))
		} else {
			r.line(pass, i18n.T("doctor.toolbox.updates.ok"), i18n.T("doctor.toolbox.updates.uptodate"), "")
		}
		if len(broken) > 0 {
			// krew and similar report no --version; it's cosmetic, not a fault.
			r.line(pass, i18n.T("doctor.toolbox.probe"),
				ui.Dim.Render(i18n.T("doctor.toolbox.probe.detail",
					i18n.V("n", strconv.Itoa(len(broken)), "list", strings.Join(broken, ", ")))), "")
		}
		doctorBlank()

		// --- Security -------------------------------------------------------
		r.beginSection(i18n.T("doctor.section.security"))
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
			return fmt.Errorf("%s", i18n.T("doctor.checks.failed", i18n.V("n", strconv.Itoa(r.fail))))
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
		r.line(pass, i18n.T("doctor.sec.cves"), ui.Dim.Render(i18n.T("doctor.sec.cves.skipped")), "")
		return
	}
	targets := CollectOSVTargets(cat)
	if len(targets) == 0 {
		r.line(pass, i18n.T("doctor.sec.cves"), ui.Dim.Render(i18n.T("doctor.sec.cves.noauditable")), "")
		return
	}

	// Hint on stderr that we're waiting on the network, then clear it so it
	// leaves no residue. Skipped in --json mode: a machine consumer wants no
	// terminal-control decoration, not even on stderr.
	showProgress := !output.JSON
	if showProgress {
		fmt.Fprint(os.Stderr, ui.Dim.Render(i18n.T("doctor.sec.cves.scanning")))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	findings := audit.ScanTools(ctx, targets)
	if showProgress {
		fmt.Fprint(os.Stderr, "\r\033[K")
	}

	// Collect the tools that actually carry vulnerabilities, most severe first.
	var vuln []audit.Finding
	for _, f := range findings {
		if len(f.Vulns) > 0 {
			vuln = append(vuln, f)
		}
	}
	if len(vuln) == 0 {
		r.line(pass, i18n.T("doctor.sec.cves"),
			i18n.T("doctor.sec.cves.clean", i18n.V("n", strconv.Itoa(len(targets)))), "")
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
	r.line(res, i18n.T("doctor.sec.cves"),
		i18n.T("doctor.sec.cves.affected", i18n.V("n", strconv.Itoa(len(vuln)), "list", strings.Join(names, ", "))),
		i18n.T("doctor.sec.cves.fix"))
}

// checkSecrets scans the workstation for leaked credentials and reports
// them as a doctor check (any critical leak is a failure).
func checkSecrets(r *doctorReport) {
	findings := secrets.ScanWorkstation()
	if len(findings) == 0 {
		r.line(pass, i18n.T("doctor.sec.secrets"), i18n.T("doctor.sec.secrets.clean"), "")
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
	detail := i18n.T("doctor.sec.secrets.found", i18n.V("n", strconv.Itoa(len(findings))))
	if critical > 0 {
		detail += i18n.T("doctor.sec.secrets.critical", i18n.V("n", strconv.Itoa(critical)))
	}
	r.line(res, i18n.T("doctor.sec.secrets"), detail,
		i18n.T("doctor.sec.secrets.fix"))
}

func brewDetail(ok bool) string {
	if ok {
		return i18n.T("doctor.system.homebrew.available")
	}
	return i18n.T("doctor.system.homebrew.notfound")
}

func shellStateDetail(on bool) string {
	if on {
		return i18n.T("doctor.shell.active")
	}
	return i18n.T("doctor.shell.inactive")
}

func printDoctorSummary(r *doctorReport) {
	total := r.pass + r.warn + r.fail
	fmt.Println(ui.Section(i18n.T("doctor.section.health")))
	fmt.Printf("  %s  %s  %s\n",
		ui.OK.Render(ui.MarkOK+" "+i18n.T("doctor.health.passed", i18n.V("n", strconv.Itoa(r.pass)))),
		ui.Warn.Render(ui.MarkWarn+" "+i18n.T("doctor.health.warnings", i18n.V("n", strconv.Itoa(r.warn)))),
		ui.Err.Render(ui.MarkErr+" "+i18n.T("doctor.health.failed", i18n.V("n", strconv.Itoa(r.fail)))))
	fmt.Printf("  %s\n", ui.Bar(r.pass, total, 24))
	fmt.Println()
	switch {
	case r.fail > 0:
		fmt.Println(ui.Err.Render(i18n.T("doctor.health.failing")))
	case r.warn > 0:
		fmt.Println(ui.Warn.Render(i18n.T("doctor.health.warned")))
	default:
		fmt.Println(ui.OKBold.Render(i18n.T("doctor.health.allgood")))
	}
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorSkipSecurity, "skip-security", false,
		i18n.T("doctor.flag.skipsecurity"))
	rootCmd.AddCommand(doctorCmd)
}
