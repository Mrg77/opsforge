// Package guardlog keeps an append-only local audit trail of the guard
// decisions opsforge makes — the one thing a raw shell history can't tell a
// DevOps engineer: "what did I run against a production context this week, and
// did the guard let it through?".
//
// The guard already evaluates every destructive command against the current
// kube/cloud/tf context; until now it decided and forgot. This records each
// non-allow decision (warn/confirm/deny) to a JSONL file under the XDG state
// dir, so `opsforge guard log` can replay it, filtered by context or time.
//
// It is deliberately: append-only (never rewrites history), local (no server),
// and best-effort (a logging failure must never break the shell — the guard's
// job is to gate the command, not to guarantee the journal).
package guardlog

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Entry is one recorded guard decision.
type Entry struct {
	Time      time.Time `json:"time"`
	Command   string    `json:"command"`
	Context   string    `json:"context"`             // the kube/cloud/tf context at the time
	Action    string    `json:"action"`              // warn | confirm | deny
	Rule      string    `json:"rule,omitempty"`      // matching rule name
	Message   string    `json:"message,omitempty"`   // the guard message shown
	Confirmed *bool     `json:"confirmed,omitempty"` // for confirm: did the user type yes? (nil if unknown)
}

// Path returns the audit log file location: $XDG_STATE_HOME/opsforge/
// guard-audit.jsonl, or ~/.local/state/opsforge/guard-audit.jsonl.
func Path() string {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "opsforge", "guard-audit.jsonl")
}

// Append records a decision. Best-effort: any error is swallowed so a full
// disk or a read-only home can never wedge the shell. allow decisions are not
// logged (they're just the normal flow; `opsforge history` already covers
// what you ran).
func Append(e Entry) {
	if e.Action == "" || e.Action == "allow" {
		return
	}
	p := Path()
	if p == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	f.Write(append(b, '\n'))
}

// Filter narrows which entries Read returns.
type Filter struct {
	Since       time.Time // zero = no lower bound
	ContextSub  string    // case-insensitive substring the context must contain ("" = any)
	ProdOnly    bool      // only entries whose context looks like production
	ActionEqual string    // only this action ("" = any)
}

// Read returns the logged entries matching the filter, oldest first. A missing
// log is not an error — it just means nothing has been guarded yet.
func Read(f Filter) ([]Entry, error) {
	p := Path()
	if p == "" {
		return nil, nil
	}
	file, err := os.Open(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var out []Entry
	sc := bufio.NewScanner(file)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e Entry
		if json.Unmarshal([]byte(line), &e) != nil {
			continue // skip a corrupt line rather than fail the whole read
		}
		if f.matches(e) {
			out = append(out, e)
		}
	}
	return out, sc.Err()
}

func (f Filter) matches(e Entry) bool {
	if !f.Since.IsZero() && e.Time.Before(f.Since) {
		return false
	}
	if f.ActionEqual != "" && e.Action != f.ActionEqual {
		return false
	}
	if f.ContextSub != "" && !strings.Contains(strings.ToLower(e.Context), strings.ToLower(f.ContextSub)) {
		return false
	}
	if f.ProdOnly && !looksProd(e.Context) {
		return false
	}
	return true
}

// looksProd mirrors the default policy's fail-safe view: any context
// containing "prod" is treated as production-like.
func looksProd(ctx string) bool {
	return strings.Contains(strings.ToLower(ctx), "prod")
}
