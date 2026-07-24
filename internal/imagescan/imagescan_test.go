package imagescan

import "testing"

func TestParsePURL(t *testing.T) {
	cases := []struct {
		purl                      string
		wantType, wantName, wantV string
		ok                        bool
	}{
		{"pkg:golang/helm.sh/helm/v3@3.14.0", "golang", "helm.sh/helm/v3", "3.14.0", true},
		{"pkg:npm/left-pad@1.3.0", "npm", "left-pad", "1.3.0", true},
		{"pkg:pypi/requests@2.31.0", "pypi", "requests", "2.31.0", true},
		{"pkg:deb/debian/openssl@3.0.11?arch=amd64", "deb", "debian/openssl", "3.0.11", true},
		{"pkg:golang/x/y@1.0.0#subpath", "golang", "x/y", "1.0.0", true},
		{"pkg:cargo/serde@1.0.0", "cargo", "serde", "1.0.0", true},
		{"not-a-purl", "", "", "", false},
		{"pkg:golang", "", "", "", false}, // no name
	}
	for _, c := range cases {
		typ, name, v, ok := parsePURL(c.purl)
		if ok != c.ok {
			t.Errorf("%s: ok=%v want %v", c.purl, ok, c.ok)
			continue
		}
		if ok && (typ != c.wantType || name != c.wantName || v != c.wantV) {
			t.Errorf("%s: got (%q,%q,%q) want (%q,%q,%q)",
				c.purl, typ, name, v, c.wantType, c.wantName, c.wantV)
		}
	}
}

func TestPurlToOSV(t *testing.T) {
	cases := map[string]string{ // purl type → OSV ecosystem
		"golang": "Go", "npm": "npm", "pypi": "PyPI",
		"cargo": "crates.io", "gem": "RubyGems", "deb": "Debian", "apk": "Alpine",
		"unknown-type": "", // unmapped → skipped
	}
	for typ, wantEco := range cases {
		eco, _ := purlToOSV(typ, "x")
		if eco != wantEco {
			t.Errorf("purlToOSV(%q) = %q, want %q", typ, eco, wantEco)
		}
	}
}

func TestComponentFromPURLSkipsUnmappedAndVersionless(t *testing.T) {
	if _, ok := componentFromPURL("pkg:golang/x/y@1.2.3", "", ""); !ok {
		t.Error("a mapped, versioned purl should produce a component")
	}
	if _, ok := componentFromPURL("pkg:brew/whatever@1.0", "", ""); ok {
		t.Error("an unmapped purl type should be skipped")
	}
	if _, ok := componentFromPURL("pkg:golang/x/y", "name", ""); ok {
		t.Error("a purl with no usable version should be skipped")
	}
	// A missing purl version falls back to the SBOM component version.
	c, ok := componentFromPURL("pkg:npm/left-pad", "left-pad", "1.3.0")
	if !ok || c.Version != "1.3.0" {
		t.Errorf("fallback version not applied: %+v ok=%v", c, ok)
	}
}

func TestParseCycloneDXExtractsMappedComponents(t *testing.T) {
	sbom := []byte(`{
      "components": [
        {"name":"helm","version":"3.14.0","purl":"pkg:golang/helm.sh/helm/v3@3.14.0"},
        {"name":"some-base-lib","version":"1.0","purl":""},
        {"name":"requests","version":"2.31.0","purl":"pkg:pypi/requests@2.31.0"},
        {"name":"dup","version":"1","purl":"pkg:golang/helm.sh/helm/v3@3.14.0"}
      ]
    }`)
	comps, err := parseCycloneDX(sbom)
	if err != nil {
		t.Fatal(err)
	}
	// helm + requests; the purl-less one is skipped, the dup is deduped.
	if len(comps) != 2 {
		t.Fatalf("want 2 components, got %d: %+v", len(comps), comps)
	}
}

func TestCorrelateFindsVersionDrift(t *testing.T) {
	image := []Component{
		{Name: "kubectl", Version: "1.27.0"},
		{Name: "helm", Version: "3.14.0"},
		{Name: "openssl", Version: "3.0.11"}, // only in the image
	}
	workstation := []WorkstationTool{
		{Name: "kubectl", Version: "1.29.3"},  // differs → drift
		{Name: "helm", Version: "3.14.0"},     // same
		{Name: "terraform", Version: "1.7.0"}, // only on the workstation
	}
	drift := Correlate(image, workstation)
	if len(drift) != 2 { // kubectl + helm are in both; openssl/terraform aren't
		t.Fatalf("want 2 correlated tools, got %d: %+v", len(drift), drift)
	}
	byName := map[string]Drift{}
	for _, d := range drift {
		byName[d.Name] = d
	}
	if !byName["kubectl"].VersionDiffer {
		t.Error("kubectl should be flagged as version drift")
	}
	if byName["helm"].VersionDiffer {
		t.Error("helm matches and should not be flagged")
	}
}

func TestDetectBackendNames(t *testing.T) {
	names := BackendNames()
	if len(names) < 2 || names[0] != "syft" {
		t.Errorf("expected syft preferred among backends, got %v", names)
	}
}
