package guardlog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendSkipsAllowAndEmpty(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	Append(Entry{Action: "allow", Command: "kubectl get pods"})
	Append(Entry{Action: "", Command: "x"})
	// Nothing recorded → file absent or empty.
	entries, err := Read(Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("allow/empty actions must not be logged, got %d", len(entries))
	}
}

func TestAppendAndReadRoundTrip(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	now := time.Now().UTC().Truncate(time.Second)
	Append(Entry{Time: now, Action: "confirm", Command: "kubectl delete ns x", Context: "gke_prod", Rule: "r1", Message: "prod"})
	Append(Entry{Time: now, Action: "deny", Command: "kubectl delete ns y", Context: "staging"})

	got, err := Read(Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 entries, got %d", len(got))
	}
	if got[0].Action != "confirm" || got[0].Command != "kubectl delete ns x" || got[0].Context != "gke_prod" {
		t.Errorf("first entry wrong: %+v", got[0])
	}

	// Private perms on the file (it can hold command lines).
	if fi, err := os.Stat(Path()); err == nil && fi.Mode().Perm() != 0o600 {
		t.Errorf("audit log perms = %v, want 0600", fi.Mode().Perm())
	}
}

func TestReadFilters(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	old := time.Now().Add(-48 * time.Hour).UTC()
	recent := time.Now().Add(-1 * time.Hour).UTC()
	Append(Entry{Time: old, Action: "confirm", Command: "a", Context: "prod-eu"})
	Append(Entry{Time: recent, Action: "deny", Command: "b", Context: "staging"})
	Append(Entry{Time: recent, Action: "warn", Command: "c", Context: "production"})

	cases := []struct {
		name string
		f    Filter
		want int
	}{
		{"all", Filter{}, 3},
		{"prod only", Filter{ProdOnly: true}, 2},                         // prod-eu + production
		{"context sub", Filter{ContextSub: "staging"}, 1},                // b
		{"action deny", Filter{ActionEqual: "deny"}, 1},                  // b
		{"since 24h", Filter{Since: time.Now().Add(-24 * time.Hour)}, 2}, // recent two
	}
	for _, c := range cases {
		got, err := Read(c.f)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != c.want {
			t.Errorf("%s: got %d entries, want %d", c.name, len(got), c.want)
		}
	}
}

func TestReadSourceFilter(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	// One from the keyboard (explicit "shell"), one from an AI agent, one
	// legacy entry with no source (must count as "shell").
	Append(Entry{Action: "deny", Command: "a", Source: "shell"})
	Append(Entry{Action: "confirm", Command: "b", Source: "mcp"})
	Append(Entry{Action: "warn", Command: "c"}) // no source → treated as shell

	cases := []struct {
		name string
		f    Filter
		want int
	}{
		{"all sources", Filter{}, 3},
		{"mcp only", Filter{SourceEqual: "mcp"}, 1},               // b
		{"shell includes empty", Filter{SourceEqual: "shell"}, 2}, // a + c
		{"source is case-insensitive", Filter{SourceEqual: "MCP"}, 1},
	}
	for _, c := range cases {
		got, err := Read(c.f)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != c.want {
			t.Errorf("%s: got %d entries, want %d", c.name, len(got), c.want)
		}
	}
}

func TestReadMissingLogIsNotAnError(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	got, err := Read(Filter{})
	if err != nil {
		t.Errorf("reading a missing log should not error: %v", err)
	}
	if got != nil {
		t.Errorf("want nil for a missing log, got %v", got)
	}
}

func TestReadSkipsCorruptLine(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	p := Path()
	os.MkdirAll(filepath.Dir(p), 0o700)
	// One good line, one garbage line.
	os.WriteFile(p, []byte(`{"action":"deny","command":"ok","context":"prod"}`+"\n{not json\n"), 0o600)
	got, err := Read(Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Command != "ok" {
		t.Errorf("a corrupt line should be skipped, kept the good one: %+v", got)
	}
}
