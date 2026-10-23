package images

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/fetch"
	"github.com/matijaerceg/misterzine-on-device/internal/media"
)

func testMZV(t *testing.T, colors ...color.RGBA) []byte {
	t.Helper()
	var frames [][]byte
	for _, c := range colors {
		img := image.NewRGBA(image.Rect(0, 0, 4, 3))
		for y := 0; y < 3; y++ {
			for x := 0; x < 4; x++ {
				img.SetRGBA(x, y, c)
			}
		}
		var b bytes.Buffer
		if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: 90}); err != nil {
			t.Fatal(err)
		}
		frames = append(frames, b.Bytes())
	}
	var out bytes.Buffer
	out.WriteString("MZV1")
	for _, n := range []uint16{4, 3, 15, uint16(len(frames))} {
		binary.Write(&out, binary.LittleEndian, n)
	}
	for _, frame := range frames {
		binary.Write(&out, binary.LittleEndian, uint32(len(frame)))
		out.Write(frame)
	}
	return out.Bytes()
}

func waitImage(t *testing.T, s *Service, req app.ImageReq) *image.RGBA {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if img, state := s.Get(req); state == app.ImageReady {
			return img
		}
		if time.Now().After(deadline) {
			_, state := s.Get(req)
			t.Fatalf("image did not become ready; state=%v", state)
		}
		select {
		case <-s.Ready():
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func TestVideoFrameSelectionNormalizationAndCache(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, nil, log.New(io.Discard, "", 0), 1<<20)
	defer s.Close()
	p := Pic{Key: "demo", Slot: fetch.SlotVideo}
	clip := testMZV(t, color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255})
	if err := os.WriteFile(s.path(p), clip, 0644); err != nil {
		t.Fatal(err)
	}
	req := app.ImageReq{Key: p.Key, Slot: p.Slot, Frame: 3, W: 4, H: 3}
	s.Want([]app.ImageReq{req})
	img := waitImage(t, s, req)
	r, _, b, _ := img.At(1, 1).RGBA()
	if b <= r {
		t.Fatal("frame 3 did not wrap to blue frame 1")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.raw[p] != nil {
		t.Fatal("decoded video frame retained in the still-image raw cache")
	}
	wantKey := keyFor(req)
	wantKey.Frame = 1
	if _, ok := s.cache[wantKey]; !ok || len(s.clips) != 1 {
		t.Fatalf("normalized frame or compressed clip not cached: frame=%d clips=%d", wantKey.Frame, len(s.clips))
	}
}

func TestVideoDownloadsThenDecodesInitialFrameRequest(t *testing.T) {
	clip := testMZV(t, color.RGBA{R: 255, A: 255}, color.RGBA{G: 255, A: 255})
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Write(clip)
	}))
	defer srv.Close()
	old := fetch.Site
	fetch.Site = srv.URL
	defer func() { fetch.Site = old }()
	s := New(t.TempDir(), fetch.NewClient("test"), log.New(io.Discard, "", 0), 1<<20)
	defer s.Close()
	req := app.ImageReq{Key: "remote", Slot: fetch.SlotVideo, Frame: 7, W: 4, H: 3}
	if _, state := s.Get(req); state != app.ImageLoading {
		t.Fatalf("initial state=%v", state)
	}
	s.Want([]app.ImageReq{req})
	img := waitImage(t, s, req)
	r, g, _, _ := img.At(1, 1).RGBA()
	if g <= r || path != "/images/video/remote.mzv" {
		t.Fatalf("wrong downloaded frame or path: r=%d g=%d path=%q", r, g, path)
	}
}

func TestVideoPauseAndOfflineStates(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, nil, log.New(io.Discard, "", 0), 1<<20)
	defer s.Close()
	p := Pic{Key: "paused", Slot: fetch.SlotVideo}
	if err := os.WriteFile(s.path(p), testMZV(t, color.RGBA{R: 255, A: 255}), 0644); err != nil {
		t.Fatal(err)
	}
	req := app.ImageReq{Key: p.Key, Slot: p.Slot, W: 4, H: 3}
	s.SetPaused(true)
	s.Want([]app.ImageReq{req})
	time.Sleep(30 * time.Millisecond)
	if _, state := s.Get(req); state != app.ImageLoading {
		t.Fatalf("paused state=%v", state)
	}
	s.mu.Lock()
	decoded := len(s.clips)
	s.mu.Unlock()
	if decoded != 0 {
		t.Fatal("clip decoded while paused")
	}
	s.SetPaused(false)
	waitImage(t, s, req)
	if _, state := s.Get(app.ImageReq{Key: "absent", Slot: fetch.SlotVideo, W: 4, H: 3}); state != app.ImageOffline {
		t.Fatalf("uncached nil-client video state=%v", state)
	}
}

func TestCompressedVideoCacheKeepsTwoClips(t *testing.T) {
	s := New(t.TempDir(), nil, log.New(io.Discard, "", 0), 1<<20)
	defer s.Close()
	data := testMZV(t, color.RGBA{R: 255, A: 255})
	for _, key := range []string{"one", "two", "three"} {
		p := Pic{Key: key, Slot: fetch.SlotVideo}
		if err := os.WriteFile(s.path(p), data, 0644); err != nil {
			t.Fatal(err)
		}
		req := app.ImageReq{Key: key, Slot: fetch.SlotVideo, W: 4, H: 3}
		s.Want([]app.ImageReq{req})
		waitImage(t, s, req)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.clips) != 2 || s.clips[Pic{Key: "one", Slot: fetch.SlotVideo}] != nil {
		t.Fatalf("clip cache contains %d clips; oldest was not evicted", len(s.clips))
	}
}

