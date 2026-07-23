// Package history reads the user's shell history and filters it by tool
// family — "show me my recent kubectl/helm commands", "my git commands",
// and so on. It parses zsh history passively (never executes anything)
// and is the data source behind `opsforge history`.
package history

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Mrg77/opsforge/internal/families"
)

// Family and the built-in groupings live in internal/families, the single
// source of truth shared with the guard engine. These aliases keep the
// history package's API stable while delegating the data.
type Family = families.Family

// Families are the built-in tool groupings (from internal/families).
var Families = families.All

// FamilyByKey returns the built-in family with the given key, or false.
func FamilyByKey(key string) (Family, bool) {
	return families.ByKey(key)
}

// Entry is one history command with how many times it appears.
type Entry struct {
	Command string `json:"command"`
	Count   int    `json:"count"`
	// LastIndex is the position of the most recent occurrence in the raw
	// history (higher = more recent); used for recency ordering.
	LastIndex int `json:"-"`
}

// HistoryFile returns the shell history file to read: $HISTFILE if set,
// else ~/.zsh_history, else ~/.bash_history.
func HistoryFile() string {
	if h := os.Getenv("HISTFILE"); h != "" {
		return h
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	for _, name := range []string{".zsh_history", ".bash_history"} {
		p := filepath.Join(home, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// zshExtended matches the ": <epoch>:<elapsed>;<command>" line prefix that
// zsh writes when EXTENDED_HISTORY is on. We keep only the command part.
var zshExtended = regexp.MustCompile(`^: \d+:\d+;`)

// firstWord returns the command's leading executable, stripping any
// leading environment assignments ("FOO=bar cmd") and `sudo`.
func firstWord(cmd string) string {
	for _, w := range strings.Fields(cmd) {
		if strings.Contains(w, "=") { // env assignment prefix
			continue
		}
		if w == "sudo" || w == "command" || w == "noglob" {
			continue
		}
		return w
	}
	return ""
}

// Match reports whether a command line belongs to a family (by its leading
// executable). bins is the family's set of executables.
func Match(cmd string, bins map[string]bool) bool {
	return bins[firstWord(cmd)]
}

// Query reads the history file and returns the distinct commands whose
// leading executable is in `bins`, most-recent first, capped at `limit`
// (0 = no cap). Duplicate commands are collapsed with a run count, keyed
// by their most recent occurrence.
func Query(path string, bins []string, limit int) ([]Entry, error) {
	set := make(map[string]bool, len(bins))
	for _, b := range bins {
		set[b] = true
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	byCmd := map[string]*Entry{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	idx := 0
	for sc.Scan() {
		line := zshExtended.ReplaceAllString(sc.Text(), "")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !Match(line, set) {
			continue
		}
		idx++
		if e, ok := byCmd[line]; ok {
			e.Count++
			e.LastIndex = idx
		} else {
			byCmd[line] = &Entry{Command: line, Count: 1, LastIndex: idx}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(byCmd))
	for _, e := range byCmd {
		entries = append(entries, *e)
	}
	// Most recent first (highest LastIndex).
	sort.Slice(entries, func(a, b int) bool {
		return entries[a].LastIndex > entries[b].LastIndex
	})
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}
