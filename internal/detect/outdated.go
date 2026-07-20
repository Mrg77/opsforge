package detect

import (
	"context"
	"encoding/json"
	"os/exec"
	"time"

	"github.com/Mrg77/opsforge/internal/catalog"
)

// Outdated returns the set of Homebrew formula/cask names that have a
// newer version available. It shells out once to `brew outdated --json`,
// so the whole toolbox is checked in a single call. When brew is absent
// or the call fails, it returns an empty set (no upgrades surfaced) —
// upgrade availability is a nice-to-have, never a hard error.
func Outdated() map[string]bool {
	out := map[string]bool{}
	if _, err := exec.LookPath("brew"); err != nil {
		return out
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	data, err := exec.CommandContext(ctx, "brew", "outdated", "--json=v2").Output()
	if err != nil {
		return out
	}
	var parsed struct {
		Formulae []struct {
			Name string `json:"name"`
		} `json:"formulae"`
		Casks []struct {
			Name string `json:"name"`
		} `json:"casks"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return out
	}
	for _, f := range parsed.Formulae {
		out[f.Name] = true
	}
	for _, c := range parsed.Casks {
		out[c.Name] = true
	}
	return out
}

// brewLeaf extracts the bare formula name from a possibly-tapped brew
// reference: "hashicorp/tap/terraform" -> "terraform".
func brewLeaf(brew string) string {
	for i := len(brew) - 1; i >= 0; i-- {
		if brew[i] == '/' {
			return brew[i+1:]
		}
	}
	return brew
}

// AllWithOutdated runs full detection and marks tools whose Homebrew
// formula appears in the outdated set. Detection and the outdated query
// run concurrently since neither depends on the other.
func AllWithOutdated(tools []catalog.Tool) map[string]Status {
	var (
		statuses map[string]Status
		outdated map[string]bool
	)
	done := make(chan struct{}, 2)
	go func() { statuses = All(tools); done <- struct{}{} }()
	go func() { outdated = Outdated(); done <- struct{}{} }()
	<-done
	<-done

	for _, t := range tools {
		s := statuses[t.Name]
		if s.Installed && t.Brew != "" && outdated[brewLeaf(t.Brew)] {
			s.Outdated = true
			statuses[t.Name] = s
		}
	}
	return statuses
}
