// Package catalog holds the curated list of DevOps tools opsforge can
// install, embedded in the binary so it works offline and stays versioned
// with the code.
package catalog

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed catalog.yaml
var raw []byte

// Tool describes one installable CLI.
type Tool struct {
	// Name is the display/catalog identifier, unique across categories.
	Name string `yaml:"name"`
	// Bin is the executable looked up in PATH to detect installation.
	Bin string `yaml:"bin"`
	// Brew is the Homebrew formula (may include a tap prefix).
	Brew string `yaml:"brew"`
	// Cask marks formulas that must be installed with `brew install --cask`.
	Cask        bool   `yaml:"cask"`
	Description string `yaml:"description"`
	// VersionArgs overrides the default `--version` invocation.
	VersionArgs []string `yaml:"version_args"`
	// CompletionZsh is the argv that prints a zsh completion script,
	// empty when the tool has no native zsh completion.
	CompletionZsh []string `yaml:"completion_zsh"`
}

// Category groups tools by domain for display.
type Category struct {
	Name  string `yaml:"name"`
	Tools []Tool `yaml:"tools"`
}

// Profile is a named preset of tools for a typical stack.
type Profile struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tools       []string `yaml:"tools"`
}

// Catalog is the full parsed catalog.
type Catalog struct {
	Profiles   []Profile  `yaml:"profiles"`
	Categories []Category `yaml:"categories"`
}

// Load parses the embedded catalog and validates its invariants.
func Load() (*Catalog, error) {
	var c Catalog
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parsing embedded catalog: %w", err)
	}
	seen := map[string]bool{}
	for _, cat := range c.Categories {
		if cat.Name == "" {
			return nil, fmt.Errorf("catalog contains a category without a name")
		}
		for _, t := range cat.Tools {
			switch {
			case t.Name == "":
				return nil, fmt.Errorf("category %q contains a tool without a name", cat.Name)
			case t.Bin == "":
				return nil, fmt.Errorf("tool %q has no bin", t.Name)
			case t.Brew == "":
				return nil, fmt.Errorf("tool %q has no brew formula", t.Name)
			case seen[t.Name]:
				return nil, fmt.Errorf("duplicate tool name %q", t.Name)
			}
			seen[t.Name] = true
		}
	}
	profiles := map[string]bool{}
	for _, p := range c.Profiles {
		if p.Name == "" {
			return nil, fmt.Errorf("catalog contains a profile without a name")
		}
		if profiles[p.Name] {
			return nil, fmt.Errorf("duplicate profile name %q", p.Name)
		}
		profiles[p.Name] = true
		for _, name := range p.Tools {
			if !seen[name] {
				return nil, fmt.Errorf("profile %q references unknown tool %q", p.Name, name)
			}
		}
	}
	return &c, nil
}

// Tools flattens the catalog into a single list.
func (c *Catalog) Tools() []Tool {
	var out []Tool
	for _, cat := range c.Categories {
		out = append(out, cat.Tools...)
	}
	return out
}

// Tool looks a tool up by its catalog name.
func (c *Catalog) Tool(name string) (Tool, bool) {
	for _, t := range c.Tools() {
		if t.Name == name {
			return t, true
		}
	}
	return Tool{}, false
}

// Profile looks a profile up by name.
func (c *Catalog) Profile(name string) (Profile, bool) {
	for _, p := range c.Profiles {
		if p.Name == name {
			return p, true
		}
	}
	return Profile{}, false
}
