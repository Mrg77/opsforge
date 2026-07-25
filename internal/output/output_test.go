package output

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout redirects os.Stdout for the duration of fn and returns what
// was written — Emit targets os.Stdout directly.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string)
	go func() { b, _ := io.ReadAll(r); done <- string(b) }()
	fn()
	_ = w.Close()
	os.Stdout = orig
	return <-done
}

func TestEmitProducesIndentedJSONWithTrailingNewline(t *testing.T) {
	out := captureStdout(t, func() {
		if err := Emit(map[string]any{"tools": 3, "ok": true}); err != nil {
			t.Errorf("Emit returned error: %v", err)
		}
	})

	// Valid JSON round-trips.
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("Emit output is not valid JSON: %v\n%s", err, out)
	}
	// 2-space indent (the contract the whole --json mode relies on).
	if !strings.Contains(out, "\n  \"") {
		t.Errorf("expected 2-space indented JSON, got:\n%s", out)
	}
	// Exactly one trailing newline.
	if !strings.HasSuffix(out, "}\n") {
		t.Errorf("expected a single trailing newline, got %q", out[len(out)-3:])
	}
}

func TestEmitSlicePreservesOrder(t *testing.T) {
	out := captureStdout(t, func() {
		_ = Emit([]string{"a", "b", "c"})
	})
	var got []string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Errorf("slice order not preserved: %v", got)
	}
}

func TestEmitReturnsErrorOnUnmarshalableValue(t *testing.T) {
	// A channel can't be marshaled — Emit must return the error, not panic.
	out := captureStdout(t, func() {
		if err := Emit(make(chan int)); err == nil {
			t.Error("Emit should error on an unmarshalable value")
		}
	})
	if out != "" {
		t.Errorf("nothing should be written to stdout on a marshal error, got %q", out)
	}
}
