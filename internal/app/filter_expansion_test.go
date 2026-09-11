package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// Existing facet tests operate on open sections; opening behavior is tested below.
func openExpandedFilters(a *App) {
	a.openPanel(ScreenFilter)
	a.panel.sectionClosed = nil
	a.buildPanel()
}

func TestActiveSectionMarkerFollowsCurrentChoices(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest([]data.Row{
		{K: "a", Base: "Arcade", Genre: "Shooter"}, {K: "b", Base: "Arcade", Genre: "Puzzle"},
	}, "", time.Now()), nil)
	heading := func(kind string) string {
		for _, e := range a.filterEntries() {
			if e.header && e.kind == kind {
				return e.text
			}
		}
		t.Fatal("missing heading", kind)
		return ""
	}
	a.SetFilters(data.Filters{GenreOff: map[string]bool{"Puzzle": true}})
	a.openPanel(ScreenFilter)
	if heading("genre") != gfx.ArrowDown+" * Genre" {
		t.Fatal("expanded active marker")
	}
	a.panel.sectionClosed["genre"] = true
	if heading("genre") != gfx.ArrowRight+" * Genre" {
		t.Fatal("collapsed active marker")
	}
	a.SetFilters(data.Filters{})
	if heading("genre") != gfx.ArrowRight+" Genre" {
		t.Fatal("default marker did not clear")
	}
	a.cfg.FilterRotation = true
	a.cfg.IniOrientation = "v"
	if !strings.Contains(heading("rot"), "* Rotation") {
		t.Fatal("INI restriction not marked")
	}
	a.cfg.FilterRotation = false
	if strings.Contains(heading("rot"), "*") {
		t.Fatal("disabled INI restriction still marked")
	}
	a.SetFilters(data.Filters{Install: data.InstallAll})
	if strings.Contains(heading("install"), "*") {
		t.Fatal("default install choice marked")
	}
}

func TestFilterOpenShowsOnlyEditsAndRemovesFavorites(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240, Rotation: gfx.RotLeft, SafeInsetX: 40, SafeInsetY: 40}, data.Ingest([]data.Row{
		{K: "a", Base: "Arcade", Year: "1980", Genre: "Shooter"},
		{K: "b", Base: "Arcade", Year: "1981", Genre: "Puzzle"},
	}, "", time.Now()), nil)
	a.openPanel(ScreenFilter)
	for _, e := range a.panel.entries {
		if !e.info && (!e.header || !strings.HasPrefix(e.text, gfx.ArrowRight)) {
			t.Fatal("unset section should be collapsed", e)
		}
		if e.kind == "fav" || strings.Contains(e.text, "unaffected") {
			t.Fatal("redundant row", e)
		}
	}
	a.SetFilters(data.Filters{GenreOff: map[string]bool{"Puzzle": true}, YearOff: map[string]bool{"1980": true}})
	a.openPanel(ScreenFilter)
	if a.panel.sectionClosed["genre"] || a.panel.sectionClosed["year"] || !a.panel.sectionClosed["install"] || !a.panel.yearOpen["1980s"] {
		t.Fatal("edits not revealed")
	}
	for i, e := range a.panel.entries {
		if e.kind == "decade" {
			a.panel.cursor = i
			hint := a.filterHint()
			if !strings.Contains(hint, "A toggle") || !strings.Contains(hint, "open/close") || a.sm.Width(hint) > a.lay.Hint.Dx()-4 {
				t.Fatal("decade hints must be explicit and fit", hint)
			}
		}
	}
	for i, e := range a.panel.entries {
		if e.text == "Arcade game filters:" && (i == 0 || a.panel.entries[i-1].text != "") {
			t.Fatal("missing spacer")
		}
		if e.kind == "genre" && e.header {
			if !strings.HasPrefix(e.text, gfx.ArrowDown) {
				t.Fatal("expanded arrow")
			}
			a.panel.cursor = i
		}
	}
	a.actPanel(platform.KeyLeft)
	a.openPanel(ScreenFilter)
	if a.panel.sectionClosed["genre"] {
		t.Fatal("page reopen should reveal edited section again")
	}
	a.SetFilters(data.Filters{})
	a.openPanel(ScreenFilter)
	if !a.panel.sectionClosed["genre"] || len(a.panel.yearOpen) != 0 {
		t.Fatal("cleared sections must close next visit")
	}
	if a.sm.Width(a.filterHint()) > a.lay.Hint.Dx()-4 {
		t.Fatal("footer does not fit")
	}
}
