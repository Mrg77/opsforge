package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/installer"
	"github.com/Mrg77/opsforge/internal/shellcfg"
	"github.com/Mrg77/opsforge/internal/ui"
	"github.com/Mrg77/opsforge/internal/versions"
)

// checkResult is one health check outcome.
type checkResult int

const (
	pass checkResult = iota
	warn
	failed
)

// doctorReport accumulates checks so we can score and summarize.
type doctorReport struct {
	pass, warn, fail int
}

func (r *doctorReport) line(res checkResult, label, detail, fix string) {
	var mark string
	switch res {
	case pass:
		mark = ui.OKMark()
		r.pass++
	case warn:
		mark = ui.WarnMark()
		r.warn++
	default:
		mark = ui.ErrMark()
		r.fail++
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

		fmt.Println(ui.Header("opsforge doctor", "a full health check of your DevOps workstation"))
		fmt.Println()

		// --- System ---------------------------------------------------------
		fmt.Println(ui.Section("System"))
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
			r.line(warn, "Version manager", "none", "install mise for `opsforge use terraform@1.5`")
		}
		fmt.Println()

		// --- Shell environment ---------------------------------------------
		fmt.Println(ui.Section("Shell environment"))
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
		fmt.Println()

		// --- Toolbox --------------------------------------------------------
		fmt.Println(ui.Section("Toolbox"))
		statuses := detect.AllWithOutdated(cat.Tools())
		installed, outdated, broken := 0, 0, []string{}
		for _, t := range cat.Tools() {
			s := statuses[t.Name]
			if !s.Installed {
				continue
			}
			installed++
			if s.Outdated {
				outdated++
			}
			if s.Version == "" {
				broken = append(broken, t.Name)
			}
		}
		r.line(pass, "Installed tools",
			fmt.Sprintf("%d of %d catalog tools", installed, len(cat.Tools())), "")
		if outdated > 0 {
			r.line(warn, "Updates available", fmt.Sprintf("%d tool(s)", outdated),
				"run `opsforge upgrade -u`")
		} else {
			r.line(pass, "Updates", "everything up to date", "")
		}
		if len(broken) > 0 {
			r.line(warn, "Version probe failed", strings.Join(broken, ", "),
				"these tools didn't report a version")
		}
		fmt.Println()

		// --- Summary --------------------------------------------------------
		printDoctorSummary(r)
		if r.fail > 0 {
			return fmt.Errorf("%d check(s) failed", r.fail)
		}
		return nil
	},
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
	rootCmd.AddCommand(doctorCmd)
}
