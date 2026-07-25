package credscan

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// sortFindings orders findings most-severe first, then by provider, then by
// path — a stable, scannable order for both the human report and JSON.
func sortFindings(fs []Finding) {
	sort.SliceStable(fs, func(a, b int) bool {
		if fs[a].Severity != fs[b].Severity {
			return fs[a].Severity > fs[b].Severity
		}
		if fs[a].Provider != fs[b].Provider {
			return fs[a].Provider < fs[b].Provider
		}
		return fs[a].Path < fs[b].Path
	})
}

// home is the user's home directory, resolved once. Scanners build paths from
// it. On the rare error it is "", and expand() leaves ~ untouched so a path
// simply won't exist — the scan degrades quietly rather than failing.
var home = func() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}()

// expand turns a leading ~ into the home directory. It only touches a leading
// "~/" (or a bare "~"); a ~ elsewhere in the path is left alone.
func expand(p string) string {
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") && home != "" {
		return filepath.Join(home, p[2:])
	}
	return p
}

// readFile reads a file for inspection, returning (nil, false) when it is
// absent or unreadable — a missing credential file is the normal case, not an
// error. The path is expanded first.
func readFile(path string) ([]byte, bool) {
	b, err := os.ReadFile(expand(path))
	if err != nil {
		return nil, false
	}
	return b, true
}

// statFile returns the FileInfo for an existing path (expanded), or (nil,
// false) when it is absent/unreadable.
func statFile(path string) (os.FileInfo, bool) {
	fi, err := os.Stat(expand(path))
	if err != nil {
		return nil, false
	}
	return fi, true
}

// worldOrGroupReadable reports whether a file's permission bits let anyone
// other than the owner read it (group or other read/write/exec set). A private
// key or token file should be 0600 (or 0400); anything looser is a finding.
func worldOrGroupReadable(mode os.FileMode) bool {
	return mode.Perm()&0o077 != 0
}
