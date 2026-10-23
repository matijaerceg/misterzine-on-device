package images

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/fetch"
	"github.com/matijaerceg/misterzine-on-device/internal/media"
)

func waitCachedFrames(t *testing.T, s *Service, keys ...scaledKey) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		ready := true
		for _, k := range keys {
			if s.cache[k] == nil {
				ready = false
				break
			}
		}
		s.mu.Unlock()
		if ready {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("video frame window was not cached")
}

func TestVideoBufferPreparesTwoFramesWithoutReadySignal(t *testing.T) {
	s := New(t.TempDir(), nil, log.New(io.Discard, "", 0), 1<<20)
	defer s.Close()
	p := Pic{Key: "buffer", Slot: fetch.SlotVideo}
	if err := os.WriteFile(s.path(p), testMZV(t,
		color.RGBA{R: 255, A: 255}, color.RGBA{G: 255, A: 255}, color.RGBA{B: 255, A: 255}, color.RGBA{R: 255, G: 255, A: 255}), 0644); err != nil {
		t.Fatal(err)
	}
	req := app.ImageReq{Key: p.Key, Slot: p.Slot, Frame: 0, W: 4, H: 3}
	s.Want([]app.ImageReq{req})
	waitImage(t, s, req)
	for {
		select {
		case <-s.Ready():
		default:
			goto drained
		}
	}
drained:
	first := keyFor(req)
	waitCachedFrames(t, s, first, scaledKey{Pic: p, Frame: 1, W: 4, H: 3}, scaledKey{Pic: p, Frame: 2, W: 4, H: 3})
	select {
	case <-s.Ready():
		t.Fatal("speculative frame caused a repaint")
	default:
	}
	for _, frame := range []int{1, 2} {
		img, state := s.Get(app.ImageReq{Key: p.Key, Slot: p.Slot, Frame: frame, W: 4, H: 3})
		if state != app.ImageReady || img == nil {
			t.Fatalf("upcoming frame %d state=%v", frame, state)
		}
	}
}

func TestVideoBufferWrapsAtClipEnd(t *testing.T) {
	s := New(t.TempDir(), nil, log.New(io.Discard, "", 0), 1<<20)
	defer s.Close()
	p := Pic{Key: "wrap", Slot: fetch.SlotVideo}
	if err := os.WriteFile(s.path(p), testMZV(t, color.RGBA{R: 255, A: 255}, color.RGBA{G: 255, A: 255}, color.RGBA{B: 255, A: 255}), 0644); err != nil {
		t.Fatal(err)
	}
	req := app.ImageReq{Key: p.Key, Slot: p.Slot, Frame: 2, W: 4, H: 3}
	s.Want([]app.ImageReq{req})
	waitImage(t, s, req)
	waitCachedFrames(t, s, scaledKey{Pic: p, Frame: 0, W: 4, H: 3}, scaledKey{Pic: p, Frame: 1, W: 4, H: 3})
}

