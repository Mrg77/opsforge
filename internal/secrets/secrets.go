// Package secrets scans the places where credentials leak on a DevOps
// workstation — shell history, shell rc files, local .env files — and
// reports what it finds with masked values. This is the #1 documented
// pain of 2026: a `kubectl create secret --from-literal=pw=...` or an
// `export GITHUB_TOKEN=...` quietly persists the credential in
// ~/.zsh_history forever.
package secrets

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Severity mirrors the audit package's coarse scale, locally defined to
// keep this package dependency-free.
type Severity int

const (
	SevInfo Severity = iota
	SevWarning
	SevCritical
)

func (s Severity) String() string {
	switch s {
	case SevCritical:
		return "CRITICAL"
	case SevWarning:
		return "WARNING"
	default:
		return "INFO"
	}
}

// Rule is one leak pattern.
type Rule struct {
	ID       string
	Desc     string
	Re       *regexp.Regexp
	Severity Severity
}

// Rules covers well-known token formats plus the generic "assigning a
// secret-looking variable on the command line" cases.
var Rules = []Rule{
	{"aws-access-key", "AWS access key ID", regexp.MustCompile(`\b(AKIA|ASIA)[0-9A-Z]{16}\b`), SevCritical},
	{"github-token", "GitHub token", regexp.MustCompile(`\b(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9]{36,}\b|\bgithub_pat_[A-Za-z0-9_]{22,}\b`), SevCritical},
	{"gitlab-token", "GitLab personal access token", regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{20,}\b`), SevCritical},
	{"slack-token", "Slack token", regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{10,}\b`), SevCritical},
	{"private-key", "Private key material", regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`), SevCritical},
	{"jwt", "JSON Web Token", regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`), SevWarning},
	{"kubectl-literal", "kubectl secret passed with --from-literal", regexp.MustCompile(`--from-literal=[^= ]+=\S+`), SevCritical},
	{"env-export-secret", "Secret exported on the command line", regexp.MustCompile(`(?i)\bexport\s+[A-Z0-9_]*(TOKEN|SECRET|PASSWORD|PASSWD|API_?KEY|CREDENTIALS?)[A-Z0-9_]*=\S+`), SevWarning},
	{"curl-auth", "Credential passed to curl", regexp.MustCompile(`(?i)curl\b[^\n]*(-u |--user |Authorization: Bearer )\S+`), SevWarning},
	{"docker-password", "docker login with inline password", regexp.MustCompile(`(?i)docker login[^\n]*(-p |--password )\S+`), SevCritical},
}

// Finding is one detected leak.
type Finding struct {
	Rule    Rule
	Source  string // file the match came from
	Line    int    // 1-based line number
	Excerpt string // the matched text, masked
}

// Mask hides the middle of a secret-looking string, keeping just enough
// to recognize it: "ghp_abcdefgh…(39 chars)".
func Mask(s string) string {
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return fmt.Sprintf("%s…(%d chars)", s[:8], len(s))
}

// zshHistPrefix strips zsh extended-history metadata (": 1690000000:0;cmd").
var zshHistPrefix = regexp.MustCompile(`^: \d+:\d+;`)

// ScanReader scans one stream line by line.
func ScanReader(r *bufio.Scanner, source string) []Finding {
	var out []Finding
	// history lines can be long (pasted blobs)
	r.Buffer(make([]byte, 0, 256*1024), 1024*1024)
	line := 0
	for r.Scan() {
		line++
		text := zshHistPrefix.ReplaceAllString(r.Text(), "")
		for _, rule := range Rules {
			if m := rule.Re.FindString(text); m != "" {
				out = append(out, Finding{
					Rule:    rule,
					Source:  source,
					Line:    line,
					Excerpt: Mask(m),
				})
			}
		}
	}
	return out
}

// ScanFile scans a single file; missing files are silently skipped.
func ScanFile(path string) []Finding {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	return ScanReader(bufio.NewScanner(f), path)
}

// DefaultTargets are the workstation locations where credentials
// habitually leak: shell history, shell rc files, and local env files in
// the current directory.
func DefaultTargets() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	targets := []string{
		filepath.Join(home, ".zsh_history"),
		filepath.Join(home, ".bash_history"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".zshenv"),
		filepath.Join(home, ".zprofile"),
		filepath.Join(home, ".bashrc"),
	}
	// .env-style files in the current directory (the classic accident).
	if cwd, err := os.Getwd(); err == nil {
		matches, _ := filepath.Glob(filepath.Join(cwd, ".env*"))
		targets = append(targets, matches...)
	}
	return targets
}

// ScanWorkstation scans all default targets.
func ScanWorkstation() []Finding {
	var out []Finding
	for _, t := range DefaultTargets() {
		out = append(out, ScanFile(t)...)
	}
	return out
}
