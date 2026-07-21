// Package versions integrates opsforge with the tool-version managers
// mise and asdf, rather than reimplementing multi-version support.
//
// The `opsforge use <tool>@<version>` command delegates to whichever
// manager is installed (mise preferred — its one-shot `mise use` both
// installs and pins), so pinning a debugging version like
// `terraform@1.5` writes the project's .mise.toml / .tool-versions using
// the mature tool the user already trusts.
package versions

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Manager identifies the detected version manager.
type Manager string

const (
	Mise Manager = "mise"
	Asdf Manager = "asdf"
	None Manager = ""
)

// Detect returns the preferred available manager (mise before asdf), or
// None when neither is installed.
func Detect() Manager {
	if _, err := exec.LookPath("mise"); err == nil {
		return Mise
	}
	if _, err := exec.LookPath("asdf"); err == nil {
		return Asdf
	}
	return None
}

// ParseSpec splits "terraform@1.5" into ("terraform", "1.5"). A missing
// version is allowed (returns an empty version) so callers can decide
// whether to require one.
func ParseSpec(spec string) (tool, version string) {
	if i := strings.LastIndex(spec, "@"); i >= 0 {
		return spec[:i], spec[i+1:]
	}
	return spec, ""
}

// Use pins tool@version in the current directory via the detected
// manager, streaming its output to the user. It returns the commands it
// ran so the caller can show exactly what happened.
func Use(mgr Manager, tool, version string) ([]string, error) {
	switch mgr {
	case Mise:
		// `mise use` installs the version if needed and writes it to the
		// local .mise.toml in one step.
		spec := tool
		if version != "" {
			spec = tool + "@" + version
		}
		cmdline := []string{"mise", "use", spec}
		return []string{strings.Join(cmdline, " ")}, run(cmdline)

	case Asdf:
		if version == "" {
			return nil, fmt.Errorf("asdf requires an explicit version (tool@version)")
		}
		// asdf needs the plugin, then an install, then a local pin.
		steps := [][]string{
			{"asdf", "plugin", "add", tool},
			{"asdf", "install", tool, version},
			{"asdf", "local", tool, version},
		}
		var ran []string
		for _, s := range steps {
			ran = append(ran, strings.Join(s, " "))
			// `plugin add` fails harmlessly if the plugin already exists;
			// don't abort the whole flow on that specific step.
			if err := run(s); err != nil && s[1] != "plugin" {
				return ran, err
			}
		}
		return ran, nil

	default:
		return nil, fmt.Errorf("no version manager found — install mise (`opsforge install mise`) or asdf")
	}
}

func run(argv []string) error {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
