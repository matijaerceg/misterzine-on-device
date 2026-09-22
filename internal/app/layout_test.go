package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/fonts"
)

// Every list layout keeps the pane inside the body, the list clear of the
// pane, a picture bigger than the classic one, and at least four rows, in
// both orientations and at the widest safe zone.
func TestListLayouts(t *testing.T) {
	classic := map[bool]int{}
	for _, size := range [][2]int{{320, 240}, {240, 320}} {
		for _, inset := range []int{15, 40} {
			for _, style := range listLayouts {
				l := NewLayout(size[0], size[1], inset, inset, fonts.Body(), fonts.NarrowTall().W, 5, style)
				if !l.Pane.In(l.Body) || !l.List.In(l.Body) || !l.Thumb.In(l.Pane) || l.List.Overlaps(l.Pane) {
					t.Fatalf("%dx%d inset %d %s: pane %v list %v thumb %v body %v", size[0], size[1], inset, style, l.Pane, l.List, l.Thumb, l.Body)
				}
				area := l.Thumb.Dx() * l.Thumb.Dy()
				if style == "text" {
					// no pane: the rows take the body, so the titles are wider
					// (horizontal) or there are more of them (tate)
					classic := NewLayout(size[0], size[1], inset, inset, fonts.Body(), fonts.NarrowTall().W, 5, "list")
					if !l.Pane.Empty() || !l.Thumb.Empty() || l.List.Max.X != l.Body.Max.X-4 || l.List.Max.Y != l.Body.Max.Y || l.TitleW < classic.TitleW || l.Lines < classic.Lines || (l.TitleW == classic.TitleW && l.Lines == classic.Lines) {
						t.Fatalf("%dx%d inset %d text: pane %v list %v body %v title %d lines %d", size[0], size[1], inset, l.Pane, l.List, l.Body, l.TitleW, l.Lines)
					}
				} else if style == "list" {
					classic[l.Portrait] = area
				} else if area <= classic[l.Portrait] {
					t.Fatalf("%dx%d inset %d %s: picture %v not bigger than the classic one", size[0], size[1], inset, style, l.Thumb)
				}
				if l.Lines < 4 {
					t.Fatalf("%dx%d inset %d %s: only %d rows", size[0], size[1], inset, style, l.Lines)
				}
				if l.Style != style || l.TextBeside != (style == "picture" || (style == "split" && l.Portrait)) || l.PaneTop != (style == "picture" && !l.Portrait) {
					t.Fatalf("%dx%d %s: flags style %q beside %v top %v", size[0], size[1], style, l.Style, l.TextBeside, l.PaneTop)
				}
			}
		}
	}
	// the shortcut relays out the list and an unknown saved value is the default
	rows := []data.Row{{Base: "Arcade", K: "a", Title: "Alpha"}, {Base: "Arcade", K: "b", Title: "Beta"}}
	a := New(Config{PhysW: 320, PhysH: 240, ListLayout: "huge"}, data.Ingest(rows, "", time.Now()), nil)
	if a.ListLayout() != "list" || a.lay.Style != "list" {
		t.Fatal("unknown layout not defaulted")
	}

	a.cycleListLayout()
	if a.ListLayout() != "split" || a.lay.Style != "split" || a.lay.Pane.Dx() <= paneW {
		t.Fatal("split not applied")
	}
	a.cycleListLayout()
	if a.ListLayout() != "picture" || !a.lay.PaneTop {
		t.Fatal("picture not applied")
	}
	a.cycleListLayout()
	if a.ListLayout() != "text" || !a.lay.Pane.Empty() || a.lay.List.Dx() <= 320-2*paneW {
		t.Fatal("text not applied")
	}
	a.cycleListLayout()
	if a.ListLayout() != "list" {
		t.Fatal("layout did not wrap")
	}
	// Options -> Layout picks one directly
	a.screen = ScreenOptions
	a.buildPanel()
	expandOptionsForTest(a)
	for i, e := range a.panel.entries {
		if e.kind == "list-layout" {
			a.panel.cursor = i
		}
	}
	for range 3 {
		a.stepValue(1)
	}
	if a.ListLayout() != "text" || a.lay.Style != "text" {
		t.Fatalf("Right on the layout row: %q", a.ListLayout())
	}
	if a.stepValue(1) {
		t.Fatal("stepped past the last layout")
	}
	a.stepValue(-1)
	if a.ListLayout() != "picture" || !a.lay.PaneTop {
		t.Fatalf("Left on the layout row: %q", a.ListLayout())
	}
	if !a.togglePanel() {
		t.Fatal("A on the layout row")
	}

}

