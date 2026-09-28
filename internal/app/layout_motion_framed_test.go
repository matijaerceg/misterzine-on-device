package app

import (
	"bytes"
	"image"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/beta"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// wantImages records what the app asks the decoder for.
type wantImages struct {
	*layoutImages
	wants [][]ImageReq
}

func (w *wantImages) Want(reqs []ImageReq) {
	w.wants = append(w.wants, append([]ImageReq(nil), reqs...))
}

// settle ends a framed motion however far it has got.
func settleLayoutMotion(a *App) {
	for a.LayoutTransitionRunning() {
		a.LayoutMotionFrame()
		a.Paint()
	}
}

// In MisterZine Arcade a Select+Y layout change moves one twelfth of the
// way per displayed frame, whatever the wall clock says, and its last frame
// is the settled layout itself.
func TestFramedLayoutMotionSteps(t *testing.T) {
	defer beta.Set(true)()
	for _, size := range [][3]int{{320, 240, 0}, {320, 240, 15}, {320, 240, 40}, {400, 300, 15}, {480, 270, 0}} {
		for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
			a, clock, imgs := layoutTestApp(rot, size[0], size[1], size[2])
			for cycle := 0; cycle < len(listLayouts); cycle++ {
				before := copyLayoutPixels(a.logical.RGBA, a.lay.Body)
				a.cycleListLayout()
				a.Paint()
				if !a.LayoutTransitionRunning() || !a.layoutMotion.framed {
					t.Fatal("no framed motion")
				}
				if !bytes.Equal(before.Pix, copyLayoutPixels(a.logical.RGBA, a.lay.Body).Pix) {
					t.Fatal("first frame flashes destination")
				}
				// the wall clock moves the motion no further
				*clock = clock.Add(time.Second)
				a.Tick(*clock)
				a.Paint()
				if a.layoutMotion.progress != 0 || !a.LayoutTransitionRunning() {
					t.Fatal("the wall clock stepped a framed motion")
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
					if a.lay != destination {
						t.Fatal("paint changed the layout")
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

// Titles and header lines take the words they have at rest on the first
// moving frame and keep them: while the list narrows (list to split, text
// to list) no title or header is shortened again frame after frame, the
// Maker view's pinned header stays on the top line, and no part-row shows
// under the last whole line.
func TestFramedLayoutMotionFitsWordsOnce(t *testing.T) {
	defer beta.Set(true)()
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
func TestFramedLayoutMotionKeepsDecoderQuiet(t *testing.T) {
	defer beta.Set(true)()
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
	settleLayoutMotion(a)
	last := w.wants[len(w.wants)-1]
	if len(last) < 2 {
		t.Fatal("the settled layout did not ask for the neighbours again", last)
	}
}

// Beta off, the stepping is the wall clock's, as in the free build.
func TestFreeLayoutMotionIgnoresFrames(t *testing.T) {
	a, _, _ := layoutTestApp(gfx.RotNone, 320, 240, 15)
	a.cycleListLayout()
	a.Paint()
	if a.layoutMotion.framed || a.LayoutMotionFrame() || a.layoutMotion.progress != 0 {
		t.Fatal("the free build steps by frames")
	}
}

// Pressing Select+Y again mid-motion carries on from the frame on screen.
func TestFramedLayoutMotionRetarget(t *testing.T) {
	defer beta.Set(true)()
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
func TestFramedLayoutMotionKeepsEmptyMessage(t *testing.T) {
	defer beta.Set(true)()
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

// One moving frame of each layout change at 320x240, free and framed: run
// the cross-compiled test binary on a board with -test.bench LayoutMotionStep
// to compare what the stepping costs there.
func BenchmarkLayoutMotionStep(b *testing.B) {
	for _, framed := range []bool{false, true} {
		for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
			name := map[bool]string{false: "free-", true: "framed-"}[framed] + rot.String()
			b.Run(name, func(b *testing.B) {
				defer beta.Set(framed)()
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
}
