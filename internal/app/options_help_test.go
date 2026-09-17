package app

import (
	"strings"
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
	rows := []data.Row{{Base: "Arcade", K: "a", Title: "Alpha", Src: "jtcores"}}
	for _, c := range []struct {
		name string
		w, h int
		rot  gfx.Rotation
	}{
		{"320x240 horizontal", 320, 240, gfx.RotNone}, {"320x240 tate", 320, 240, gfx.RotRight},
		{"480x270 horizontal", 480, 270, gfx.RotNone}, {"480x270 tate", 480, 270, gfx.RotRight},
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
			a.cfg.SaverStyle = "shots"
			for _, e := range append(a.optionsEntries(), a.saverEntries()...) {
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

// Credits sits above Quit, the last selectable row of Options; the build
// and data details follow as greyed info rows, so they cost no room until
// scrolled to.
func TestOptionsEndsWithCreditsThenQuit(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240, Version: "v9.9.9", Launcher: func() bool { return true }}, data.Ingest([]data.Row{{Base: "Arcade", K: "a", Title: "Alpha"}}, "", time.Now()), nil)
	E := a.optionsEntries()
	n := len(E)
	if n < 6 || E[n-6].kind != "credits" || E[n-5].kind != "quit" {
		t.Fatalf("rows before the details %q, %q; want credits then quit", E[n-6].text, E[n-5].text)
	}
	for _, e := range E[n-4:] {
		if !e.header || !e.info || e.kind != "" {
			t.Fatalf("detail row %q must be a greyed info row", e.text)
		}
	}
	if E[n-4].text != "" || E[n-3].text != "misterzine v9.9.9" || !strings.HasPrefix(E[n-2].text, "Catalog: ") || !strings.HasPrefix(E[n-1].text, "Last checked: ") {
		t.Fatalf("detail rows %q, %q, %q, %q", E[n-4].text, E[n-3].text, E[n-2].text, E[n-1].text)
	}
}

// A check that finishes while Options is open refreshes the Last checked row.
func TestOptionsLastCheckedRowFollowsChecks(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest([]data.Row{{Base: "Arcade", K: "a", Title: "Alpha"}}, "", time.Now()), nil)
	a.openPanel(ScreenOptions)
	last := func() string { return a.panel.entries[len(a.panel.entries)-1].text }
	if last() != "Last checked: Not yet" {
		t.Fatalf("before any check: %q", last())
	}
	now := time.Date(2026, 9, 17, 12, 34, 0, 0, time.Local)
	a.SetCatalogChecked(now)
	if want := "Last checked: " + catalogStamp(now, "Not yet"); last() != want {
		t.Fatalf("after a check: %q, want %q", last(), want)
	}
}
