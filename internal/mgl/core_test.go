package mgl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCorePresentHonorsDirectoryAndVariant(t *testing.T) {
	card := t.TempDir()
	os.MkdirAll(filepath.Join(card, "_Console"), 0755)
	for _, name := range []string{"NeoGeo_20260101.rbf", "NeoGeo_LLAPI_20260101.rbf"} {
		if err := os.WriteFile(filepath.Join(card, "_Console", name), nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		rbf  string
		want bool
	}{
		{"_Console/NeoGeo", true}, {"_Console/NeoGeo_LLAPI", true},
		{"_Other/NeoGeo", false}, {"_Console/NeoGeoYC", false},
		{"/etc/NeoGeo", false}, {"../NeoGeo", false},
	} {
		if got := CorePresent(card, Document{RBF: tc.rbf}); got != tc.want {
			t.Errorf("%s: %v", tc.rbf, got)
		}
	}
	os.Remove(filepath.Join(card, "_Console", "NeoGeo_20260101.rbf"))
	if CorePresent(card, Document{RBF: "_Console/NeoGeo"}) {
		t.Fatal("different variant satisfied base core")
	}
}
