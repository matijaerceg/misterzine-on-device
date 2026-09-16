package app

import (
	"image"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

func TestLiveCanvasKeepsNavigation(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
		var requested string
		a := New(Config{PhysW: 480, PhysH: 270, Canvas: "full", Rotation: rot, RememberSort: true, Action: func(kind, arg string) {
			if kind == "canvas" {
				requested = arg
			}
		}}, data.Ingest([]data.Row{{K: "game", Base: "Arcade", Title: "Game"}}, "test", time.Now()), nil)
		a.screen = ScreenOptions
		a.buildPanel()
		expandOptionsForTest(a)
		for i, e := range a.panel.entries {
			if e.kind == "canvas" {
				a.panel.cursor = i
			}
		}
		a.EnablePageTransitions()
		a.Paint()
		a.stepValue(-1)
		if requested != "320x240" {
			t.Fatalf("host not notified: %q", requested)
		}
		for _, size := range [][2]int{{320, 240}, {360, 270}, {480, 270}, {320, 240}} {
			key := a.CursorKey()
			images := &pagePauseImages{paused: true}
			a.cfg.Images = images
			a.SetCanvasSize(size[0], size[1])
			if images.paused {
				t.Fatal("image loading left paused after resize")
			}
			if a.Screen() != ScreenOptions || a.panel.entries[a.panel.cursor].kind != "canvas" || a.CursorKey() != key || a.rot != rot {
				t.Fatal("navigation lost")
			}
			frame, dirty := a.Paint()
			if frame.Rect != image.Rect(0, 0, size[0], size[1]) || len(dirty) == 0 {
				t.Fatalf("bad physical frame %v dirty%v", frame.Rect, dirty)
			}
			w, h := size[0], size[1]
			if rot.Rotated() {
				w, h = h, w
			}
			if a.Logical().Rect != image.Rect(0, 0, w, h) {
				t.Fatal("wrong rotated canvas")
			}
		}
		requested = ""
		a.RestoreCanvasChoice("full")
		if requested != "" || a.Canvas() != "full" || a.panel.entries[a.panel.cursor].idx != 2 {
			t.Fatal("rollback requested another switch or lost option")
		}
	}
}
