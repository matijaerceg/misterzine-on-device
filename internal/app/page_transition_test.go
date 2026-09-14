package app

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestPageWipeLayering(t *testing.T) {
	for _, back := range []bool{false, true} {
		const w, h = 320, 240
		from, incoming := image.NewRGBA(image.Rect(0, 0, w, h)), image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(from, from.Rect, image.NewUniform(color.RGBA{255, 0, 0, 255}), image.Point{}, draw.Src)
		draw.Draw(incoming, incoming.Rect, image.NewUniform(color.RGBA{0, 0, 255, 255}), image.Point{}, draw.Src)
		dst := make([]byte, len(from.Pix))
		previous := make([]byte, w*h)
		for step := 0; step <= 10; step++ {
			copy(dst, incoming.Pix)
			composePageWipe(dst, from.Pix, w, h, from.Stride, float64(step)/10, back)
			if step == 0 && !bytes.Equal(dst, from.Pix) {
				t.Fatal("outgoing frame changed before wipe started")
			}
			if step == 10 && !bytes.Equal(dst, incoming.Pix) {
				t.Fatal("outgoing pixels remained at completion")
			}
			for i := range previous {
				blue := dst[i*4+2]
				if blue < previous[i] {
					t.Fatal("reveal moved backwards")
				}
				previous[i] = blue
			}
			if step == 5 {
				left, right := dst[(h/2)*from.Stride+2], dst[(h/2)*from.Stride+(w-1)*4+2]
				if (!back && (left != 0 || right != 255)) || (back && (left != 255 || right != 0)) {
					t.Fatal("wrong reveal direction")
				}
				for y := 0; y < h; y++ {
					changes := 0
					previous := dst[y*from.Stride+2]
					for x := 1; x < w; x++ {
						pixel := dst[y*from.Stride+x*4+2]
						if pixel != 0 && pixel != 255 {
							t.Fatal("dissolve blurred a page pixel")
						}
						if pixel != previous {
							changes++
						}
						previous = pixel
					}
					if changes < 3 || changes > 66 {
						t.Fatalf("soft dissolve missing or too broad: %d changes", changes)
					}
				}
			}
		}
	}
}

func TestPageTransitionInputAndCompletion(t *testing.T) {
	for _, frame := range []bool{false, true} {
		a, clock := saverApp()
		a.EnablePageTransitions()
		a.Paint()
		before := append([]byte(nil), a.logical.Pix...)
		tap := func(k platform.Key) {
			a.Handle(platform.Event{Key: k, Pressed: true, At: *clock})
			a.Handle(platform.Event{Key: k, At: *clock})
		}
		tap(platform.KeyBack)
		a.Paint()
		if a.screen != ScreenOptions || !a.PageTransitionRunning() || a.transition.back {
			t.Fatal("opening Options did not start forward wipe")
		}
		if !bytes.Equal(before, a.logical.Pix) {
			t.Fatal("first frame flashed incoming content")
		}
		if a.NextTick().After(clock.Add(frameDur)) {
			t.Fatal("animation not scheduled")
		}
		*clock = clock.Add(75 * time.Millisecond)
		if frame {
			a.Frame(*clock)
		} else {
			a.Tick(*clock)
		}
		a.Paint()
		halfway := append([]byte(nil), a.logical.Pix...)
		if bytes.Equal(halfway, before) {
			t.Fatal("wipe did not advance")
		}
		tap(platform.KeyBack)
		a.Paint()
		if a.screen != ScreenList || !a.transition.back {
			t.Fatal("Back was blocked during the wipe")
		}
		if !bytes.Equal(halfway, a.logical.Pix) {
			t.Fatal("interruption flashed an intermediate frame")
		}
		*clock = clock.Add(pageWipeDuration)
		if frame {
			a.Frame(*clock)
		} else {
			a.Tick(*clock)
		}
		a.Paint()
		if a.PageTransitionRunning() || !a.transition.next.IsZero() {
			t.Fatal("wipe did not end")
		}
		final := append([]byte(nil), a.logical.Pix...)
		a.all = true
		a.Paint()
		if !bytes.Equal(final, a.logical.Pix) {
			t.Fatal("wipe left stale pixels")
		}
		tap(platform.KeyDown)
		a.Paint()
		if a.PageTransitionRunning() {
			t.Fatal("list movement animated")
		}
		tap(platform.KeyRight)
		a.Paint()
		if a.PageTransitionRunning() {
			t.Fatal("list paging animated")
		}
		tap(platform.KeySpace)
		a.Paint()
		if !a.PageTransitionRunning() {
			t.Fatal("view switch did not animate")
		}
	}
}

func TestPageTransitionSaverAndResize(t *testing.T) {
	a, clock := saverApp()
	a.EnablePageTransitions()
	a.Paint()
	a.openOptions()
	a.Paint()
	a.saver.active = true
	a.all = true
	a.Paint()
	if a.PageTransitionRunning() {
		t.Fatal("saver retained page wipe")
	}
	a.saver.active = false
	a.all = true
	a.Paint()
	if a.PageTransitionRunning() {
		t.Fatal("wake replayed page wipe")
	}
	a.actPanel(platform.KeyBack)
	a.Paint()
	*clock = clock.Add(pageWipeDuration)
	a.Tick(*clock)
	a.Paint()
	a.SetRotation(1)
	a.Paint()
	if a.PageTransitionRunning() {
		t.Fatal("resize animated incompatible frames")
	}
}

