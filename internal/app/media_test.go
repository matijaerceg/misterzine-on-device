package app

import (
	"image"
	"image/color"
	"reflect"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

type mediaImages struct {
	state   ImageState
	pending bool
	wants   []ImageReq
	gets    []ImageReq
}

func (m *mediaImages) Get(r ImageReq) (*image.RGBA, ImageState) {
	m.gets = append(m.gets, r)
	if r.Slot == "video" && (m.state != ImageReady || m.pending && r.Frame > 0) {
		return nil, m.state
	}
	im := image.NewRGBA(image.Rect(0, 0, r.W, r.H))
	im.SetRGBA(0, 0, color.RGBA{R: uint8(r.Frame + 10), A: 255})
	return im, ImageReady
}
func (m *mediaImages) Want(r []ImageReq) { m.wants = append([]ImageReq(nil), r...) }
func (m *mediaImages) SetPaused(bool)    {}

func mediaApp(t *testing.T, preference string) (*App, *mediaImages, *time.Time) {
	t.Helper()
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	m := &mediaImages{}
	r := data.Row{Base: "Arcade", K: "game", Title: "Game", Img: "shot", ImgSlots: []string{"snap", "title"}, ImgW: 320, ImgH: 240, MediaKey: "media", MediaSlots: []string{"video", "flyer"}}
	a := New(Config{PhysW: 320, PhysH: 240, Images: m, ListShot: preference, TimerNow: func() time.Time { return now }, Now: func() time.Time { return now }, TransitionsDisabled: true, SaverDisabled: true}, data.Ingest([]data.Row{r}, "", now), nil)
	a.Paint()
	return a, m, &now
}

func TestMediaSelectionAndFallback(t *testing.T) {
	a, m, _ := mediaApp(t, "video")
	r, _, _ := a.current()
	for _, pref := range []string{"video", "flyer"} {
		key, slot := thumbSlot(r, pref)
		if key != "media" || slot != pref {
			t.Fatalf("%s: %s/%s", pref, key, slot)
		}
	}
	if got := shotSlots(r); !reflect.DeepEqual(got, []string{"snap", "title", "video", "flyer"}) {
		t.Fatal(got)
	}
	if stretchShot(r, "flyer") || stretchShot(r, "video") {
		t.Fatal("extra media must preserve its own aspect")
	}
	m.state = ImageOffline
	m.gets = nil
	a.Invalidate()
	a.Paint()
	if len(m.gets) < 2 || m.gets[0].Slot != "video" || m.gets[1].Slot != "snap" {
		t.Fatalf("offline fallback: %+v", m.gets)
	}
	if !a.nextVideoTick().IsZero() {
		t.Fatal("offline video is spinning the timer")
	}
	if a.video.last != nil {
		t.Fatal("failed clip retained a frozen frame")
	}
	r.MediaSlots = nil
	if key, slot := thumbSlot(r, "flyer"); key != "shot" || slot != "snap" {
		t.Fatalf("unadvertised fallback: %s/%s", key, slot)
	}
}

func TestVideoTimelineHoldsPendingFrameAndStopsOffscreen(t *testing.T) {
	a, m, now := mediaApp(t, "video")
	first := a.video.last
	if first == nil || a.nextVideoTick().IsZero() {
		t.Fatal("preview did not start")
	}
	m.pending = true
	*now = now.Add(videoInterval)
	if !a.tickVideo(*now) {
		t.Fatal("no video tick")
	}
	_, dirty := a.Paint()
	if len(dirty) == 0 || a.video.last != first || !a.video.next.IsZero() {
		t.Fatal("pending decode must retain last frame and wait for Ready")
	}
	if len(m.wants) != 1 || m.wants[0].Frame != 1 {
		t.Fatalf("pending frame lost: %+v", m.wants)
	}
	m.pending = false
	a.Invalidate()
	a.Paint()
	if a.video.last == first || a.video.next.IsZero() {
		t.Fatal("decoded frame did not resume playback")
	}
	for i := 0; i < 20; i++ {
		*now = now.Add(videoInterval)
		a.tickVideo(*now)
		a.Paint()
	}
	if len(m.wants) != 1 {
		t.Fatalf("obsolete frame backlog: %d", len(m.wants))
	}
	a.screen = ScreenOptions
	if !a.nextVideoTick().IsZero() {
		t.Fatal("video kept ticking behind Options")
	}
	a.screen = ScreenList
	a.saver.active = true
	if !a.nextVideoTick().IsZero() {
		t.Fatal("video kept ticking under screensaver")
	}
}

func TestDetailsAndFullscreenMedia(t *testing.T) {
	a, m, now := mediaApp(t, "video")
	a.actList(platform.KeyEnter)
	a.Paint()
	if a.screen != ScreenDetails || !a.video.visible {
		t.Fatal("no details preview")
	}
	*now = now.Add(videoInterval)
	a.tickVideo(*now)
	a.Paint()
	if a.video.req.Frame != 1 {
		t.Fatal("details preview did not animate")
	}
	a.actDetails(platform.KeyEnter)
	a.Paint()
	if a.screen != ScreenShot || a.slot != 2 || !a.video.center {
		t.Fatal("artwork did not open preferred video")
	}
	a.actShot(platform.KeyRight)
	a.Paint()
	if a.slot != 3 || a.video.visible || !a.nextVideoTick().IsZero() {
		t.Fatal("flyer still animates")
	}
	last := m.gets[len(m.gets)-1]
	if last.Key != "media" || last.Slot != "flyer" || last.Native || last.Stretch {
		t.Fatalf("flyer request: %+v", last)
	}
	a.actShot(platform.KeyBack)
	if a.screen != ScreenDetails {
		t.Fatal("Back lost details")
	}
}

func TestVideoFailureDuringPartialPaintFallsBack(t *testing.T) {
	a, m, now := mediaApp(t, "video")
	m.state = ImageMissing
	m.gets = nil
	*now = now.Add(videoInterval)
	a.tickVideo(*now)
	a.Paint()
	if a.video.last != nil || !a.video.next.IsZero() {
		t.Fatal("failed frame kept playing")
	}
	found := false
	for _, r := range m.gets {
		found = found || r.Slot == "snap"
	}
	if !found {
		t.Fatal("failed video did not request screenshot fallback")
	}
}

func TestArtTypeCycleAndVideoPrivacy(t *testing.T) {
	a, _, _ := mediaApp(t, "gameplay")
	for _, want := range []string{"title", "video", "flyer", "gameplay"} {
		a.cycleListShot()
		if a.ListShot() != want {
			t.Fatalf("got %s want %s", a.ListShot(), want)
		}
	}
	a.cfg.ListShot = "video"
	a.cfg.Mature = MaturePixelate
	a.ds.Rows[0].Mat = 1
	a.Paint()
	if !a.video.req.Pixelate {
		t.Fatal("video bypassed mature pixelation")
	}
}
