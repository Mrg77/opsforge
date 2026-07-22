// Package output centralizes machine-readable rendering. Commands build a
// plain Go value (struct or map) describing their result and, when the
// user passes --json, emit it as indented JSON instead of the human UI.
// This keeps every command CI-consumable without scattering json.Marshal
// calls through the presentation code.
package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// JSON is set by the global --json persistent flag. Commands check it to
// decide between machine and human output.
var JSON bool

// Emit writes v as indented JSON to stdout. It is the single place JSON is
// produced, so the shape stays consistent (2-space indent, trailing
// newline). Returns an error only if v cannot be marshaled.
func Emit(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding json: %w", err)
	}
	_, err = os.Stdout.Write(append(b, '\n'))
	return err
}
