package shellcfg

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHelpCompletionParsesSubcommands feeds representative `--help` layouts
// through the module's _opsforge_help_parse function and checks the extracted
// subcommand names. This locks in the parser against the two dominant formats
// (aligned two-space, and gh's "name:") so a future tweak can't silently break
// terraform/kubectl/helm/gh/docker completion.
func TestHelpCompletionParsesSubcommands(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available")
	}
	// Locate the module body from the embedded FS.
	body, err := moduleFS.ReadFile("modules/help-completion.zsh")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	mod := filepath.Join(dir, "help-completion.zsh")
	if err := os.WriteFile(mod, body, 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		help string
		want []string
	}{
		{
			name: "terraform-style (aligned two-space)",
			help: "Subcommands:\n" +
				"    list                List resources in the state\n" +
				"    mv                  Move an item in the state\n" +
				"    replace-provider    Replace provider in the state\n",
			want: []string{"list", "mv", "replace-provider"},
		},
		{
			name: "gh-style (name colon)",
			help: "CORE COMMANDS\n" +
				"  auth:          Authenticate gh and git with GitHub\n" +
				"  browse:        Open repositories in the browser\n",
			want: []string{"auth", "browse"},
		},
		{
			name: "cobra-style (two-space)",
			help: "Available Commands:\n" +
				"  create      create a new chart\n" +
				"  dependency  manage a chart's dependencies\n",
			want: []string{"create", "dependency"},
		},
		{
			name: "flags are ignored",
			help: "Flags:\n" +
				"  -h, --help   help for foo\n" +
				"  create      make a thing\n",
			want: []string{"create"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Source the module, pipe the help text through the parser.
			script := "source " + mod + " 2>/dev/null; _opsforge_help_parse"
			cmd := exec.Command("zsh", "-c", script)
			cmd.Stdin = strings.NewReader(tc.help)
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			got := map[string]bool{}
			for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
				if line == "" {
					continue
				}
				name := strings.SplitN(line, "\t", 2)[0]
				got[name] = true
			}
			for _, w := range tc.want {
				if !got[w] {
					t.Errorf("want subcommand %q in parsed output; got %v\nraw:\n%s", w, keys(got), out)
				}
			}
		})
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
