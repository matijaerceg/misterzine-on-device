package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// A note on a marker line (members.go's headerNote) sits flush right,
// and a header name too long for the line gives way to it, leaving a
// stretch of rule between the two at every width.
func TestMarkerNoteKeepsItsRule(t *testing.T) {
	long := strings.Repeat("Nichibutsu ", 12)
	for _, c := range []struct {
		name         string
		w, h, inset  int
		rot          gfx.Rotation
		layout, note string
	}{
		{"320x240", 320, 240, 15, gfx.RotNone, "list", "120"},
		{"320x240 tate", 320, 240, 40, gfx.RotRight, "list", "1234"},
		{"480x270 picture", 480, 270, 15, gfx.RotNone, "picture", "7"},
		{"426x240 tate text", 426, 240, 15, gfx.RotLeft, "text", "88"},
	} {
		a := New(Config{PhysW: c.w, PhysH: c.h, Rotation: c.rot, SafeInsetX: c.inset, SafeInsetY: c.inset, ListLayout: c.layout},
			data.Ingest([]data.Row{{K: "a", Title: "Alpha", Base: "Arcade", Manufacturer: long}}, "", time.Now()), nil)
		r := a.lay.lineRect(0)
		canvas := gfx.New(a.lay.W, a.lay.H)
		a.paintMarker(canvas, r, long, c.note)
		mid := r.Min.Y + r.Dy()/2
		nx := r.Max.X - a.sm.Width(" "+c.note)
		for x := nx - a.sm.W; x < nx; x++ {
			if canvas.RGBAAt(x, mid) != gen.Eva.Line {
				t.Fatalf("%s: no rule at x=%d before the note at %d", c.name, x, nx)
			}
		}
		if canvas.RGBAAt(r.Max.X-1, mid) == gen.Eva.Line {
			t.Fatalf("%s: the note does not reach the right end", c.name)
		}
	}
}
