package app

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestHeldListScroll(t *testing.T) {
	for _, rotation := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
		for _, grouped := range []bool{false, true} {
			for _, speed := range []string{"20", "30", "60"} {
				for _, direction := range []platform.Key{platform.KeyDown, platform.KeyUp} {
					t.Run(fmt.Sprint(rotation, grouped, speed, direction), func(t *testing.T) {
						now := time.Now()
						rows := make([]data.Row, 100)
						for i := range rows {
							rows[i] = data.Row{K: fmt.Sprint(i), Title: fmt.Sprintf("%c Game %03d", rune(65+i/4), i), Base: "Arcade"}
						}
						a := New(Config{PhysW: 320, PhysH: 240, Rotation: rotation, TimerNow: func() time.Time { return now }}, data.Ingest(rows, "", now), nil)
						if grouped {
							a.SetSort(data.SortAlphabetical)
						}
						a.cfg.Scroll = speed
						a.cursor = 40
						if grouped {
							if direction == platform.KeyDown {
								a.cursor = 42
							} else {
								a.cursor = 41
							}
						}
						a.top = centeredTop(a.screenLine(a.cursor), a.totalLines(), a.lay.Lines)
						a.Paint()
						// A tap must remain immediate.
						a.act(direction)
						if a.ListScrollRunning() {
							t.Fatal("tap animated")
						}
						oldTop := a.top
						a.rep.press(direction, now)
						now = now.Add(time.Second)
						a.Frame(now)
						frames := a.rep.every
						delta := (a.top - oldTop) * a.lay.Line
						if delta == 0 {
							t.Fatal("fixture did not scroll")
						}
						if want := delta * (frames - 1) / frames; a.listMotion.offset != want {
							t.Fatalf("offset %d want %d", a.listMotion.offset, want)
						}
						a.Paint()
						cursor := a.cursor
						a.rep.release(direction)
						for i := 1; i < frames; i++ {
							now = now.Add(frameDur)
							a.ListScrollFrame(now)
							a.Paint()
							partial := append([]byte(nil), a.logical.Pix...)
							a.all = true
							a.Paint()
							if !bytes.Equal(partial, a.logical.Pix) {
								t.Fatal("partial paint differs from full paint")
							}
							if want := delta * (frames - i - 1) / frames; a.listMotion.offset != want {
								t.Fatalf("release offset %d want %d", a.listMotion.offset, want)
							}
						}
						if a.ListScrollRunning() || a.cursor != cursor {
							t.Fatal("release advanced selection or failed to settle")
						}
					})
				}
			}
		}

	}
}

func TestListScrollCancellation(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(nil, "", time.Now()), nil)
	a.listMotion = listMotion{offset: 8, frames: 2}
	a.act(platform.KeyBack)
	if a.ListScrollRunning() || a.listMotion.offset != 0 {
		t.Fatal("navigation retained motion")
	}
}
