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

// wantImages records what the app asks the decoder for.
type wantImages struct {
	*layoutImages
	wants [][]ImageReq
}

func (w *wantImages) Want(reqs []ImageReq) {
	w.wants = append(w.wants, append([]ImageReq(nil), reqs...))
}

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

// settleLayoutMotion steps the layout motion frame by frame to its end.
func settleLayoutMotion(t testing.TB, a *App) {
	t.Helper()
	for frames := 0; a.LayoutTransitionRunning(); frames++ {
		if frames > 2*layoutMotionSteps {
			t.Fatal("the layout motion did not settle")
		}
		a.LayoutMotionFrame()
		a.Paint()
	}
}

// A Select+Y layout change moves one twelfth of the way per displayed
// frame, whatever the wall clock says, asks for the destination picture
// size alone, and its last frame is the settled layout itself.
func TestLayoutMotionSteps(t *testing.T) {
	for _, size := range [][3]int{{320, 240, 0}, {320, 240, 15}, {320, 240, 40}, {400, 300, 15}, {480, 270, 0}, {640, 360, 20}} {
		for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
			a, clock, imgs := layoutTestApp(rot, size[0], size[1], size[2])
			for cycle := 0; cycle < len(listLayouts); cycle++ {
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
				if a.NextTick().After(*clock) {
					t.Fatal("motion not scheduled")
				}
				// the wall clock moves the motion no further
				*clock = clock.Add(time.Second)
				a.Tick(*clock)
				a.Paint()
				if a.layoutMotion.progress != 0 || !a.LayoutTransitionRunning() {
					t.Fatal("the wall clock stepped the motion")
				}
				destination, n := a.lay, len(imgs.requests)
				for step := 1; step < layoutMotionSteps; step++ {
					if !a.LayoutMotionFrame() {
						t.Fatal("step refused", step)
					}
					if a.LayoutMotionFrame() {
						t.Fatal("stepped twice before a paint")
					}
					if got, want := a.layoutMotion.progress, float64(step)/float64(layoutMotionSteps); got != want {
						t.Fatalf("step %d at %v, want %v", step, got, want)
					}
					if _, dirty := a.Paint(); len(dirty) == 0 {
						t.Fatal("step not presented", step)
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
				if !a.LayoutMotionFrame() || a.LayoutTransitionRunning() {
					t.Fatal("the last step did not settle")
				}
				a.Paint()
				end := append([]byte(nil), a.logical.Pix...)
				a.Invalidate()
				a.Paint()
				if !bytes.Equal(end, a.logical.Pix) {
					t.Fatal("last frame differs from the settled layout", rot, size, a.ListLayout())
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
		settleLayoutMotion(t, a)
	}
}

// Optional visual artifact: all frames of each transition, in both orientations.
func TestLayoutMotionPreview(t *testing.T) {
	dir := os.Getenv("MZ_LAYOUT_PREVIEW")
	if dir == "" {
		t.Skip("set MZ_LAYOUT_PREVIEW to export frames")
	}
	every := 1
	if os.Getenv("MZ_LAYOUT_PREVIEW_FPS") == "30" {
		every = 2
	}
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		a, _, _ := layoutTestApp(rot, 320, 240, 0)
		// Settle the selected row before recording so the loop closes
		// seamlessly: a full round of the layouts ends back at the first.
		for range listLayouts {
			a.cycleListLayout()
			settleLayoutMotion(t, a)
		}
		for cycle := range listLayouts {
			a.cycleListLayout()
			for step := 0; step <= layoutMotionSteps; step++ {
				if step > 0 {
					a.LayoutMotionFrame()
				}
				a.Paint()
				if step%every != 0 {
					continue
				}
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				f, err := os.Create(filepath.Join(dir, fmt.Sprintf("r%d-c%d-f%02d.png", rot, cycle, step/every)))
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

// One moving frame of each layout change at 320x240: run the cross-compiled
// test binary on a board with -test.bench LayoutMotionStep to see what a
// step costs there.
func BenchmarkLayoutMotionStep(b *testing.B) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		b.Run(rot.String(), func(b *testing.B) {
			a, _, _ := layoutTestApp(rot, 320, 240, 15)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if i%11 == 0 {
					b.StopTimer()
					a.layoutMotion, a.all = layoutMotion{}, true // settle at once
					a.Paint()
					a.cycleListLayout()
					a.Paint()
					b.StartTimer()
				}
				a.layoutMotion.progress = float64(1+i%11) / 12
				a.layoutMotion.dirty = true
				a.Paint()
			}
		})
	}
}

func TestLayoutMotionLoadingDestinationKeepsCaptionGeometry(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		a, _, imgs := layoutTestApp(rot, 320, 240, 0)
		a.cycleListLayout()
		a.Paint()
		settleLayoutMotion(t, a)
		imgs.missing = true
		a.cycleListLayout()
		a.Paint()
		targetThumb, targetText := a.layoutMotion.toThumb, a.layoutMotion.toText
		if a.layoutMotion.art == nil {
			t.Fatal("lost visible artwork while target loads")
		}
		imgs.missing = false
		settleLayoutMotion(t, a)
		if targetThumb != a.paintedThumb || targetText != a.paintedPaneText {
			t.Fatalf("loading target moved captions at completion: %v %v -> %v %v", targetThumb, targetText, a.paintedThumb, a.paintedPaneText)
		}
	}
}

func TestLayoutDividerVisibleThroughMotion(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
		a, _, _ := layoutTestApp(rot, 320, 240, 0)
		for cycle := 0; cycle < 3; cycle++ {
			a.cycleListLayout()
			a.Paint()
			for frame := 1; frame < 6; frame++ {
				a.LayoutMotionFrame()
				a.Paint()
				d := a.layoutMotion.currentDivider
				if d[0].X != d[1].X || d[1].Y != d[2].Y {
					t.Fatal("divider lost its connected corner")
				}
				if d[0] == d[2] {
					t.Fatal("divider collapsed")
				}
				// on the way to the text layout it slides off the body's edge
				inside := 0
				for y := d[0].Y; y <= d[1].Y; y++ {
					if !image.Pt(d[0].X, y).In(a.lay.Body) {
						continue
					}
					inside++
					if got := a.logical.RGBAAt(d[0].X, y); got != gen.Eva.Line {
						t.Fatal("vertical divider disappeared", rot, cycle, frame, y)
					}
				}
				for x := d[1].X; x <= d[2].X; x++ {
					if !image.Pt(x, d[2].Y).In(a.lay.Body) {
						continue
					}
					inside++
					if got := a.logical.RGBAAt(x, d[2].Y); got != gen.Eva.Line {
						t.Fatal("horizontal divider disappeared", rot, cycle, frame, x)
					}
				}
				if inside == 0 && a.layoutMotion.progress < 0.9 {
					t.Fatal("divider left the body early", rot, cycle, frame)
				}
			}
			from := a.layoutMotion.currentDivider
			a.cycleListLayout()
			a.Paint()
			if a.layoutMotion.currentDivider != from {
				t.Fatal("divider jumped on repeated press")
			}
			settleLayoutMotion(t, a)
		}
	}
}

// Titles and header lines take the words they have at rest on the first
// moving frame and keep them: while the list narrows (list to split, text
// to list) no title or header is shortened again frame after frame, the
// Maker view's pinned header stays on the top line, and no part-row shows
// under the last whole line.
func TestLayoutMotionFitsWordsOnce(t *testing.T) {
	for _, mode := range []data.SortMode{data.SortUpdated, data.SortMaker, data.SortYear} {
		a, clock, _ := layoutTestApp(gfx.RotNone, 320, 240, 15)
		a.SetSort(mode)
		a.cursor = 24
		a.ensureVisible()
		a.Paint()
		*clock = clock.Add(time.Second) // the page change to the view ends
		a.Tick(*clock)
		a.Paint()
		if mode == data.SortMaker && a.pinnedHeader() == "" {
			t.Fatal("no pinned header to keep")
		}
		layoutWordsOnce(t, a)
	}
}

func layoutWordsOnce(t *testing.T, a *App) {
	for _, want := range []string{"split", "picture", "text", "list"} {
		a.cycleListLayout()
		a.Paint()
		var frames [][]byte
		for a.LayoutTransitionRunning() {
			a.LayoutMotionFrame()
			a.Paint()
			frames = append(frames, append([]byte(nil), a.logical.Pix...))
		}
		if a.ListLayout() != want {
			t.Fatal("layout", a.ListLayout())
		}
		if want != "split" && want != "list" {
			continue // the rows travel up or down to or from the top pane
		}
		settled := frames[len(frames)-1]
		l := a.lay
		// the titles' column at rest, over every whole line and the gap below
		column := image.Rect(l.List.Min.X, l.List.Min.Y, l.List.Min.X+a.body.W+l.TitleW, l.List.Max.Y)
		for i, f := range frames[:len(frames)-1] {
			for y := column.Min.Y; y < column.Max.Y; y++ {
				o := a.logical.PixOffset(column.Min.X, y)
				if !bytes.Equal(f[o:o+column.Dx()*4], settled[o:o+column.Dx()*4]) {
					t.Fatalf("%v to %s, frame %d line y=%d: the list changed words or showed a part-row before it settled", a.Sort(), want, i+1, y)
				}
			}
		}
	}
}

// While the layout moves, the decoder is asked for the destination picture
// alone; a picture that lands does not repaint the screen but is taken into
// the motion on its next frame, and the settled layout brings the
// neighbours' prefetch back.
func TestLayoutMotionKeepsDecoderQuiet(t *testing.T) {
	a, _, imgs := layoutTestApp(gfx.RotNone, 320, 240, 15)
	for i := range a.ds.Rows {
		a.ds.Rows[i].Img = a.ds.Rows[i].K // a picture of its own for every row
	}
	w := &wantImages{layoutImages: imgs}
	a.cfg.Images = w
	imgs.missing = true
	a.cycleListLayout()
	a.Paint()
	if len(w.wants) != 1 || len(w.wants[0]) != 1 || w.wants[0][0] != a.layoutMotion.request {
		t.Fatalf("decoder asked for %v, want the destination %v alone", w.wants, a.layoutMotion.request)
	}
	for step := 0; step < 3; step++ {
		a.LayoutMotionFrame()
		a.Paint()
	}
	if a.layoutMotion.restArt {
		t.Fatal("took a picture that was not there")
	}
	imgs.missing = false
	a.Invalidate()
	if a.all {
		t.Fatal("a landed picture repaints everything mid-motion")
	}
	a.LayoutMotionFrame()
	a.Paint()
	if m := &a.layoutMotion; !m.restArt || m.art != imgs.cache[m.request] {
		t.Fatal("the landed destination picture was not taken")
	}
	settleLayoutMotion(t, a)
	last := w.wants[len(w.wants)-1]
	if len(last) < 2 {
		t.Fatal("the settled layout did not ask for the neighbours again", last)
	}
}

// Pressing Select+Y again mid-motion carries on from the frame on screen.
func TestLayoutMotionRetarget(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		a, _, _ := layoutTestApp(rot, 320, 240, 15)
		a.cycleListLayout()
		a.Paint()
		for step := 0; step < 5; step++ {
			a.LayoutMotionFrame()
			a.Paint()
		}
		before := copyLayoutPixels(a.logical.RGBA, a.lay.Body)
		a.cycleListLayout()
		a.Paint()
		if !bytes.Equal(before.Pix, copyLayoutPixels(a.logical.RGBA, a.lay.Body).Pix) {
			t.Fatal("interrupted frame jumped")
		}
		steps := 0
		for a.LayoutTransitionRunning() {
			a.LayoutMotionFrame()
			a.Paint()
			steps++
		}
		if steps != layoutMotionSteps || a.ListLayout() != "picture" {
			t.Fatal("retarget took", steps, "frames to", a.ListLayout())
		}
	}
}

// An empty list keeps its message through the motion, so the last frame
// does not bring it in.
func TestLayoutMotionKeepsEmptyMessage(t *testing.T) {
	a, clock, _ := layoutTestApp(gfx.RotLeft, 320, 240, 15)
	a.SetData(data.Ingest(nil, "test", *clock), nil)
	a.Invalidate()
	a.Paint()
	a.cycleListLayout()
	a.Paint()
	for step := 0; step < layoutMotionSteps/2; step++ {
		a.LayoutMotionFrame()
		a.Paint()
	}
	muted := 0
	line := a.layoutMotion.current.lineRect(1)
	for y := line.Min.Y; y < line.Max.Y; y++ {
		for x := line.Min.X; x < line.Max.X; x++ {
			if a.logical.RGBAAt(x, y) == gen.Eva.Muted {
				muted++
			}
		}
	}
	if muted == 0 {
		t.Fatal("the empty-list message left during the motion")
	}
}
