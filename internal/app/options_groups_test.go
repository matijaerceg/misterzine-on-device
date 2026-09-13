package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/fonts"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// Options is five sections, each headed by its mark and a title without a
// colon, with the pad rows under Controls between Display and Operation.
func TestOptionsGroups(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240, Launcher: func() bool { return true }}, data.Ingest([]data.Row{{K: "a", Title: "Alpha"}}, "", time.Now()), nil)
	want := []struct {
		glyph, title string
		kinds        []string
	}{
		{gfx.SectionData, "Data", []string{"refresh", "update", "update-result", "rescan", "prefetch", "clearimg"}},
		{gfx.SectionList, "List", []string{"sources", "filter-rotation", "remember-sort", "views", "title-font", "list-shot", "date-format", "list-layout"}},
		{gfx.SectionDisplay, "Display", []string{"follow-rotation", "rotation", "screensaver", "saver-style", "inset", "canvas"}},
		{gfx.SectionControls, "Controls", []string{"button-labels", "ok-button", "menu-button", "scroll", "hold-delay"}},
		{gfx.SectionOperation, "Operation", []string{"launcher", "open-at-boot", "return-after-game", "troubleshooting", "credits", "quit"}},
	}
	var got []struct {
		glyph, title string
		kinds        []string
	}
	for _, e := range a.optionsEntries() {
		switch {
		case e.header && e.glyph != "":
			got = append(got, struct {
				glyph, title string
				kinds        []string
			}{e.glyph, e.text, nil})
		case e.header:
			if e.text != "" {
				t.Errorf("header %q without a mark", e.text)
			}
		default:
			got[len(got)-1].kinds = append(got[len(got)-1].kinds, e.kind)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("%d sections, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].glyph != want[i].glyph || got[i].title != want[i].title {
			t.Errorf("section %d is %q %q, want %q %q", i, got[i].glyph, got[i].title, want[i].glyph, want[i].title)
		}
		if len(got[i].kinds) != len(want[i].kinds) {
			t.Errorf("%s rows %v, want %v", want[i].title, got[i].kinds, want[i].kinds)
			continue
		}
		for j := range want[i].kinds {
			if got[i].kinds[j] != want[i].kinds[j] {
				t.Errorf("%s row %d is %q, want %q", want[i].title, j, got[i].kinds[j], want[i].kinds[j])
			}
		}
	}
	// the rows that only apply given the row above them are children, drawn
	// a step in behind the branch mark; every other row sits flush
	children := map[string]bool{"rotation": true, "saver-style": true, "saver-bright": true, "open-at-boot": true, "return-after-game": true}
	a.cfg.SaverStyle = "shots"
	seen := 0
	for _, e := range a.optionsEntries() {
		if e.header {
			continue
		}
		if e.child != children[e.kind] {
			t.Errorf("row %q child=%v, want %v", e.kind, e.child, children[e.kind])
		}
		if children[e.kind] {
			seen++
		}
		want := e.text
		if children[e.kind] {
			want = gfx.ChildMark + " " + e.text
		}
		if e.label() != want {
			t.Errorf("row %q draws %q, want %q", e.kind, e.label(), want)
		}
	}
	if seen != len(children) {
		t.Errorf("%d child rows, want %d", seen, len(children))
	}
	// a child row that does not fit falls back to its short name behind
	// the mark rather than losing its tail; a row without one is cut
	bright := panelEntry{text: "Screensaver brightness", child: true, short: "Brightness"}
	if got := bright.fitLabel(30); got != gfx.ChildMark+" Screensaver brightness" {
		t.Errorf("wide: %q", got)
	}
	if got := bright.fitLabel(20); got != gfx.ChildMark+" Brightness" {
		t.Errorf("narrow: %q", got)
	}
	boot := panelEntry{text: "Return after game", child: true}
	if got := boot.fitLabel(10); got != gfx.Fit(gfx.ChildMark+" Return after game", 10) {
		t.Errorf("no short name: %q", got)
	}
	// the marks exist in every font, one cell wide, with enough ink to read
	for name, f := range map[string]*gfx.Font{"body": fonts.Body(), "small": fonts.Small(), "narrow": fonts.Narrow(), "tall": fonts.NarrowTall()} {
		for _, g := range []string{gfx.SectionData, gfx.SectionList, gfx.SectionDisplay, gfx.SectionControls, gfx.SectionOperation, gfx.ChildMark} {
			ink := 0
			for _, row := range f.Glyph(g[0]) {
				if row != 0 {
					ink++
				}
			}
			if ink < 3 {
				t.Errorf("%s font: mark %x has %d inked rows", name, g[0], ink)
			}
			if f.Width(g) != f.W {
				t.Errorf("%s font: mark %x is %d px wide", name, g[0], f.Width(g))
			}
		}
	}
}
