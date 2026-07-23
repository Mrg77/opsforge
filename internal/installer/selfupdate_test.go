package installer

import "testing"

func TestNewerAvailable(t *testing.T) {
	cases := []struct {
		name          string
		current       string
		latest        string
		wantAvailable bool
		wantDev       bool
	}{
		{"older patch", "v0.4.0", "v0.5.0", true, false},
		{"older minor within same major", "v0.4.0", "v0.4.1", true, false},
		{"same version", "v0.5.0", "v0.5.0", false, false},
		{"current newer than latest", "v0.6.0", "v0.5.0", false, false},
		{"bare current vs v-tagged latest", "0.4.0", "v0.5.0", true, false},
		{"v-tagged current vs bare latest", "v0.4.0", "0.5.0", true, false},
		{"dev build never updates", "dev", "v0.5.0", false, true},
		{"empty version treated as dev", "", "v0.5.0", false, true},
		{"unparseable latest is not offered", "v0.4.0", "not-a-version", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := NewerAvailable(c.current, c.latest)
			if got.Available != c.wantAvailable {
				t.Errorf("Available = %v, want %v (current=%q latest=%q)",
					got.Available, c.wantAvailable, c.current, c.latest)
			}
			if got.Dev != c.wantDev {
				t.Errorf("Dev = %v, want %v (current=%q)", got.Dev, c.wantDev, c.current)
			}
			if got.Current != c.current || got.Latest != c.latest {
				t.Errorf("echoed versions = (%q,%q), want (%q,%q)",
					got.Current, got.Latest, c.current, c.latest)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"v0.4.0", "v0.5.0", true},
		{"v0.5.0", "v0.4.0", false},
		{"v0.5.0", "v0.5.0", false},
		{"v1.0.0", "v1.0.1", true},
		{"garbage", "v0.5.0", false},
		{"v0.4.0", "garbage", false},
	}
	for _, c := range cases {
		if got := compareVersions(c.current, c.latest); got != c.want {
			t.Errorf("compareVersions(%q, %q) = %v, want %v",
				c.current, c.latest, got, c.want)
		}
	}
}
