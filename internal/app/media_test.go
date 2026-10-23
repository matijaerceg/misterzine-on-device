package app

import (
	"image"
	"image/color"
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

func TestFlyersOnly(t *testing.T) {
	a, m, _ := mediaApp(t, "video")
	if a.ListShot() != "gameplay" {
		t.Fatal("legacy video preference did not fall back")
	}
	for _, want := range []string{"title", "flyer", "gameplay"} {
		a.cycleListShot()
		if a.ListShot() != want {
			t.Fatal(a.ListShot())
		}
	}
	a.cfg.ListShot = "flyer"
	a.Invalidate()
	a.Paint()
	if len(m.gets) == 0 || m.gets[len(m.gets)-1].Slot != "flyer" {
		t.Fatal("flyer unavailable")
	}
	a.actList(platform.KeyEnter)
	a.Paint()
	a.actDetails(platform.KeyEnter)
	a.Paint()
	if a.video.visible || !a.nextVideoTick().IsZero() {
		t.Fatal("video active")
	}
	for _, req := range m.gets {
		if req.Slot == "video" {
			t.Fatal("requested shelved video")
		}
	}
	row, _, _ := a.current()
	for _, slot := range shotSlots(row) {
		if slot == "video" {
			t.Fatal("video in detail slots")
		}
	}
	if a.screen != ScreenShot {
		t.Fatal("flyer fullscreen unavailable")
	}
}
