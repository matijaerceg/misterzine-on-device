package app

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

type layoutImages struct {
	requests []ImageReq
	missing  bool
	cache    map[ImageReq]*image.RGBA
}

func (f *layoutImages) Get(r ImageReq) (*image.RGBA, ImageState) {
	f.requests = append(f.requests, r)
	if f.missing {
		return nil, ImageLoading
	}
	if f.cache == nil {
		f.cache = make(map[ImageReq]*image.RGBA)
	}
	if v := f.cache[r]; v != nil {
		return v, ImageReady
	}
	w, h := r.W, r.H
	if !r.Stretch {
		w, h = r.W, 320*r.W/240
		if h > r.H {
			h = r.H
			w = 240 * h / 320
		}
	}
	v := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			col := color.RGBA{24, 24, 65, 255}
			if ((x*12/max(w, 1))+(y*16/max(h, 1)))%3 == 0 {
				col = color.RGBA{52, 170, 200, 255}
			}
			if x > w/3 && x < w*2/3 && y > h/3 && y < h*2/3 {
				col = color.RGBA{230, 160, 36, 255}
			}
			v.SetRGBA(x, y, col)
		}
	}
	f.cache[r] = v
	return v, ImageReady
}
func (*layoutImages) Want([]ImageReq) {}
func (*layoutImages) SetPaused(bool)  {}

func layoutTestApp(rot gfx.Rotation, w, h, inset int) (*App, *time.Time, *layoutImages) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	rows := make([]data.Row, 50)
	for i := range rows {
		rows[i] = data.Row{K: fmt.Sprint(i), Title: fmt.Sprintf("Game %02d: A long arcade title", i), Base: "Arcade", MRA: "game.mra", Img: "game", ImgSlots: []string{"snap"}, ImgW: 240, ImgH: 320, Year: "1984", Manufacturer: "Arcade Company", Genre: "Shooter", Updated: "2026-09-01"}
	}
	imgs := &layoutImages{}
	a := New(Config{PhysW: w, PhysH: h, Rotation: rot, SafeInsetX: inset, SafeInsetY: inset, Now: func() time.Time { return now }, Images: imgs}, data.Ingest(rows, "test", now), nil)
	a.cursor = 24
	a.ensureVisible()
	a.EnablePageTransitions()
	a.Paint()
	return a, &now, imgs
}

func TestLayoutMotionFramesAndCompletion(t *testing.T) {
	for _, size := range [][3]int{{320, 240, 0}, {320, 240, 40}, {480, 270, 0}, {640, 360, 20}} {
		for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
			a, clock, imgs := layoutTestApp(rot, size[0], size[1], size[2])
			for cycle := 0; cycle < 3; cycle++ {
				key := a.CursorKey()
				before := copyLayoutPixels(a.logical.RGBA, a.lay.Body)
				a.cycleListLayout()
				a.Paint()
				if !a.LayoutTransitionRunning() || a.PageTransitionRunning() {
					t.Fatal("wrong animation")
				}
				if !bytes.Equal(before.Pix, copyLayoutPixels(a.logical.RGBA, a.lay.Body).Pix) {
					t.Fatal("first frame flashes destination")
				}
				if a.NextTick().After(clock.Add(frameDur)) {
					t.Fatal("motion not scheduled")
				}
				destination := a.lay
				n := len(imgs.requests)
				for frame := 1; frame <= 12; frame++ {
					*clock = clock.Add(layoutMotionDuration / 12)
					a.Tick(*clock)
					_, dirty := a.Paint()
					if len(dirty) == 0 {
						t.Fatal("frame not presented")
					}
					if a.lay != destination || a.CursorKey() != key {
						t.Fatal("paint changed navigation")
					}
					for _, req := range a.wants {
						if req.W != destination.Thumb.Dx() || req.H != destination.Thumb.Dy() {
							t.Fatal("requested intermediate image size", req)
						}
					}
				}
				if len(imgs.requests) != n {
					t.Fatal("image provider consulted per frame")
				}
				*clock = clock.Add(time.Millisecond)
				a.Tick(*clock)
				a.Paint()
				if a.LayoutTransitionRunning() {
					t.Fatal("did not finish")
				}
				end := append([]byte(nil), a.logical.Pix...)
				a.Invalidate()
				a.Paint()
				if !bytes.Equal(end, a.logical.Pix) {
					t.Fatal("final frame differs from static layout")
				}
			}
		}
	}
}

func TestLayoutMotionInterruptions(t *testing.T) {
	for _, action := range []string{"again", "navigate", "art", "resize", "rotate", "disabled", "saver"} {
		t.Run(action, func(t *testing.T) {
			a, clock, _ := layoutTestApp(gfx.RotNone, 320, 240, 0)
			a.cycleListLayout()
			a.Paint()
			*clock = clock.Add(65 * time.Millisecond)
			a.Frame(*clock)
			a.Paint()
			before := copyLayoutPixels(a.logical.RGBA, a.lay.Body)
			switch action {
			case "again":
				a.cycleListLayout()
			case "navigate":
				a.act(platform.KeyEnter)
			case "art":
				a.cycleListShot()
			case "resize":
				a.SetCanvasSize(480, 270)
			case "rotate":
				a.SetRotation(gfx.RotLeft)
			case "disabled":
				a.cfg.TransitionsDisabled = true
			case "saver":
				a.saver.active = true
			}
			a.Paint()
			if action == "again" {
				if !a.LayoutTransitionRunning() || a.ListLayout() != "picture" {
					t.Fatal("repeat queued or lost")
				}
				if !bytes.Equal(before.Pix, copyLayoutPixels(a.logical.RGBA, a.lay.Body).Pix) {
					t.Fatal("interrupted frame jumped")
				}
			} else if a.LayoutTransitionRunning() {
				t.Fatal("animation survived incompatible change")
			}
		})
	}
}

