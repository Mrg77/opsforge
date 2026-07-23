package catalog

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// overlayPaths returns the user catalog files to merge on top of the
// embedded catalog, in load order (most general first):
//
//  1. every ~/.config/opsforge/catalog.d/*.yaml (sorted)
//  2. the file named by $OPSFORGE_CATALOG, if set
//
// This lets an engineer add internal/private tools without a PR against
// the embedded catalog — the #1 reason the fixed 106-tool list wasn't
// enough for real teams.
func overlayPaths() []string {
	var paths []string
	if home, err := os.UserHomeDir(); err == nil {
		dir := filepath.Join(home, ".config", "opsforge", "catalog.d")
		if entries, err := os.ReadDir(dir); err == nil {
			var files []string
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				if ext := filepath.Ext(e.Name()); ext == ".yaml" || ext == ".yml" {
					files = append(files, filepath.Join(dir, e.Name()))
				}
			}
			sort.Strings(files)
			paths = append(paths, files...)
		}
	}
	if f := os.Getenv("OPSFORGE_CATALOG"); f != "" {
		paths = append(paths, f)
	}
	return paths
}

// MergeOverlays layers user catalog files onto the receiver. Each overlay
// is a Catalog document with the same shape as the embedded one:
//
//	categories:
//	  - name: Internal
//	    tools:
//	      - name: acme-cli
//	        bin: acme
//	        brew: acmecorp/tap/acme-cli
//	        description: Our internal deploy CLI
//	profiles:
//	  - name: my-stack
//	    tools: [acme-cli, kubectl]
//
// Merge semantics:
//   - a tool whose name matches an existing tool REPLACES it (override),
//   - a new tool is appended to its named category (created if absent),
//   - a profile whose name matches an existing one replaces it, else it is
//     appended.
//
// A missing overlay path is skipped silently (the env var may point at a
// file that isn't there yet); an unreadable or malformed file is a hard
// error so a typo can't silently drop the user's tools.
func (c *Catalog) MergeOverlays(paths []string) error {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("reading catalog overlay %s: %w", path, err)
		}
		var ov Catalog
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true) // reject unknown fields — catch typos early
		if err := dec.Decode(&ov); err != nil {
			return fmt.Errorf("parsing catalog overlay %s: %w", path, err)
		}
		c.mergeCategories(ov.Categories)
		c.mergeProfiles(ov.Profiles)
	}
	return nil
}

func (c *Catalog) mergeCategories(cats []Category) {
	for _, oc := range cats {
		for _, t := range oc.Tools {
			if !c.replaceTool(t) {
				c.appendTool(oc.Name, t)
			}
		}
	}
}

// replaceTool overwrites an existing tool with the same name, anywhere in
// the catalog. Returns true if it replaced one.
func (c *Catalog) replaceTool(t Tool) bool {
	for ci := range c.Categories {
		for ti := range c.Categories[ci].Tools {
			if c.Categories[ci].Tools[ti].Name == t.Name {
				c.Categories[ci].Tools[ti] = t
				return true
			}
		}
	}
	return false
}

// appendTool adds a new tool to a category, creating the category if it
// doesn't exist yet.
func (c *Catalog) appendTool(category string, t Tool) {
	for ci := range c.Categories {
		if c.Categories[ci].Name == category {
			c.Categories[ci].Tools = append(c.Categories[ci].Tools, t)
			return
		}
	}
	c.Categories = append(c.Categories, Category{Name: category, Tools: []Tool{t}})
}

func (c *Catalog) mergeProfiles(profiles []Profile) {
	for _, op := range profiles {
		replaced := false
		for pi := range c.Profiles {
			if c.Profiles[pi].Name == op.Name {
				c.Profiles[pi] = op
				replaced = true
				break
			}
		}
		if !replaced {
			c.Profiles = append(c.Profiles, op)
		}
	}
}
