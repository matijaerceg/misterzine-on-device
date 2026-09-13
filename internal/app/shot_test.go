package app

import (
	"bytes"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestArtworkStartLaunchesSelectedVersionAndShowsFailure(t *testing.T) {
	for _, installed := range []bool{true, false} {
		now := time.Unix(100, 0)
		calls := 0
		target := ""
		a := New(Config{PhysW: 320, PhysH: 240, SafeInsetX: 40, SafeInsetY: 40,
			Now:          func() time.Time { return now },
			Launch:       func(p string) { calls++; target = p },
			Exists:       func(string) bool { return installed },
			Status:       func(int) data.Status { return data.StatusCurrent },
			Alternatives: func(*data.Row) []string { return []string{"_Arcade/Alternative.mra"} },
		}, data.Ingest([]data.Row{{K: "game", Title: "Game", Base: "Arcade", Core: "game", MRA: "_Arcade/Game.mra"}}, "", now), nil)
		a.actList(platform.KeyEnter)
		pick := 0
		if installed {
			a.actDetails(platform.KeyRight)
			pick = 1
		}
		a.actDetails(platform.KeyEnter)
		a.Paint()
		before := append([]byte(nil), a.Logical().Pix...)
		a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: now})
		a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: now})
		a.Tick(now.Add(time.Second))
		a.Handle(platform.Event{Key: platform.KeyStart, At: now})
		a.Paint()
		if a.Screen() != ScreenShot || a.detail.pick != pick {
			t.Fatal("Start lost artwork/version context")
		}
		if installed {
			if calls != 1 || target != "_Arcade/Alternative.mra" {
				t.Fatalf("launches=%d target=%q", calls, target)
			}
		} else {
			if calls != 0 || a.notice == "" || bytes.Equal(before, a.Logical().Pix) {
				t.Fatal("missing target did not show a visible failure in artwork")
			}
		}
	}
}

func TestShotReturnPreservesSelectedVersion(t *testing.T) {
	for _, rotation := range []struct {
		name string
		rot  gfx.Rotation
	}{{"horizontal", gfx.RotNone}, {"left", gfx.RotLeft}, {"right", gfx.RotRight}} {
		for _, exit := range []struct {
			name string
			key  platform.Key
		}{{"B", platform.KeyBack}, {"A", platform.KeyEnter}, {"X", platform.KeyTab}} {
			t.Run(rotation.name+"/"+exit.name, func(t *testing.T) {
				now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
				row := data.Row{
					K: "game", Title: "Example Game", Base: "Arcade", Core: "Example",
					MRA: "_Arcade/Example.mra", Updated: "2026-09-08", Date: "2020-01-01",
					Img: "example", ImgSlots: []string{"snap", "title"}, ImgW: 320, ImgH: 240,
					Rot: "Horizontal", Brot: "upside down", Genre: "Shooter", Manufacturer: "Example",
					SN: "example", Year: "1985", Reg: "World", Scr: 2, Res: "320x240", Plr: "2",
					Ctl: "8-way Â· 2 buttons", Spc: "Example controls", Flip: "yes", Act: "2026-09-08",
					Src: "official", Deprecated: true, Beta: true, Note: "Example note",
				}
				alts := []string{"_Arcade/_alternatives/Example A.mra", "_Arcade/_alternatives/Example B.mra"}
				launched := ""
				a := New(Config{
					PhysW: 320, PhysH: 240, Rotation: rotation.rot, SafeInsetX: 15, SafeInsetY: 15,
					Now: func() time.Time { return now }, ClockTrusted: true,
					Status:       func(int) data.Status { return data.StatusCurrent },
					Exists:       func(string) bool { return true },
					Alternatives: func(*data.Row) []string { return alts },
					Launch:       func(path string) { launched = path },
				}, data.Ingest([]data.Row{row}, "test", now), nil)
				tap := func(key platform.Key) {
					a.Handle(platform.Event{Key: key, Pressed: true, At: now})
					a.Handle(platform.Event{Key: key, At: now.Add(time.Millisecond)})
					a.Paint()
					now = now.Add(50 * time.Millisecond) // past the bounce guard
				}
				tap(platform.KeyEnter)
				tap(platform.KeyRight)
				tap(platform.KeyRight)
				tap(platform.KeyDown)
				scroll := a.detail.scroll
				if a.detail.pick != 2 || scroll <= 0 {
					t.Fatal("fixture must select an alternative and scroll its details")
				}
				now = now.Add(time.Second)
				tap(platform.KeyEnter)
				if a.Screen() != ScreenShot {
					t.Fatal("A did not open artwork")
				}
				tap(platform.KeyRight)
				tap(exit.key)
				if exit.key != platform.KeyBack {
					if a.Screen() != ScreenShot || a.slot != 1 {
						t.Fatal("A/X changed or closed artwork")
					}
					tap(platform.KeyBack)
				}
				if a.Screen() != ScreenDetails || a.detail.pick != 2 || a.detail.scroll != scroll {
					t.Errorf("return lost details state: screen=%v pick=%d scroll=%d (wanted %d)", a.Screen(), a.detail.pick, a.detail.scroll, scroll)
				}
				// A is artwork now: even an immediate new press should work.
				tap(platform.KeyEnter)
				if a.Screen() != ScreenShot || a.slot != 0 {
					t.Fatal("immediate A did not reopen the first shot")
				}
				tap(platform.KeyBack)
				tap(platform.KeyStart)
				if launched != alts[1] {
					t.Fatalf("Start launched %q instead of selected version %q", launched, alts[1])
				}
				// A separate visit from the list reopens on the remembered version
				// with the information back at the top.
				tap(platform.KeyBack)
				tap(platform.KeyEnter)
				if a.detail.pick != 2 || a.detail.scroll != 0 {
					t.Fatalf("a fresh details visit: pick=%d scroll=%d, wanted the remembered version 2 at the top", a.detail.pick, a.detail.scroll)
				}
			})
		}
	}
}

func TestRotationKeysTurnImageInPressedDirection(t *testing.T) {
	for _, tc := range []struct {
		key   platform.Key
		want  gfx.Rotation
		label string
	}{
		{platform.KeyLeft, gfx.RotRight, "monitor CW"},
		{platform.KeyRight, gfx.RotLeft, "monitor CCW"},
	} {
		a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(nil, "test", time.Now()), nil)
		a.openPanel(ScreenOptions)
		for i, entry := range a.panel.entries {
			if entry.kind == "rotation" {
				a.panel.cursor = i
				break
			}
		}
		a.actPanel(tc.key)
		if a.Rotation() != tc.want {
			t.Fatalf("%v selected %v", tc.key, a.Rotation())
		}
		e := a.panel.entries[a.panel.cursor]
		if e.vals[e.idx] != tc.label {
			t.Fatalf("monitor label=%q", e.vals[e.idx])
		}
	}
}
