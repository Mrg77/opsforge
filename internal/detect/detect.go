// Package detect discovers which catalog tools are already present on the
// machine and extracts a human-readable version string for each.
package detect

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Mrg77/opsforge/internal/catalog"
)

// SafeProbeEnv returns the environment for probing tools without side
// effects. KUBECONFIG points at an empty file so kubectl (and wrappers
// shipped by cloud SDKs) can never discover an exec credential plugin —
// otherwise a mere version probe can pop an OIDC browser login.
func SafeProbeEnv() []string {
	return append(os.Environ(), "KUBECONFIG="+os.DevNull)
}

// Status is the detection result for one tool.
type Status struct {
	Installed bool
	// Version is a best-effort one-line version string, empty when the
	// tool is absent or its version output could not be parsed.
	Version string
	// Outdated is true when a newer version is available (currently
	// determined via `brew outdated` for brew-managed tools).
	Outdated bool
}

const versionTimeout = 3 * time.Second

// Tool checks PATH for the tool's binary and, when present, runs its
// version command.
func Tool(t catalog.Tool) Status {
	if _, err := exec.LookPath(t.Bin); err != nil {
		return Status{}
	}
	return Status{Installed: true, Version: version(t)}
}

// detectConcurrency bounds how many version probes run at once. Each probe
// execs a CLI (some heavyweight, e.g. gcloud/az), so an unbounded fan-out
// over a large catalog would spawn a hundred-plus processes at once and
// thrash a constrained CI runner. We cap at a small multiple of the CPU
// count — enough to hide the per-probe latency, not enough to fork-bomb.
func detectConcurrency() int {
	n := runtime.NumCPU() * 2
	if n < 4 {
		n = 4
	}
	if n > 16 {
		n = 16
	}
	return n
}

// All runs detection concurrently over a flat tool list, keyed by tool
// name. Version commands of heavyweight CLIs (gcloud, az...) take up to
// a few seconds each, so running them sequentially is not an option — but
// the fan-out is bounded (see detectConcurrency) so it never fork-bombs.
func All(tools []catalog.Tool) map[string]Status {
	out := make(map[string]Status, len(tools))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, detectConcurrency())
	for _, t := range tools {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}        // acquire a slot
			defer func() { <-sem }() // release it
			s := Tool(t)
			mu.Lock()
			out[t.Name] = s
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}

func version(t catalog.Tool) string {
	args := t.VersionArgs
	if len(args) == 0 {
		args = []string{"--version"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), versionTimeout)
	defer cancel()
	// Some tools print the version on stderr, so both streams are read;
	// a non-zero exit still often carries usable output.
	cmd := exec.CommandContext(ctx, t.Bin, args...)
	cmd.Env = SafeProbeEnv()
	// Wrapper CLIs (gcloud, az) spawn children that inherit the output
	// pipe; without WaitDelay, killing the parent on timeout leaves
	// CombinedOutput blocked on the pipe forever.
	cmd.WaitDelay = time.Second
	out, _ := cmd.CombinedOutput()
	return FirstVersionLine(string(out))
}

// FirstVersionLine returns the first non-empty line containing a digit,
// which reliably skips ASCII-art banners and blank prefixes.
func FirstVersionLine(output string) string {
	for line := range strings.Lines(output) {
		line = strings.TrimSpace(line)
		if line == "" || !strings.ContainsAny(line, "0123456789") {
			continue
		}
		if len(line) > 60 {
			line = line[:60] + "…"
		}
		return line
	}
	return ""
}
