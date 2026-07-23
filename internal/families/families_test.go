package families

import (
	"strings"
	"testing"
)

func TestKeysAndByKey(t *testing.T) {
	if len(Keys()) != len(All) {
		t.Fatalf("Keys() len %d != All len %d", len(Keys()), len(All))
	}
	if f, ok := ByKey("kube"); !ok || f.Label != "Kubernetes" {
		t.Errorf("ByKey(kube) = %+v, %v", f, ok)
	}
	if _, ok := ByKey("nope"); ok {
		t.Error("ByKey(nope) should be false")
	}
}

// The default guard policy protects verbs on specific tools; each such
// tool must belong to a family so history/prefilter stay in sync with what
// the guards actually watch. This guards against the taxonomy drifting
// apart again.
func TestGuardedToolsAreInAFamily(t *testing.T) {
	bf := BinFamily()
	for _, tool := range []string{"kubectl", "helm", "terraform"} {
		if _, ok := bf[tool]; !ok {
			t.Errorf("guarded tool %q is not in any family", tool)
		}
	}
	// Every family with danger verbs must have at least one bin.
	for _, f := range All {
		if len(f.DangerVerbs) > 0 && len(f.Bins) == 0 {
			t.Errorf("family %q has danger verbs but no bins", f.Key)
		}
		if strings.TrimSpace(f.Label) == "" {
			t.Errorf("family %q has empty label", f.Key)
		}
	}
}