// Optional visual artifact for reviewing actual app frames at five points.
func TestPageTransitionPreview(t *testing.T) {
	path := os.Getenv("PAGE_WIPE_PREVIEW")
	if path == "" {
		t.Skip("no preview requested")
	}
	a, clock := saverApp()
	a.EnablePageTransitions()
	a.Paint()
	a.openOptions()
	w, h := a.logical.W(), a.logical.H()
	sheet := image.NewRGBA(image.Rect(0, 0, w*5, h*2))
	for row := 0; row < 2; row++ {
		if row == 1 {
			a.actPanel(platform.KeyBack)
		}
		a.Paint()
		start := *clock
		for col := 0; col < 5; col++ {
			*clock = start.Add(time.Duration(col) * pageWipeDuration / 4)
			a.Tick(*clock)
			a.Paint()
			draw.Draw(sheet, image.Rect(col*w, row*h, (col+1)*w, (row+1)*h), a.logical.RGBA, image.Point{}, draw.Src)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, sheet); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkPageWipe(b *testing.B) {
	from := make([]byte, 320*240*4)
	dst := make([]byte, len(from))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		composePageWipe(dst, from, 320, 240, 320*4, 0.5, false)
	}
}

func TestPageTransitionCachedDestinationRefreshes(t *testing.T) {
	a, clock := saverApp()
	a.EnablePageTransitions()
	a.Paint()
	a.openOptions()
	a.Paint()
	incoming := append([]byte(nil), a.transition.to...)
	*clock = clock.Add(40 * time.Millisecond)
	a.Tick(*clock)
	a.Paint()
	composed := append([]byte(nil), a.logical.Pix...)
	a.Paint()
	if !bytes.Equal(composed, a.logical.Pix) {
		t.Fatal("cached frame blended twice")
	}
	a.Handle(platform.Event{Key: platform.KeyDown, Pressed: true, At: *clock})
	a.Handle(platform.Event{Key: platform.KeyDown, At: *clock})
	a.Paint()
	if bytes.Equal(incoming, a.transition.to) {
		t.Fatal("input failed to refresh the destination")
	}
	refreshed := append([]byte(nil), a.transition.to...)
	*clock = clock.Add(30 * time.Millisecond)
	a.Tick(*clock)
	a.Paint()
	if !bytes.Equal(refreshed, a.transition.to) {
		t.Fatal("animation modified its cached destination")
	}
}

func TestPageWipeDirtyFramesMatchFullRotation(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
		a, clock := saverApp()
		a.SetRotation(rot)
		a.EnablePageTransitions()
		a.Paint()
		for trip := 0; trip < 2; trip++ {
			if trip == 0 {
				a.openOptions()
			} else {
				a.actPanel(platform.KeyBack)
			}
			_, dirty := a.Paint()
			if len(dirty) != 0 {
				t.Fatal("initial outgoing frame was needlessly copied")
			}
			start := *clock
			for _, ms := range []int{17, 36, 52, 71, 89, 108, 126, 144, 167, 185, 200} {
				*clock = start.Add(time.Duration(ms) * time.Millisecond)
				a.Tick(*clock)
				physical, dirty := a.Paint()
				reference := image.NewRGBA(physical.Rect)
				gfx.RotateAll(reference, a.logical.RGBA, rot)
				if !bytes.Equal(physical.Pix, reference.Pix) {
					t.Fatalf("rot %v trip %d time %d: stale pixels outside dirty strip", rot, trip, ms)
				}
				area := 0
				for _, r := range dirty {
					area += r.Dx() * r.Dy()
				}
				if area >= physical.Rect.Dx()*physical.Rect.Dy() {
					t.Fatal("cached animation copied the whole screen")
				}
			}
		}
	}
}

func TestPageWipeDefersBackgroundInvalidation(t *testing.T) {
	a, clock := saverApp()
	a.EnablePageTransitions()
	a.Paint()
	a.openOptions()
	a.Paint()
	a.Invalidate()
	if a.all || !a.transition.pending {
		t.Fatal("background work interrupted wipe")
	}
	*clock = clock.Add(pageWipeDuration)
	a.Tick(*clock)
	a.Paint()
	if a.PageTransitionRunning() || a.transition.next.IsZero() {
		t.Fatal("deferred refresh not scheduled after completion")
	}
	*clock = clock.Add(frameDur)
	a.Tick(*clock)
	if !a.all {
		t.Fatal("deferred refresh was lost")
	}
	a.Paint()
}

type pagePauseImages struct {
	noImages
	paused bool
}

func (p *pagePauseImages) SetPaused(paused bool) { p.paused = paused }
func TestPageWipePausesAndResumesPictures(t *testing.T) {
	a, clock := saverApp()
	pics := &pagePauseImages{}
	a.cfg.Images = pics
	a.EnablePageTransitions()
	a.Paint()
	a.openOptions()
	a.Paint()
	if !pics.paused {
		t.Fatal("decoder competed with transition")
	}
	*clock = clock.Add(pageWipeDuration)
	a.Tick(*clock)
	a.Paint()
	if pics.paused {
		t.Fatal("decoder remained paused after transition")
	}
	a.actPanel(platform.KeyBack)
	a.Paint()
	a.saver.active = true
	a.all = true
	a.Paint()
	if pics.paused {
		t.Fatal("cancelling for saver left decoder paused")
	}
}
