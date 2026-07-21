package versions

import "testing"

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
