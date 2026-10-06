package access

import "testing"

func TestCoverageSurvivesGraduation(t *testing.T) {
	for _, named := range Catalog() {
		f := named.Feature
		if f.Covered(f.Since-1) || !f.Covered(f.Since) || !f.Covered(f.Since+1) {
			t.Fatalf("bad boundary: %s", named.Name)
		}
		if f.Allowed(f.Since, false) {
			t.Fatal("beta enabled without preference")
		}
		f.Beta = false
		if !f.Allowed(f.Since, false) {
			t.Fatal("graduation lost owned access")
		}
		if f.Allowed(0, false) == f.Fancy {
			t.Fatal("wrong graduation tier")
		}
	}
}