func TestFullDisplayLayouts(t *testing.T) {
	// The canvases Full display now picks (720p and 1440p, 1366x768,
	// 1600x900, 1680x1050), then the larger ones it used to.
	for _, size := range [][2]int{{480, 270}, {270, 480}, {426, 240}, {240, 426}, {455, 256}, {256, 455}, {533, 300}, {300, 533}, {420, 262}, {262, 420},
		{640, 360}, {360, 640}, {512, 288}, {288, 512}, {683, 384}, {384, 683}, {688, 288}, {288, 688}} {
		for _, inset := range []int{0, 15, 40} {
			for _, style := range listLayouts {
				l := NewLayout(size[0], size[1], inset, inset, fonts.Body(), fonts.NarrowTall().W, 5, style)
				if !l.Pane.In(l.Body) || !l.List.In(l.Body) || !l.Thumb.In(l.Pane) || l.List.Overlaps(l.Pane) || l.Lines < 4 || (style == "text") != l.Pane.Empty() {
					t.Fatalf("%v inset %d %s: invalid layout %+v", size, inset, style, l)
				}
				if l.Portrait && style == "split" && l.Pane.Max.Y-l.Thumb.Max.Y-3 < paneTextH {
					t.Fatalf("no fallback caption room: %+v", l)
				}
				if !l.Portrait && style == "split" && (!l.PaneText.In(l.Pane) || l.PaneText.Dy() < paneTextH) {
					t.Fatalf("metadata clipped: %+v", l)
				}
			}
		}
	}
	classic := NewLayout(270, 360, 15, 15, fonts.Body(), fonts.NarrowTall().W, 5, "list")
	full := NewLayout(270, 480, 15, 15, fonts.Body(), fonts.NarrowTall().W, 5, "list")
	if full.Lines <= classic.Lines {
		t.Fatal("full display must add tate rows")
	}
}

func TestWidescreenAllowsOneColumn(t *testing.T) {
	for _, c := range []struct {
		w, h int
		want bool
	}{
		{480, 270, true}, {426, 240, true}, {455, 256, true}, {533, 300, true}, {688, 288, true},
		{425, 240, false}, // two columns short is not a rounded-down 16:9 display
		{420, 262, false}, {341, 256, false}, {360, 270, false}, {320, 240, false}, {270, 480, false},
	} {
		if got := widescreen(c.w, c.h); got != c.want {
			t.Errorf("widescreen(%d, %d) = %v, want %v", c.w, c.h, got, c.want)
		}
	}
}

func TestPictureColumnsAndTateCaptions(t *testing.T) {
	for _, inset := range []int{0, 15, 40} {
		// 426x240, 455x256 and 533x300 are 16:9 displays rounded down to
		// the framebuffer, a column short of 16:9 each.
		for _, size := range [][2]int{{480, 270}, {426, 240}, {455, 256}, {533, 300}, {640, 360}, {512, 288}, {688, 288}} {
			l := NewLayout(size[0], size[1], inset, inset, fonts.Body(), fonts.NarrowTall().W, 5, "picture")
			if !l.PictureColumns || l.PaneTop || l.TextBeside || l.Thumb.Overlaps(l.PaneText) || l.PaneText.Max.X != l.Body.Max.X || l.TitleW > l.List.Dx()-fonts.Body().W {
				t.Fatalf("columns invalid: %+v", l)
			}
			// The art is as large as the columns allow: full height, or
			// where they leave less width (426x240 with no safe zone),
			// the whole width between the titles and the metadata. At
			// the default safe zone every size gets full height.
			fullH := l.Thumb.Dy() == l.Body.Dy()-6
			artW := (l.PaneText.Min.X - 4) - (l.Pane.Min.X + 3)
			if (!fullH && l.Thumb.Dx() != artW) || (inset == 15 && !fullH) {
				t.Fatalf("%v inset %d: art %v is neither full height nor full width: %+v", size, inset, l.Thumb, l)
			}
			for _, style := range []string{"split", "picture"} {
				portrait := NewLayout(size[1], size[0], inset, inset, fonts.Body(), fonts.NarrowTall().W, 5, style)
				if portrait.Pane.Max.Y-portrait.Thumb.Max.Y-3 < paneH {
					t.Fatalf("tate captions clipped: %+v", portrait)
				}
			}
		}
	}
	// 4:3 and 16:10 canvases keep the stacked picture: the classic size,
	// Full display on a 1024x768 display, and on 1680x1050.
	for _, size := range [][2]int{{320, 240}, {341, 256}, {420, 262}} {
		l := NewLayout(size[0], size[1], 15, 15, fonts.Body(), fonts.NarrowTall().W, 5, "picture")
		if l.PictureColumns || !l.PaneTop {
			t.Fatalf("%v picture must remain stacked", size)
		}
	}
}
