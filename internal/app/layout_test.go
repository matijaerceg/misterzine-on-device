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
				if style == "list" {
					classic[l.Portrait] = area
				} else if area <= classic[l.Portrait] {
					t.Fatalf("%dx%d inset %d %s: picture %v not bigger than the classic one", size[0], size[1], inset, style, l.Thumb)
				}
				if l.Lines < 4 {
					t.Fatalf("%dx%d inset %d %s: only %d rows", size[0], size[1], inset, style, l.Lines)
				}
				if l.Style != style || l.TextBeside != (style != "list" && !(style == "split" && !l.Portrait)) || l.PaneTop != (style == "picture" && !l.Portrait) {
					t.Fatalf("%dx%d %s: flags style %q beside %v top %v", size[0], size[1], style, l.Style, l.TextBeside, l.PaneTop)
				}
			}
		}
	}
	// the option relays out the list and an unknown saved value is the default
	rows := []data.Row{{Base: "Arcade", K: "a", Title: "Alpha"}, {Base: "Arcade", K: "b", Title: "Beta"}}
	a := New(Config{PhysW: 320, PhysH: 240, ListLayout: "huge"}, data.Ingest(rows, "", time.Now()), nil)
	if a.ListLayout() != "list" || a.lay.Style != "list" {
		t.Fatal("unknown layout not defaulted")
	}
	a.screen = ScreenOptions
	a.buildPanel()
	expandOptionsForTest(a)
	for i, e := range a.panel.entries {
		if e.kind == "list-layout" {
			a.panel.cursor = i
		}
	}
	a.stepValue(1)
	if a.ListLayout() != "split" || a.lay.Style != "split" || a.lay.Pane.Dx() <= paneW {
		t.Fatalf("split not applied: %q %v", a.ListLayout(), a.lay.Pane)
	}
	a.stepValue(1)
	if a.ListLayout() != "picture" || !a.lay.PaneTop {
		t.Fatal("picture not applied")
	}
	if !a.togglePanel() {
		t.Fatal("A on the layout row")
	}
}

func TestFullDisplayLayouts(t *testing.T) {
	for _, size := range [][2]int{{480, 270}, {270, 480}, {640, 360}, {360, 640}, {512, 288}, {288, 512}, {683, 384}, {384, 683}, {688, 288}, {288, 688}} {
		for _, inset := range []int{0, 15, 40} {
			for _, style := range listLayouts {
				l := NewLayout(size[0], size[1], inset, inset, fonts.Body(), fonts.NarrowTall().W, 5, style)
				if !l.Pane.In(l.Body) || !l.List.In(l.Body) || !l.Thumb.In(l.Pane) || l.List.Overlaps(l.Pane) || l.Lines < 4 {
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

func TestPictureColumnsAndTateCaptions(t *testing.T) {
	for _, inset := range []int{0, 15, 40} {
		for _, size := range [][2]int{{480, 270}, {640, 360}, {512, 288}, {688, 288}} {
			l := NewLayout(size[0], size[1], inset, inset, fonts.Body(), fonts.NarrowTall().W, 5, "picture")
			if !l.PictureColumns || l.PaneTop || l.TextBeside || l.Thumb.Overlaps(l.PaneText) || l.Thumb.Dy() != l.Body.Dy()-6 || l.PaneText.Max.X != l.Body.Max.X || l.TitleW > l.List.Dx()-fonts.Body().W {
				t.Fatalf("columns invalid: %+v", l)
			}
			for _, style := range []string{"split", "picture"} {
				portrait := NewLayout(size[1], size[0], inset, inset, fonts.Body(), fonts.NarrowTall().W, 5, style)
				if portrait.Pane.Max.Y-portrait.Thumb.Max.Y-3 < paneH {
					t.Fatalf("tate captions clipped: %+v", portrait)
				}
			}
		}
	}
	classic := NewLayout(320, 240, 15, 15, fonts.Body(), fonts.NarrowTall().W, 5, "picture")
	if classic.PictureColumns || !classic.PaneTop {
		t.Fatal("classic picture must remain stacked")
	}
}
