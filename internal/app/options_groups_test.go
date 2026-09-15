package app

import (
	"image"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/fonts"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// Options is five sections, each headed by its mark and a title without a
// colon, with the pad rows under Controls between Display and Operation.
func TestOptionsGroups(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240, Launcher: func() bool { return true }}, data.Ingest([]data.Row{{Base: "Arcade", K: "a", Title: "Alpha"}}, "", time.Now()), nil)
	want := []struct {
		glyph, title string
		kinds        []string
	}{
		{gfx.SectionData, "Data", []string{"refresh", "update", "update-result", "rescan", "prefetch", "clearimg"}},
		{gfx.SectionList, "List", []string{"show-non-arcade", "sources", "show-deprecated", "filter-rotation", "views", "remember-sort", "default-view", "title-font", "list-shot", "date-format", "list-layout"}},
		{gfx.SectionDisplay, "Display", []string{"follow-rotation", "rotation", "saver-options", "inset", "canvas", "page-transitions"}},
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
	children := map[string]bool{"remember-sort": true, "default-view": true, "rotation": true, "open-at-boot": true, "return-after-game": true}
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
			want = strings.Repeat("  ", e.depth) + gfx.ChildMark + " " + e.text
		}
		if e.label() != want {
			t.Errorf("row %q draws %q, want %q", e.kind, e.label(), want)
		}
	}
	if seen != len(children) {
		t.Errorf("%d child rows, want %d", seen, len(children))
	}
	// a child row that does not fit falls back to its short name rather
	// than losing its tail; a row without one is cut
	bright := panelEntry{text: "Screensaver brightness", child: true, short: "Brightness"}
	if got := bright.fitText(22); got != "Screensaver brightness" {
		t.Errorf("wide: %q", got)
	}
	if got := bright.fitText(20); got != "Brightness" {
		t.Errorf("narrow: %q", got)
	}
	boot := panelEntry{text: "Return after game", child: true}
	if got := boot.fitText(10); got != gfx.Fit("Return after game", 10) {
		t.Errorf("no short name: %q", got)
	}
	// the mark is drawn in the headings' grey and the name in the row's
	// colour two cells in, so the selected child row reads its mark grey
	a.openPanel(ScreenOptions)
	for i, e := range a.panel.entries {
		if e.kind == "rotation" {
			a.panel.cursor = i
		}
	}
	a.Paint()
	l := &a.lay
	font := a.sm
	row := l.Body.Min.Y + 2
	for i := a.panel.top; i < a.panel.cursor; i++ {
		if e := a.panel.entries[i]; e.header && e.info && e.text == "" {
			row += font.H / 2
		} else {
			row += font.H
		}
	}
	count := func(r image.Rectangle, want rgb) int {
		n := 0
		for py := r.Min.Y; py < r.Max.Y; py++ {
			for px := r.Min.X; px < r.Max.X; px++ {
				if a.logical.RGBA.RGBAAt(px, py) == want {
					n++
				}
			}
		}
		return n
	}
	x := l.Body.Min.X + 4
	mark := image.Rect(x, row, x+font.W, row+font.H)
	name := image.Rect(x+2*font.W, row, x+10*font.W, row+font.H)
	if count(mark, gen.Eva.Muted) == 0 || count(mark, gen.Eva.Accent) != 0 {
		t.Errorf("the selected child row's mark is not drawn in the headings' grey")
	}
	if count(name, gen.Eva.Accent) == 0 || count(name, gen.Eva.Muted) != 0 {
		t.Errorf("the selected child row's name is not drawn in the accent")
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
