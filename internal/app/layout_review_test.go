package app

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

func saveLayoutCheck(t *testing.T, a *App, name string) {
	t.Helper()
	if dir := os.Getenv("MZ_RENDER_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, a.Logical()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOverscanLayoutsStayInsideTheirRegions(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
		for _, inset := range []int{15, 30, 40} {
			t.Run(fmt.Sprintf("%v/%d", rot, inset), func(t *testing.T) {
				row := data.Row{K: "game", Title: "A deliberately long game title that takes two lines", Base: "Arcade", Core: "Game", Img: "game", ImgSlots: []string{"snap", "title"}, ImgW: 320, ImgH: 240, Manufacturer: "Example manufacturer", Year: "1985", Genre: "Shooter", Ctl: "8-way 2 buttons", Plr: "2", SN: "game", Rot: "Horizontal", Brot: "upside down", Res: "320x240"}
				a := New(Config{PhysW: 320, PhysH: 240, Rotation: rot, SafeInsetX: inset, SafeInsetY: inset, Alternatives: func(*data.Row) []string { return []string{"a.mra", "b.mra", "c.mra", "d.mra", "e.mra"} }}, data.Ingest([]data.Row{row}, "test", time.Now()), nil)
				a.screen = ScreenDetails
				a.Paint()
				saveLayoutCheck(t, a, fmt.Sprintf("details-%v-%d", rot, inset))
				launchH := a.sm.H + 1
				r := image.Rect(a.lay.Body.Min.X, a.lay.Body.Max.Y-launchH-1, a.lay.Body.Max.X, a.lay.Body.Max.Y)
				pixels := func() []byte {
					var b []byte
					for y := r.Min.Y; y < r.Max.Y; y++ {
						for x := r.Min.X; x < r.Max.X; x++ {
							c := a.Logical().RGBAAt(x, y)
							b = append(b, c.R, c.G, c.B, c.A)
						}
					}
					return b
				}
				before := pixels()
				a.detail.scroll = 4
				a.all = true
				a.Paint()
				if !bytes.Equal(before, pixels()) {
					t.Fatal("scrolling specs changed pixels in Versions region")
				}
				a.screen = ScreenCalibrate
				a.all = true
				a.Paint()
				saveLayoutCheck(t, a, fmt.Sprintf("calibrate-%v-%d", rot, inset))
				for y := 0; y < a.Logical().Rect.Dy(); y++ {
					for x := 0; x < a.Logical().Rect.Dx(); x++ {
						if !image.Pt(x, y).In(a.lay.Root) && a.Logical().RGBAAt(x, y) != gen.Eva.Bg {
							t.Fatalf("calibration draws outside safe frame at %d,%d", x, y)
						}
					}
				}
				a.logical.Fill(a.logical.Rect, gen.Eva.Bg)
				a.filters.FavOnly = true
				a.ds.Rows = make([]data.Row, 1208)
				a.view = make([]int, 905)
				a.paintStatus(a.logical)
				for y := a.lay.Status.Min.Y; y < a.lay.Status.Max.Y; y++ {
					for x := 0; x < a.Logical().Rect.Dx(); x++ {
						if (x < a.lay.Status.Min.X || x >= a.lay.Status.Max.X) && a.Logical().RGBAAt(x, y) != gen.Eva.Bg {
							t.Fatalf("status text escapes safe frame at %d,%d", x, y)
						}
					}
				}
			})
		}
	}
}

func TestFirstVisitCannotEnableSinceFilter(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest([]data.Row{{K: "game"}}, "test", time.Now()), nil)
	openExpandedFilters(a)
	explained := false
	for i, e := range a.panel.entries {
		if e.text == "available after your first visit" && e.info {
			explained = true
		}
		if e.kind == "since" {
			a.panel.cursor = i
			a.togglePanel()
			if a.filters.Since {
				t.Fatal("first visit enabled an empty Since filter")
			}
		}
	}
	if !explained {
		t.Fatal("missing first-visit explanation")
	}
	a.filters.Since = true
	if a.emptyListMessage() != "no previous visit yet" {
		t.Fatal("saved Since filter gives misleading empty message")
	}
}

func TestFinishedUpdateDoesNotRepaintUntilChanged(t *testing.T) {
	for _, status := range []string{"completed", "failed", "errors", "cancelled", "interrupted", "restarted"} {
		t.Run(status, func(t *testing.T) {
			a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(nil, "test", time.Now()), nil)
			s := updater.State{ID: "run", Status: status, Lines: []string{"finished"}}
			a.SetUpdate(s, true)
			a.Paint()
			for i := 0; i < 4; i++ {
				a.SetUpdate(s, false)
				a.Tick(time.Now().Add(time.Duration(i) * time.Second))
				if _, dirty := a.Paint(); dirty != nil {
					t.Fatal("unchanged result repainted")
				}
			}
			s.Message = "new information"
			a.SetUpdate(s, false)
			if _, dirty := a.Paint(); dirty == nil {
				t.Fatal("changed result did not repaint")
			}
			s.Status = "running"
			a.SetUpdate(s, false)
			a.Paint()
			a.Tick(time.Now().Add(time.Second))
			if _, dirty := a.Paint(); dirty == nil {
				t.Fatal("active spinner stopped animating")
			}
		})
	}
}
