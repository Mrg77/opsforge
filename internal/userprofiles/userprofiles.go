// Package userprofiles persists user-defined tool profiles to
// ~/.config/opsforge/profiles.yaml, so a DevOps can capture their own
// stack once and reinstall it on any machine with
// `opsforge install --profile <name>`.
//
// User profiles are kept separate from the embedded catalog profiles:
// the file is hand-editable, survives binary upgrades, and never mixes
// with the shipped defaults.
package userprofiles

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/Mrg77/opsforge/internal/catalog"
)

type file struct {
	Profiles []catalog.Profile `yaml:"profiles"`
}

// Path returns the profiles file location (honoring $HOME).
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "opsforge", "profiles.yaml"), nil
}

// Load reads user profiles, returning an empty slice when the file does
// not exist yet (the common first-run case).
func Load() ([]catalog.Profile, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f file
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return f.Profiles, nil
}

// Save adds or replaces a profile by name and writes the file back,
// keeping profiles sorted for a stable, diff-friendly file.
func Save(p catalog.Profile) error {
	if p.Name == "" {
		return fmt.Errorf("profile name is required")
	}
	if len(p.Tools) == 0 {
		return fmt.Errorf("profile %q has no tools", p.Name)
	}
	existing, err := Load()
	if err != nil {
		return err
	}
	replaced := false
	for i := range existing {
		if existing[i].Name == p.Name {
			existing[i] = p
			replaced = true
			break
		}
	}
	if !replaced {
		existing = append(existing, p)
	}
	sort.Slice(existing, func(i, j int) bool { return existing[i].Name < existing[j].Name })
	return write(existing)
}

// Delete removes a profile by name. It is not an error to delete a
// profile that does not exist.
func Delete(name string) error {
	existing, err := Load()
	if err != nil {
		return err
	}
	out := existing[:0]
	for _, p := range existing {
		if p.Name != name {
			out = append(out, p)
		}
	}
	return write(out)
}

func write(profiles []catalog.Profile) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(file{Profiles: profiles})
	if err != nil {
		return err
	}
	header := "# opsforge user profiles — edit freely or manage via the picker (press s).\n"
	return os.WriteFile(path, append([]byte(header), data...), 0o644)
}
