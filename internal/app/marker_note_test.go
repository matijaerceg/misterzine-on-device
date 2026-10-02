package app

import (
	"image"
	"strings"
	"testing"
	"time"

	"bytes"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// Notes stay beside the fitted header name, in parentheses, at every width.
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
		expected := gfx.New(a.lay.W, a.lay.H)
		expected.HLine(r.Min.X, r.Max.X-1, r.Min.Y+r.Dy()/2, pal.Line)
		note := " (" + c.note + ")"
		cols := min(a.sm.Cols(r.Dx())-2, (r.Dx()-a.body.W)/a.sm.W-3-len(note))
		label := " " + gfx.Fit(long, cols) + note + " "
		x := r.Min.X + a.body.W
		expected.Fill(image.Rect(x, r.Min.Y, x+a.sm.Width(label), r.Max.Y), pal.Bg)
		expected.Text(x, r.Min.Y+2, a.sm, label, pal.Muted)
		if !bytes.Equal(canvas.Pix, expected.Pix) {
			t.Fatalf("%s: marker note is not beside its fitted name", c.name)
		}
	}
}
