package audit

import "testing"

// Non-régression du faux positif jq : un "fixed" non-parsable ne doit pas
// transformer une CVE en faux positif, sans casser les vrais matchs.
func TestInRangeHandlesDebianVersions(t *testing.T) {
	cases := []struct {
		name, cv, intro, fixed string
		want                   bool
	}{
		{"jq bug: fixed Debian illisible", "1.8.2", "0", "1.6-2.1+deb11u2", false},
		{"fixed absent = range ouverte", "1.8.2", "0", "", true},
		{"version sous le fix parsable", "1.5.0", "0", "1.7.0", true},
		{"version au-dessus du fix", "1.8.2", "0", "1.7.0", false},
		{"fixed avec suffixe -6 parsable", "1.7.0", "0", "1.7.1-6", true},
		{"version = fixed, exclue", "1.7.0", "0", "1.7.0", false},
	}
	for _, c := range cases {
		got := inRange(canonical(c.cv), c.intro, c.fixed)
		if got != c.want {
			t.Errorf("%s: inRange(%s,%s,%s)=%v want %v", c.name, c.cv, c.intro, c.fixed, got, c.want)
		}
	}
}
