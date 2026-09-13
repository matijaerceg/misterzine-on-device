package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// Every Options hint fits the help box without truncation: three lines
// horizontal, four in tate, on the classic 320x240 canvas and the 360x270
// fit canvas, with the 15 px safe zone a CRT typically needs. The tightest
// case is 320x240 tate, 40 columns by 4 lines. Credits and Quit carry no
// hint at all.
func TestOptionsHintsFitTheHelpBox(t *testing.T) {
	rows := []data.Row{{K: "a", Title: "Alpha", Src: "jtcores"}}
	for _, c := range []struct {
		name string
		w, h int
		rot  gfx.Rotation
	}{
		{"320x240 horizontal", 320, 240, gfx.RotNone}, {"320x240 tate", 320, 240, gfx.RotRight},
		{"360x270 horizontal", 360, 270, gfx.RotNone}, {"360x270 tate", 360, 270, gfx.RotRight},
	} {
		for _, update := range []string{"", "v9.9.9"} {
			a := New(Config{PhysW: c.w, PhysH: c.h, Rotation: c.rot, SafeInsetX: 15, SafeInsetY: 15, Launcher: func() bool { return true }},
				data.Ingest(rows, "", time.Now()), nil)
			a.SetAppUpdate(update)
			lines := 4
			if !a.lay.Portrait {
				lines = 3
			}
			cols := a.sm.Cols(a.lay.Body.Dx() - 6)
			for _, e := range a.optionsEntries() {
				if e.kind == "credits" || e.kind == "quit" {
					if e.help != "" {
						t.Errorf("%s: %s has a hint: %q", c.name, e.text, e.help)
					}
					continue
				}
				if e.header {
					continue
				}
				if e.help == "" {
					t.Errorf("%s: %s has no hint", c.name, e.text)
					continue
				}
				if n := len(gfx.Wrap(e.help, cols, 99)); n > lines {
					t.Errorf("%s (%d cols x %d lines): %s needs %d lines: %q", c.name, cols, lines, e.text, n, e.help)
				}
			}
		}
	}
}

// Credits sits above Quit, which is the last row of Options.
func TestOptionsEndsWithCreditsThenQuit(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240, Launcher: func() bool { return true }}, data.Ingest([]data.Row{{K: "a", Title: "Alpha"}}, "", time.Now()), nil)
	E := a.optionsEntries()
	if n := len(E); n < 2 || E[n-2].kind != "credits" || E[n-1].kind != "quit" {
		t.Fatalf("last rows %q, %q; want credits then quit", E[len(E)-2].text, E[len(E)-1].text)
	}
}
