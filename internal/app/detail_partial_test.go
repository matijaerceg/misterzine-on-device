package app

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestDetailPartialPaintMatchesFull(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
		clock := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
		calls := 0
		a := New(Config{PhysW: 320, PhysH: 240, Rotation: rot, TimerNow: func() time.Time { return clock }, Exists: func(string) bool { calls++; return true }}, data.Ingest([]data.Row{{K: "game", Title: "Game", Base: "Arcade", MRA: "Game.mra", Note: strings.Repeat("Some lengthy information to scroll. ", 60)}}, "", clock), nil)
		a.screen = ScreenDetails
		a.Paint()
		for _, key := range []platform.Key{platform.KeyDown, platform.KeyUp, platform.KeyDown} {
			a.actDetails(key)
			for n := 0; n < 8; n++ {
				clock = clock.Add(17 * time.Millisecond)
				a.DetailScrollFrame(clock)
				before := calls
				frame, dirty := a.Paint()
				if calls != before {
					t.Fatal("scroll queried launch files")
				}
				area := 0
				for _, r := range dirty {
					area += r.Dx() * r.Dy()
				}
				if area >= frame.Rect.Dx()*frame.Rect.Dy() {
					t.Fatal("scroll repainted whole screen")
				}
				want := append([]byte(nil), frame.Pix...)
				a.all = true
				full, _ := a.Paint()
				if !bytes.Equal(want, full.Pix) {
					t.Fatalf("rot=%v key=%v frame=%d partial differs", rot, key, n)
				}
			}
		}
	}
}

func TestDetailMarqueePartialPaintMatchesFull(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		clock := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
		calls := 0
		a := New(Config{PhysW: 320, PhysH: 240, Rotation: rot, TimerNow: func() time.Time { return clock }, Alternatives: func(*data.Row) []string { calls++; return []string{strings.Repeat("Long version name ", 8) + ".mra"} }}, data.Ingest([]data.Row{{K: "game", Title: "Game", Base: "Arcade", Note: strings.Repeat("Information. ", 80)}}, "", clock), nil)
		a.screen = ScreenDetails
		a.Paint()
		for n := 0; n < 10; n++ {
			clock = clock.Add(100 * time.Millisecond)
			before := calls
			a.tickMarquee(clock)
			frame, dirty := a.Paint()
			if calls != before {
				t.Fatal("marquee queried versions")
			}
			area := 0
			for _, r := range dirty {
				area += r.Dx() * r.Dy()
			}
			if area >= frame.Rect.Dx()*frame.Rect.Dy() {
				t.Fatal("marquee repainted whole screen")
			}
			want := append([]byte(nil), frame.Pix...)
			a.all = true
			full, _ := a.Paint()
			if !bytes.Equal(want, full.Pix) {
				t.Fatalf("rot=%v frame=%d marquee differs", rot, n)
			}
		}
	}
}