func TestCorruptVideoIsRemovedAndReportedMissing(t *testing.T) {
	s := New(t.TempDir(), nil, log.New(io.Discard, "", 0), 1<<20)
	defer s.Close()
	p := Pic{Key: "corrupt", Slot: fetch.SlotVideo}
	if err := os.WriteFile(s.path(p), []byte("not an mzv"), 0644); err != nil {
		t.Fatal(err)
	}
	req := app.ImageReq{Key: p.Key, Slot: p.Slot, W: 4, H: 3}
	s.Want([]app.ImageReq{req})
	select {
	case <-s.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("corrupt video was not reported")
	}
	if _, state := s.Get(req); state != app.ImageMissing {
		t.Fatalf("state=%v", state)
	}
	if _, err := os.Stat(s.path(p)); !os.IsNotExist(err) {
		t.Fatalf("corrupt clip was not removed: %v", err)
	}
}

func TestOversizedLocalVideoIsRejected(t *testing.T) {
	s := New(t.TempDir(), nil, log.New(io.Discard, "", 0), 1<<20)
	defer s.Close()
	p := Pic{Key: "oversized", Slot: fetch.SlotVideo}
	f, err := os.Create(s.path(p))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(media.MaxBytes + 1); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()
	req := app.ImageReq{Key: p.Key, Slot: p.Slot, W: 4, H: 3}
	s.Want([]app.ImageReq{req})
	select {
	case <-s.Ready():
	case <-time.After(3 * time.Second):
		t.Fatal("oversized video was not rejected")
	}
	if _, state := s.Get(req); state != app.ImageMissing {
		t.Fatalf("state=%v", state)
	}
	if _, err := os.Stat(s.path(p)); !os.IsNotExist(err) {
		t.Fatalf("oversized clip was not removed: %v", err)
	}
}

func TestInvalidMediaKeyNeverQueuesOrEscapes(t *testing.T) {
	s := New(t.TempDir(), fetch.NewClient("test"), log.New(io.Discard, "", 0), 1<<20)
	defer s.Close()
	for _, slot := range []string{fetch.SlotVideo, fetch.SlotFlyer} {
		req := app.ImageReq{Key: "../escape", Slot: slot, W: 4, H: 3}
		if _, state := s.Get(req); state != app.ImageMissing {
			t.Fatalf("%s state=%v", slot, state)
		}
		s.Want([]app.ImageReq{req})
		s.mu.Lock()
		wanted := len(s.wanted)
		s.mu.Unlock()
		if wanted != 0 {
			t.Fatalf("invalid %s request was queued", slot)
		}
	}
}

func TestMalformedAndOversizedFlyersAreRemoved(t *testing.T) {
	s := New(t.TempDir(), nil, log.New(io.Discard, "", 0), 1<<20)
	defer s.Close()
	var big bytes.Buffer
	if err := png.Encode(&big, image.NewRGBA(image.Rect(0, 0, media.MaxFlyerDimension+1, 1))); err != nil {
		t.Fatal(err)
	}
	for key, data := range map[string][]byte{"malformed": []byte("not png"), "too-wide": big.Bytes()} {
		p := Pic{Key: key, Slot: fetch.SlotFlyer}
		if err := os.WriteFile(s.path(p), data, 0644); err != nil {
			t.Fatal(err)
		}
		req := app.ImageReq{Key: key, Slot: fetch.SlotFlyer, W: 40, H: 40}
		s.Want([]app.ImageReq{req})
		select {
		case <-s.Ready():
		case <-time.After(2 * time.Second):
			t.Fatalf("%s flyer was not rejected", key)
		}
		if _, state := s.Get(req); state != app.ImageMissing {
			t.Fatalf("%s state=%v", key, state)
		}
		if _, err := os.Stat(s.path(p)); !os.IsNotExist(err) {
			t.Fatalf("%s flyer was not removed: %v", key, err)
		}
	}
}

func TestLocalVideoSupport(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(dir+"/video", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/video/local.mzv", testMZV(t, color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255}), 0644); err != nil {
		t.Fatal(err)
	}
	l := NewLocal(dir)
	img, state := l.Get(app.ImageReq{Key: "local", Slot: fetch.SlotVideo, Frame: 5, W: 4, H: 3})
	if state != app.ImageReady {
		t.Fatalf("state=%v", state)
	}
	r, _, b, _ := img.At(1, 1).RGBA()
	if b <= r || l.raw[Pic{Key: "local", Slot: fetch.SlotVideo}] != nil {
		t.Fatal("local provider selected the wrong frame or retained it as raw")
	}
	if _, state := l.Get(app.ImageReq{Key: "local", Slot: fetch.SlotVideo, Frame: 0, W: 4, H: 3}); state != app.ImageReady {
		t.Fatalf("second frame state=%v", state)
	}
	if len(l.cache) != 1 {
		t.Fatalf("local video retained %d scaled frames for one size", len(l.cache))
	}
}

func TestFailedClipDoesNotServeEarlierCachedFrames(t *testing.T) {
	s := New(t.TempDir(), nil, log.New(io.Discard, "", 0), 1<<20)
	defer s.Close()
	req := app.ImageReq{Key: "failed", Slot: fetch.SlotVideo, W: 4, H: 3}
	s.mu.Lock()
	s.put(keyFor(req), image.NewRGBA(image.Rect(0, 0, 4, 3)))
	s.failed[keyFor(req).Pic] = true
	s.mu.Unlock()
	if img, state := s.Get(req); img != nil || state != app.ImageMissing {
		t.Fatalf("corrupt clip still served a cached frame: %v", state)
	}
}
