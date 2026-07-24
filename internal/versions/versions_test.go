package versions

import (
	"fmt"
	"strings"
	"testing"
)

// withRecordedRun swaps the package's run for a recorder that captures every
// command line (and optionally fails a matching step), restoring it after.
func withRecordedRun(t *testing.T, failIfContains string) *[]string {
	t.Helper()
	orig := run
	var ran []string
	run = func(argv []string) error {
		line := strings.Join(argv, " ")
		ran = append(ran, line)
		if failIfContains != "" && strings.Contains(line, failIfContains) {
			return fmt.Errorf("simulated failure: %s", line)
		}
		return nil
	}
	t.Cleanup(func() { run = orig })
	return &ran
}

func TestParseSpec(t *testing.T) {
	cases := []struct {
		in            string
		tool, version string
	}{
		{"terraform@1.5", "terraform", "1.5"},
		{"node@20.11.0", "node", "20.11.0"},
		{"kubectl", "kubectl", ""},
		{"go@1.22", "go", "1.22"},
	}
	for _, c := range cases {
		tool, version := ParseSpec(c.in)
		if tool != c.tool || version != c.version {
			t.Errorf("ParseSpec(%q) = (%q,%q), want (%q,%q)",
				c.in, tool, version, c.tool, c.version)
		}
	}
}

func TestUseWithNoManagerErrors(t *testing.T) {
	if _, err := Use(None, "terraform", "1.5"); err == nil {
		t.Error("Use with no manager should return an error")
	}
}

func TestUseAsdfRequiresVersion(t *testing.T) {
	if _, err := Use(Asdf, "terraform", ""); err == nil {
		t.Error("asdf Use without a version should error (asdf needs an explicit version)")
	}
}

func TestUseMiseComposesSingleCommand(t *testing.T) {
	ran := withRecordedRun(t, "")

	got, err := Use(Mise, "terraform", "1.5")
	if err != nil {
		t.Fatalf("Use(mise): %v", err)
	}
	want := "mise use terraform@1.5"
	if len(got) != 1 || got[0] != want {
		t.Errorf("returned cmds = %v, want [%q]", got, want)
	}
	if len(*ran) != 1 || (*ran)[0] != want {
		t.Errorf("ran = %v, want [%q]", *ran, want)
	}
}

func TestUseMiseWithoutVersionPinsBareTool(t *testing.T) {
	withRecordedRun(t, "")
	got, err := Use(Mise, "kubectl", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "mise use kubectl" {
		t.Errorf("bare tool should pin without @: %v", got)
	}
}

func TestUseAsdfRunsPluginInstallLocal(t *testing.T) {
	ran := withRecordedRun(t, "")

	got, err := Use(Asdf, "terraform", "1.5")
	if err != nil {
		t.Fatalf("Use(asdf): %v", err)
	}
	want := []string{
		"asdf plugin add terraform",
		"asdf install terraform 1.5",
		"asdf local terraform 1.5",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("asdf steps = %v, want %v", got, want)
	}
	if strings.Join(*ran, "|") != strings.Join(want, "|") {
		t.Errorf("asdf ran = %v, want %v", *ran, want)
	}
}

func TestUseAsdfToleratesPluginAlreadyAdded(t *testing.T) {
	// `plugin add` failing (plugin already present) must NOT abort the flow —
	// install and local still run.
	ran := withRecordedRun(t, "plugin add")

	_, err := Use(Asdf, "terraform", "1.5")
	if err != nil {
		t.Errorf("a failing `plugin add` should be tolerated, got %v", err)
	}
	if len(*ran) != 3 {
		t.Errorf("all three steps should run despite plugin-add failure, ran %v", *ran)
	}
}

func TestUseAsdfAbortsOnInstallFailure(t *testing.T) {
	// A real failure (install) must abort before `local`.
	ran := withRecordedRun(t, "asdf install")

	_, err := Use(Asdf, "terraform", "1.5")
	if err == nil {
		t.Error("a failing install should abort the flow")
	}
	// plugin add + install ran; local did not.
	if len(*ran) != 2 {
		t.Errorf("flow should stop after install failure, ran %v", *ran)
	}
}
