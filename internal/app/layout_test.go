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
	rows := []data.Row{{K: "a", Title: "Alpha"}, {K: "b", Title: "Beta"}}
	a := New(Config{PhysW: 320, PhysH: 240, ListLayout: "huge"}, data.Ingest(rows, "", time.Now()), nil)
	if a.ListLayout() != "list" || a.lay.Style != "list" {
		t.Fatal("unknown layout not defaulted")
	}
	a.screen = ScreenOptions
	a.buildPanel()
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