func TestLayoutMotionEmptyMissingAndDisabled(t *testing.T) {
	for _, kind := range []string{"empty", "missing", "disabled", "static"} {
		a, clock, imgs := layoutTestApp(gfx.RotLeft, 320, 240, 40)
		switch kind {
		case "empty":
			a.SetData(data.Ingest(nil, "test", *clock), nil)
		case "missing":
			imgs.missing = true
		case "disabled":
			a.cfg.TransitionsDisabled = true
		case "static":
			a.transition.enabled = false
		}
		a.Invalidate()
		a.Paint()
		a.cycleListLayout()
		a.Paint()
		if (kind == "disabled" || kind == "static") && a.LayoutTransitionRunning() {
			t.Fatal("ignored disabled animation")
		}
		*clock = clock.Add(100 * time.Millisecond)
		a.Tick(*clock)
		a.Paint()
		*clock = clock.Add(100 * time.Millisecond)
		a.Tick(*clock)
		a.Paint()
		if a.LayoutTransitionRunning() {
			t.Fatal("did not finish")
		}
	}
}

// Optional visual artifact: all frames of each transition, in both orientations.
func TestLayoutMotionPreview(t *testing.T) {
	dir := os.Getenv("MZ_LAYOUT_PREVIEW")
	if dir == "" {
		t.Skip("set MZ_LAYOUT_PREVIEW to export frames")
	}
	frames := 12
	if os.Getenv("MZ_LAYOUT_PREVIEW_FPS") == "30" {
		frames = 6
	}
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		a, clock, _ := layoutTestApp(rot, 320, 240, 0)
		// Settle the selected row before recording so the loop closes seamlessly.
		for i := 0; i < 3; i++ {
			a.cycleListLayout()
			*clock = clock.Add(layoutMotionDuration)
			a.Tick(*clock)
			a.Paint()
		}
		for cycle := 0; cycle < 3; cycle++ {
			a.cycleListLayout()
			start := *clock
			for frame := 0; frame <= frames; frame++ {
				if frame > 0 {
					*clock = start.Add(time.Duration(frame) * layoutMotionDuration / time.Duration(frames))
					a.Tick(*clock)
				}
				a.Paint()
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				f, err := os.Create(filepath.Join(dir, fmt.Sprintf("r%d-c%d-f%02d.png", rot, cycle, frame)))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(f, a.logical.RGBA)
				f.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

func BenchmarkLayoutMotionFrame(b *testing.B) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		b.Run(rot.String(), func(b *testing.B) {
			a, _, _ := layoutTestApp(rot, 640, 360, 0)
			a.cycleListLayout()
			a.Paint()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				a.layoutMotion.progress = float64(1+i%11) / 12
				a.layoutMotion.dirty = true
				a.Paint()
			}
		})
	}
}

func TestLayoutMotionLoadingDestinationKeepsCaptionGeometry(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		a, clock, imgs := layoutTestApp(rot, 320, 240, 0)
		a.cycleListLayout()
		a.Paint()
		*clock = clock.Add(layoutMotionDuration)
		a.Tick(*clock)
		a.Paint()
		imgs.missing = true
		a.cycleListLayout()
		a.Paint()
		targetThumb, targetText := a.layoutMotion.toThumb, a.layoutMotion.toText
		if a.layoutMotion.art == nil {
			t.Fatal("lost visible artwork while target loads")
		}
		imgs.missing = false
		*clock = clock.Add(layoutMotionDuration)
		a.Tick(*clock)
		a.Paint()
		if targetThumb != a.paintedThumb || targetText != a.paintedPaneText {
			t.Fatalf("loading target moved captions at completion: %v %v -> %v %v", targetThumb, targetText, a.paintedThumb, a.paintedPaneText)
		}
	}
}

func TestLayoutDividerVisibleThroughMotion(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
		a, clock, _ := layoutTestApp(rot, 320, 240, 0)
		for cycle := 0; cycle < 3; cycle++ {
			a.cycleListLayout()
			a.Paint()
			for frame := 1; frame < 6; frame++ {
				*clock = clock.Add(30 * time.Millisecond)
				a.Tick(*clock)
				a.Paint()
				d := a.layoutMotion.currentDivider
				if d[0].X != d[1].X || d[1].Y != d[2].Y {
					t.Fatal("divider lost its connected corner")
				}
				if d[0] == d[2] {
					t.Fatal("divider collapsed")
				}
				for y := d[0].Y; y <= d[1].Y; y++ {
					if got := a.logical.RGBAAt(d[0].X, y); got != gen.Eva.Line {
						t.Fatal("vertical divider disappeared", rot, cycle, frame, y)
					}
				}
				for x := d[1].X; x <= d[2].X; x++ {
					if got := a.logical.RGBAAt(x, d[2].Y); got != gen.Eva.Line {
						t.Fatal("horizontal divider disappeared", rot, cycle, frame, x)
					}
				}
			}
			from := a.layoutMotion.currentDivider
			a.cycleListLayout()
			a.Paint()
			if a.layoutMotion.currentDivider != from {
				t.Fatal("divider jumped on repeated press")
			}
			*clock = clock.Add(layoutMotionDuration)
			a.Tick(*clock)
			a.Paint()
		}
	}
}
