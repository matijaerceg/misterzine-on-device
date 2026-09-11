package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestMainRowsFollowCenterAndStopAtEnds(t *testing.T) {
	var rows []data.Row
	for i := 0; i < 11; i++ {
		rows = append(rows, data.Row{K: fmt.Sprint(i), Title: fmt.Sprintf("%02d", i)})
	}
	a := New(Config{PhysW: 320, PhysH: 240, RememberSort: true, LastSort: data.SortAlphabetical}, data.Ingest(rows, "", time.Now()), nil)
	a.lay.Lines = 5
	want := []int{0, 0, 0, 1, 2, 3, 4, 5, 6, 6, 6}
	for i := 0; i < len(rows); i++ {
		if i > 0 {
			a.actList(platform.KeyDown)
		}
		if a.top != want[i] {
			t.Fatalf("down row %d: top %d want %d", i, a.top, want[i])
		}
	}
	for i := len(rows) - 2; i >= 0; i-- {
		a.actList(platform.KeyUp)
		if a.top != want[i] {
			t.Fatalf("up row %d: top %d want %d", i, a.top, want[i])
		}
	}
	// A last-look divider occupies a real display line.
	a.split = 4
	a.cursor = 4
	a.actList(platform.KeyDown)
	if a.top != 4 {
		t.Fatal("centering ignored marker line", a.top)
	}
}

func TestFilterSectionsAndCenteredRows(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		a := New(Config{PhysW: 320, PhysH: 240, Rotation: rot}, data.Ingest([]data.Row{
			{K: "a", Base: "Arcade", Year: "1980", Genre: "Shooter"},
			{K: "b", Base: "Arcade", Year: "1981", Genre: "Puzzle"},
			{K: "c", Base: "Arcade", Year: "1990"},
		}, "", time.Now()), nil)
		a.openPanel(ScreenFilter)
		a.Paint()
		for steps := 0; steps < len(a.panel.entries)+2; steps++ {
			a.actPanel(platform.KeyDown)
			a.Paint()
			p := a.panel
			if p.top < 0 || p.top > max(0, len(p.entries)-p.lines) {
				t.Fatal("scroll beyond end")
			}
			if p.top > 0 && p.top < len(p.entries)-p.lines && p.cursor-p.top != p.lines/2 {
				t.Fatal("selection not centered")
			}
		}
		if a.panel.top != max(0, len(a.panel.entries)-a.panel.lines) {
			t.Fatal("bottom not filled")
		}
		for steps := 0; steps < len(a.panel.entries)+2; steps++ {
			a.actPanel(platform.KeyUp)
			a.Paint()
		}
		if a.panel.top != 0 {
			t.Fatal("did not return to top")
		}
		var headers []int
		for i, e := range a.panel.entries {
			if e.header && !e.info && e.kind != "" {
				headers = append(headers, i)
			}
		}
		for _, i := range headers[1:] {
			a.actPanel(platform.KeyPageDown)
			if a.panel.cursor != i {
				t.Fatal("R did not jump to next section", a.panel.entries[a.panel.cursor])
			}
		}
		a.actPanel(platform.KeyPageDown)
		if a.panel.cursor != headers[len(headers)-1] {
			t.Fatal("R should stop at final section")
		}
		for j := len(headers) - 2; j >= 0; j-- {
			a.actPanel(platform.KeyPageUp)
			if a.panel.cursor != headers[j] {
				t.Fatal("L did not jump to previous section")
			}
		}
		a.actPanel(platform.KeyLeft)
		if !a.panel.sectionClosed["install"] {
			t.Fatal("Left must collapse section")
		}
		a.actPanel(platform.KeyPageDown)
		if e := a.panel.entries[a.panel.cursor]; e.kind != "fav" || !e.header {
			t.Fatal("R must skip collapsed section")
		}
		a.actPanel(platform.KeyPageUp)
		a.actPanel(platform.KeyRight)
		if a.panel.sectionClosed["install"] {
			t.Fatal("Right must reopen section")
		}
		for i, e := range a.panel.entries {
			if e.kind == "decade" && e.value == "1980s" {
				a.panel.cursor = i
				break
			}
		}
		a.actPanel(platform.KeyRight)
		a.actPanel(platform.KeyDown)
		a.actPanel(platform.KeyLeft)
		if a.panel.yearOpen["1980s"] || a.panel.sectionClosed["year"] {
			t.Fatal("first Left must close decade only")
		}
		a.actPanel(platform.KeyLeft)
		if !a.panel.sectionClosed["year"] || !a.panel.entries[a.panel.cursor].header {
			t.Fatal("second Left must close year section")
		}
		a.actPanel(platform.KeyRight)
		if a.panel.sectionClosed["year"] || a.filters.Active() {
			t.Fatal("expansion must not change filters")
		}
		a.actPanel(platform.KeyPageDown)
		if e := a.panel.entries[a.panel.cursor]; e.kind != "rot" || !e.header {
			t.Fatal("expanded years created extra section stops")
		}
	}
}
