package shellcfg

import (
	"strings"
	"testing"
)

func TestRemoveBlock(t *testing.T) {
	block := markerStart + "\neval \"$(opsforge shell env)\"\n" + markerEnd + "\n"
	cases := []struct {
		name, in, want string
	}{
		{"no block", "export FOO=1\n", "export FOO=1\n"},
		{"only block", block, ""},
		{"block at end", "export FOO=1\n" + block, "export FOO=1\n"},
		{"block in middle", "a\n" + block + "b\n", "a\nb\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := RemoveBlock(c.in); got != c.want {
				t.Errorf("RemoveBlock() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestEnvIsFastAndWellFormed(t *testing.T) {
	out, err := Env(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, markerStart) {
		t.Error("env output does not start with the opsforge marker")
	}
	if !strings.Contains(out, "compinit") {
		t.Error("env output does not guard compinit")
	}
	if !strings.Contains(out, markerEnd) {
		t.Error("env output is not terminated by the end marker")
	}
}
