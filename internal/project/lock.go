package project

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/Mrg77/opsforge/internal/audit"
	"github.com/Mrg77/opsforge/internal/detect"
)

// LockFileName sits next to opsforge.yaml and pins the exact versions the
// project resolved to — the piece that turns "workstation-as-code" from a
// wish into reproducibility a reviewer can trust (like package-lock.json /
// mise.lock). `sync` writes it; `sync --check` fails when the machine
// drifts from it.
const LockFileName = "opsforge.lock"

// Lock is the resolved state of a project's toolchain.
type Lock struct {
	// Version pins the lockfile format.
	Version int `yaml:"version"`
	// Tools maps each required tool to its locked version, sorted by name
	// so the file is stable across runs (clean diffs).
	Tools []LockedTool `yaml:"tools"`
}

// LockedTool is one pinned tool.
type LockedTool struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// LockPath returns the lockfile path alongside a given opsforge.yaml.
func LockPath(manifestPath string) string {
	return filepath.Join(filepath.Dir(manifestPath), LockFileName)
}

// BuildLock captures the currently-installed versions of the project's
// resolved tools into a Lock. Only installed tools are pinned (you can't
// lock a version you don't have); the caller reports any that were missing.
func BuildLock(tools []string, statuses map[string]detect.Status) Lock {
	l := Lock{Version: 1}
	for _, name := range tools {
		s := statuses[name]
		if !s.Installed {
			continue
		}
		l.Tools = append(l.Tools, LockedTool{
			Name:    name,
			Version: audit.NormalizeVersion(s.Version), // bare x.y.z, comparable
		})
	}
	sort.Slice(l.Tools, func(a, b int) bool { return l.Tools[a].Name < l.Tools[b].Name })
	return l
}

// WriteLock writes the lockfile.
func WriteLock(path string, l Lock) error {
	var b bytes.Buffer
	b.WriteString("# opsforge.lock — resolved tool versions for this project.\n")
	b.WriteString("# Written by `opsforge sync`; checked by `opsforge sync --check`.\n")
	b.WriteString("# Commit it so anyone reproduces the exact same toolchain.\n")
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(l); err != nil {
		return err
	}
	_ = enc.Close()
	return os.WriteFile(path, b.Bytes(), 0o644)
}

// ReadLock reads a lockfile; ok=false when it doesn't exist yet.
func ReadLock(path string) (Lock, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Lock{}, false, nil
	}
	if err != nil {
		return Lock{}, false, err
	}
	var l Lock
	if err := yaml.Unmarshal(data, &l); err != nil {
		return Lock{}, false, fmt.Errorf("parsing %s: %w", path, err)
	}
	return l, true, nil
}

// LockDrift is a single deviation between the lockfile and the machine.
type LockDrift struct {
	Name     string `json:"name"`
	Expected string `json:"expected"` // version in the lock ("" = not locked)
	Got      string `json:"got"`      // version on the machine ("" = missing)
}

// CheckLock compares the lockfile against the currently-installed versions
// and returns the drifts (empty = the machine matches the lock exactly).
func CheckLock(l Lock, statuses map[string]detect.Status) []LockDrift {
	var drift []LockDrift
	for _, lt := range l.Tools {
		got := audit.NormalizeVersion(statuses[lt.Name].Version)
		if !statuses[lt.Name].Installed {
			drift = append(drift, LockDrift{lt.Name, lt.Version, ""})
			continue
		}
		// An empty locked version means "pinned but version was unknown at
		// lock time" — don't flag a mismatch we can't judge.
		if lt.Version != "" && got != lt.Version {
			drift = append(drift, LockDrift{lt.Name, lt.Version, got})
		}
	}
	return drift
}
