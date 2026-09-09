package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestDetailsSpaceDoesNotDependOnAlternativeCount(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
		for _, inset := range []int{15, 40} {
			row := data.Row{K: "game", Title: "A long title for the cramped details layout", Core: "Game",
				Img: "game", ImgSlots: []string{"snap"}, ImgW: 320, ImgH: 240,
				Note: strings.Repeat("Every word of this information must remain reachable. ", 20)}
			alts := []string{}
			a := New(Config{PhysW: 320, PhysH: 240, Rotation: rot, SafeInsetX: inset, SafeInsetY: inset,
				Alternatives: func(*data.Row) []string { return alts }},
				data.Ingest([]data.Row{row}, "test", time.Now()), nil)
			a.screen = ScreenDetails
			a.Paint()
			before := a.detail.lines
			alts = make([]string, 12)
			a.all = true
			a.Paint()
			if a.detail.lines != before || before < 1 {
				t.Fatalf("rotation=%v inset=%d: alternatives changed visible information from %d to %d", rot, inset, before, a.detail.lines)
			}
			for n := 0; n < 4; n++ {
				old := a.detail.scroll
				a.actDetails(platform.KeyPageDown)
				a.Paint()
				if step := a.detail.scroll - old; step < 0 || step > a.detail.lines {
					t.Fatalf("paging skipped information: step=%d visible=%d", step, a.detail.lines)
				}
				if a.detail.scroll == old {
					break // already at the end of the information
				}
			}
		}
	}
}

func TestDetailsWrapKeepsAllText(t *testing.T) {
	source := "Note:     Some words and averylongunbrokenvaluewithnospace must survive."
	for _, cols := range []int{2, 18, 42, 58} {
		lines := wrapDetailLines([]paneLine{{text: source}}, cols)
		var combined string
		for _, line := range lines {
			if len(line.text) > cols {
				t.Fatalf("%d columns: overflow %q", cols, line.text)
			}
			combined += strings.ReplaceAll(line.text, " ", "")
		}
		if combined != strings.ReplaceAll(source, " ", "") {
			t.Fatalf("%d columns: lost text: %q", cols, combined)
		}
	}
}

func TestNavigationUsesSelectedHoldDelay(t *testing.T) {
	for _, delay := range []int{200, 300, 500} {
		for _, screen := range []Screen{ScreenList, ScreenFilter, ScreenOptions} {
			now := time.Now()
			a := New(Config{PhysW: 320, PhysH: 240, HoldDelay: delay}, data.Ingest(nil, "test", now), nil)
			if screen != ScreenList {
				a.openPanel(screen)
			}
			a.Handle(platform.Event{Key: platform.KeyDown, Pressed: true, At: now})
			a.rep.frameDue(now.Add(time.Duration(delay-1)*time.Millisecond), a.repeatStep)
			if a.rep.count != 0 {
				t.Fatal("repeated before the selected delay")
			}
			a.rep.frameDue(now.Add(time.Duration(delay)*time.Millisecond), a.repeatStep)
			if a.rep.count != 1 {
				t.Fatal("did not repeat when the selected delay elapsed")
			}
		}
	}
}