func TestVideoBufferPriorityAheadOfEarlierScreenshot(t *testing.T) {
	s := New(t.TempDir(), nil, log.New(io.Discard, "", 0), 1<<20)
	s.Close() // exercise scheduler ordering without a competing worker
	video := Pic{Key: "priority", Slot: fetch.SlotVideo}
	still := Pic{Key: "shot", Slot: "snap"}
	data := testMZV(t, color.RGBA{R: 255, A: 255}, color.RGBA{G: 255, A: 255}, color.RGBA{B: 255, A: 255})
	if err := os.WriteFile(s.path(video), data, 0644); err != nil {
		t.Fatal(err)
	}
	var stillData bytes.Buffer
	if err := png.Encode(&stillData, image.NewRGBA(image.Rect(0, 0, 4, 3))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.path(still), stillData.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	clip, err := media.Read(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	videoReq := app.ImageReq{Key: video.Key, Slot: video.Slot, W: 4, H: 3}
	stillReq := app.ImageReq{Key: still.Key, Slot: still.Slot, W: 4, H: 3}
	s.mu.Lock()
	s.rememberClip(video, clip)
	s.put(keyFor(videoReq), image.NewRGBA(image.Rect(0, 0, 4, 3)))
	s.wanted = []scaledKey{keyFor(stillReq), keyFor(videoReq)}
	s.paused = false
	s.mu.Unlock()
	k, ok := s.nextDecode()
	if !ok || k.Pic != video || k.Frame != 1 {
		t.Fatalf("next priority=%+v, ok=%t", k, ok)
	}
}

func TestVideoFutureCompletingPendingGetWakesNextFrame(t *testing.T) {
	s := New(t.TempDir(), nil, log.New(io.Discard, "", 0), 1<<20)
	s.Close() // insert the future frame at a deterministic Get/Want boundary
	p := Pic{Key: "pending", Slot: fetch.SlotVideo}
	data := testMZV(t, color.RGBA{R: 255, A: 255}, color.RGBA{G: 255, A: 255}, color.RGBA{B: 255, A: 255})
	if err := os.WriteFile(s.path(p), data, 0644); err != nil {
		t.Fatal(err)
	}
	clip, err := media.Read(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	currentReq := app.ImageReq{Key: p.Key, Slot: p.Slot, Frame: 0, W: 4, H: 3}
	futureReq := app.ImageReq{Key: p.Key, Slot: p.Slot, Frame: 1, W: 4, H: 3}
	s.mu.Lock()
	s.rememberClip(p, clip)
	s.put(keyFor(currentReq), image.NewRGBA(image.Rect(0, 0, 4, 3)))
	s.wanted = []scaledKey{keyFor(currentReq)}
	s.mu.Unlock()
	if _, state := s.Get(futureReq); state != app.ImageLoading {
		t.Fatalf("preparation Get state=%v", state)
	}

	// Drive just the scheduler-selected lookahead frame synchronously so the
	// cache insertion falls deterministically between Get and Want.
	s.decodeVideo(keyFor(futureReq))
	select {
	case <-s.Ready():
	default:
		t.Fatal("future completion did not wake the pending Get")
	}
	s.Want([]app.ImageReq{futureReq})
	select {
	case <-s.Ready():
	default:
		t.Fatal("Want did not wake after the waiting frame was already cached")
	}
	if img, state := s.Get(futureReq); state != app.ImageReady || img == nil {
		t.Fatalf("next frame state=%v", state)
	}
	for {
		select {
		case <-s.Ready():
		default:
			goto drained
		}
	}
drained:
	s.Want([]app.ImageReq{futureReq})
	select {
	case <-s.Ready():
		t.Fatal("unchanged Want caused a redundant repaint")
	default:
	}
}

func TestVideoBufferDropsFutureWorkOnPauseOrWantReplacement(t *testing.T) {
	for _, pause := range []bool{true, false} {
		s := New(t.TempDir(), nil, log.New(io.Discard, "", 0), 1<<20)
		s.Close() // test selection directly without a competing worker
		p := Pic{Key: "queued", Slot: fetch.SlotVideo}
		clipData := testMZV(t, color.RGBA{R: 255, A: 255}, color.RGBA{G: 255, A: 255}, color.RGBA{B: 255, A: 255})
		if err := os.WriteFile(s.path(p), clipData, 0644); err != nil {
			t.Fatal(err)
		}
		req := app.ImageReq{Key: p.Key, Slot: p.Slot, W: 4, H: 3}
		s.mu.Lock()
		clip, err := media.Read(bytes.NewReader(clipData))
		if err != nil {
			s.mu.Unlock()
			t.Fatal(err)
		}
		s.rememberClip(p, clip)
		s.put(keyFor(req), image.NewRGBA(image.Rect(0, 0, 4, 3)))
		s.wanted = []scaledKey{keyFor(req)}
		s.mu.Unlock()
		if pause {
			s.SetPaused(true)
		} else {
			s.Want([]app.ImageReq{{Key: "other", Slot: fetch.SlotVideo, W: 4, H: 3}})
		}
		if k, ok := s.nextDecode(); ok && k.Pic == p {
			t.Fatal("stale future frame remained queued")
		}
	}
}

func TestVideoBufferSkipsWindowWithOnlyOneSpareFrameAndSameWant(t *testing.T) {
	// One scaled frame takes 48 bytes. The visible frame and one future frame
	// fit, but the complete window does not, so speculation stays disabled.
	s := New(t.TempDir(), nil, log.New(io.Discard, "", 0), 96)
	defer s.Close()
	p := Pic{Key: "small-budget", Slot: fetch.SlotVideo}
	if err := os.WriteFile(s.path(p), testMZV(t, color.RGBA{R: 255, A: 255}, color.RGBA{G: 255, A: 255}, color.RGBA{B: 255, A: 255}), 0644); err != nil {
		t.Fatal(err)
	}
	req := app.ImageReq{Key: p.Key, Slot: p.Slot, W: 4, H: 3}
	s.Want([]app.ImageReq{req})
	waitImage(t, s, req)
	time.Sleep(25 * time.Millisecond)
	s.Want([]app.ImageReq{req})
	s.mu.Lock()
	if len(s.cache) != 1 || s.cache[keyFor(req)] == nil {
		s.mu.Unlock()
		t.Fatalf("cache churned under insufficient budget: entries=%d", len(s.cache))
	}
	if len(s.futureBlocked) != 2 {
		s.mu.Unlock()
		t.Fatalf("capacity failure did not block the complete window: %d keys", len(s.futureBlocked))
	}
	s.mu.Unlock()
	if k, ok := s.nextDecode(); ok && k.Slot == fetch.SlotVideo && k != keyFor(req) {
		t.Fatalf("same Want retried blocked video frame %d", k.Frame)
	}
}
