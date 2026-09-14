package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// The Options legend follows the row under the cursor: Change with the
// arrow that can still move on a live choice row, A with what it does on
// a row that answers A, and Back alone on a greyed row; every legend fits
// the hint bar in both layouts at the widest safe zone.
func TestOptionsHintFollowsTheRow(t *testing.T) {
	rows := []data.Row{{K: "a", Title: "Alpha", Src: "jtcores"}}
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotRight} {
		a := New(Config{PhysW: 320, PhysH: 240, Rotation: rot, SafeInsetX: 40, SafeInsetY: 40, Launcher: func() bool { return false }},
			data.Ingest(rows, "", time.Now()), nil)
		a.cfg.SaverStyle = "shots"
		a.openPanel(ScreenOptions)
		both, left, right := gfx.ArrowLeft+" "+gfx.ArrowRight+" Change", gfx.ArrowLeft+" Change", gfx.ArrowRight+" Change"
		seen := map[string]bool{}
		for i, e := range a.panel.entries {
			if e.header {
				continue
			}
			a.panel.cursor = i
			hint := a.optionsHint()
			if !strings.HasSuffix(hint, "B Back") {
				t.Errorf("%s: %q does not end with Back", e.text, hint)
			}
			change := len(e.vals) > 1 && !e.disabled
			if strings.Contains(hint, "Change") != change {
				t.Errorf("%s: %q names Change=%v, want %v", e.text, hint, !change, change)
			}
			if change {
				want := both
				if e.idx == 0 {
					want = right
				} else if e.idx == len(e.vals)-1 {
					want = left
				}
				if !strings.HasPrefix(hint, want+"  ") {
					t.Errorf("%s (choice %d of %d): %q, want it to start %q", e.text, e.idx, len(e.vals), hint, want)
				}
			}
			if acts := optionsActs[e.kind] != ""; strings.Contains(hint, "  A ") != acts && !strings.HasPrefix(hint, "A ") == acts {
				t.Errorf("%s: %q names A=%v, want %v", e.text, hint, !acts, acts)
			}
			if e.disabled && hint != "B Back" {
				t.Errorf("%s is greyed yet the legend is %q", e.text, hint)
			}
			if !a.hintFits(hint) {
				t.Errorf("rot=%v: %q does not fit the hint bar", rot, hint)
			}
			seen[hint] = true
		}
		// the rows that matter for the shape: a greyed child, both ends
		// of a choice, a preview row and an action row
		for _, want := range []string{"B Back", "A Open  B Back", both + "  B Back", "A Quit  B Back"} {
			if !seen[want] {
				t.Errorf("rot=%v: no row gives %q; got %v", rot, want, seen)
			}
		}
		// the painted bar is the row's legend
		for i, e := range a.panel.entries {
			if e.header {
				continue
			}
			a.panel.cursor = i
			a.all = true
			a.Paint()
			c := gfx.New(a.lay.W, a.lay.H)
			a.paintHint(c, a.optionsHint())
			for y := a.lay.Hint.Min.Y; y < a.lay.Hint.Max.Y; y++ {
				for x := a.lay.Hint.Min.X; x < a.lay.Hint.Max.X; x++ {
					if a.logical.RGBA.RGBAAt(x, y) != c.RGBA.RGBAAt(x, y) {
						t.Fatalf("rot=%v %s: the painted legend is not %q", rot, e.text, a.optionsHint())
					}
				}
			}
		}
	}
}
