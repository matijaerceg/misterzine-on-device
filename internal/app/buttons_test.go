package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/fonts"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

func TestButtonLabelSets(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	rows := []data.Row{{Base: "Arcade", K: "a", Title: "Alpha", Updated: "2026-09-07"}}
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return now }}, data.Ingest(rows, "", now), nil)
	if a.ButtonLabels() != "mister" {
		t.Fatalf("default set %q", a.ButtonLabels())
	}
	a.cfg.ButtonLabels = "nonsense"
	if a.ButtonLabels() != "mister" {
		t.Fatalf("unknown set fell back to %q", a.ButtonLabels())
	}
	for set, want := range map[string][4]string{
		"mister":      {"A", "B", "X", "Y"},
		"xbox":        {"B", "A", "Y", "X"},
		"playstation": {gfx.Circle, gfx.Cross, gfx.Triangle, gfx.Square},
		"numbers":     {"1", "2", "3", "4"},
	} {
		a.cfg.ButtonLabels = set
		for i, name := range []string{"A", "B", "X", "Y"} {
			if got := a.btn(name); got != want[i] {
				t.Errorf("%s: %s = %q, want %q", set, name, got, want[i])
			}
		}
		// hint chunks: only the bare names change, and never a word's letters
		if got := a.btn("Hold B"); got != "Hold "+want[1] {
			t.Errorf("%s: Hold B = %q", set, got)
		}
		if got := a.btn("A " + gfx.ArrowLeft + " " + gfx.ArrowRight); got != want[0]+" "+gfx.ArrowLeft+" "+gfx.ArrowRight {
			t.Errorf("%s: arrows chunk = %q", set, got)
		}
		for _, same := range []string{"Start", "L/R", "Up/Down", "A-Z", "BAXY", gfx.ArrowUp} {
			if got := a.btn(same); got != same {
				t.Errorf("%s: %q became %q", set, same, got)
			}
		}
	}
	// the PlayStation symbols exist in every font at one cell each
	for name, f := range map[string]*gfx.Font{"body": fonts.Body(), "small": fonts.Small(), "narrow": fonts.Narrow(), "tall": fonts.NarrowTall()} {
		for _, g := range []string{gfx.Cross, gfx.Circle, gfx.Square, gfx.Triangle} {
			ink := 0
			for _, row := range f.Glyph(g[0]) {
				if row != 0 {
					ink++
				}
			}
			if ink < 5 {
				t.Errorf("%s font: glyph %x has %d inked rows", name, g[0], ink)
			}
			if f.Width(g) != f.W {
				t.Errorf("%s font: glyph %x is %d px wide", name, g[0], f.Width(g))
			}
		}
	}
}
