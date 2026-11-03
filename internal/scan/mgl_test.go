package scan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

func TestMGLArcadeRowUsesShortcutCoreDirectory(t *testing.T) {
	card := t.TempDir()
	os.Mkdir(filepath.Join(card, "_Console"), 0755)
	os.WriteFile(filepath.Join(card, "_Console", "NeoGeo_20260101.rbf"), nil, 0644)
	shortcut := filepath.Join(card, "Metal Slug.mgl")
	os.WriteFile(shortcut, []byte(`<mistergamedescription><rbf>_Console/NeoGeo</rbf><file type="f" index="1" path="mslug.neo"/></mistergamedescription>`), 0644)
	r := data.Row{K: "local:neogeo:mslug:fixture", Src: data.SrcLocal, Base: "Arcade", Core: "NeoGeo", MRA: "Metal Slug.mgl"}
	idx := ScanCores(card)
	if got := Status(card, idx, &r); got != data.StatusFoundUndated {
		t.Fatalf("console core should serve arcade MGL: %v", got)
	}
	os.Remove(filepath.Join(card, "_Console", "NeoGeo_20260101.rbf"))
	if got := Status(card, idx, &r); got != data.StatusNotFound {
		t.Fatalf("removed core: %v", got)
	}
}
