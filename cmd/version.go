package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/output"
	"github.com/Mrg77/opsforge/internal/ui"
)

// versionInfo is the shape emitted by `opsforge version --json`.
type versionInfo struct {
	Version  string `json:"version"`
	Commit   string `json:"commit,omitempty"`
	Date     string `json:"date,omitempty"`
	Go       string `json:"go"`
	Platform string `json:"platform"`
	DevBuild bool   `json:"dev_build"`
}

func currentVersion() versionInfo {
	return versionInfo{
		Version:  version,
		Commit:   commit,
		Date:     date,
		Go:       runtime.Version(),
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
		DevBuild: version == "dev" || version == "",
	}
}

// versionCmd is the discoverable, top-level `opsforge version` — the reflex
// command (like `docker version`, `kubectl version`). `opsforge --version` and
// `opsforge self version` still work; this is the one people actually type.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the opsforge version, commit and build info",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		v := currentVersion()
		if output.JSON {
			return output.Emit(v)
		}
		fmt.Printf("opsforge %s\n", ui.Accent.Render(v.Version))
		if v.Commit != "" {
			fmt.Printf("  %s %s\n", ui.Dim.Render("commit  "), v.Commit)
		}
		if v.Date != "" {
			fmt.Printf("  %s %s\n", ui.Dim.Render("built   "), v.Date)
		}
		fmt.Printf("  %s %s\n", ui.Dim.Render("go      "), v.Go)
		fmt.Printf("  %s %s\n", ui.Dim.Render("platform"), v.Platform)
		if v.DevBuild {
			fmt.Println(ui.Faint.Render("  a source/dev build — `opsforge self update` needs a released binary"))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
