package app

import (
	"image"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/fonts"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// The horizontal picture layout keeps its text in one column: a horizontal
// shot, a vertical one and a loading one all leave the text where it was,
// and the vertical shot sits centred in its box.
func TestPictureTextColumn(t *testing.T) {
	for _, c := range []struct{ w, h, inset int }{{320, 240, 0}, {320, 240, 15}, {320, 240, 40}, {360, 270, 15}, {400, 300, 15}, {400, 300, 40}, {420, 262, 15}} {
		l := NewLayout(c.w, c.h, c.inset, c.inset, fonts.Body(), fonts.NarrowTall().W, 5, "picture")
		if !l.PaneTop || !l.ArtCentered || l.TextBeside {
			t.Fatalf("%dx%d inset %d: top %v centred %v beside %v", c.w, c.h, c.inset, l.PaneTop, l.ArtCentered, l.TextBeside)
		}
		if !l.PaneText.In(l.Pane) || l.PaneText.Min.X != l.Thumb.Max.X+4 || l.PaneText.Dx() < 60 {
			t.Fatalf("%dx%d inset %d: text column %v beside picture %v in pane %v", c.w, c.h, c.inset, l.PaneText, l.Thumb, l.Pane)
		}
		rows := []data.Row{
			{K: "wide", Title: "Wide", Base: "Arcade", MRA: "w.mra", Img: "wide", ImgSlots: []string{"snap"}, ImgW: 320, ImgH: 240, Updated: "2026-09-03"},
			{K: "tall", Title: "Tall", Base: "Arcade", MRA: "t.mra", Img: "tall", ImgSlots: []string{"snap"}, ImgW: 240, ImgH: 320, Updated: "2026-09-02"},
			{K: "slow", Title: "Slow", Base: "Arcade", MRA: "s.mra", Img: "slow", ImgSlots: []string{"snap"}, ImgW: 240, ImgH: 320, Updated: "2026-09-01"},
		}
		imgs := &columnImages{loading: "slow"}
		now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
		a := New(Config{PhysW: c.w, PhysH: c.h, SafeInsetX: c.inset, SafeInsetY: c.inset, ListLayout: "picture", Now: func() time.Time { return now }, Images: imgs}, data.Ingest(rows, "test", now), nil)
		texts := map[string]image.Rectangle{}
		for _, key := range []string{"wide", "tall", "slow"} {
			a.MoveToKey(key)
			a.Invalidate()
			a.Paint()
			texts[key] = a.paintedPaneText
			if !a.paintedThumb.In(a.lay.Thumb) {
				t.Fatalf("%s drawn %v outside its box %v", key, a.paintedThumb, a.lay.Thumb)
			}
			if key == "tall" {
				left, right := a.paintedThumb.Min.X-a.lay.Thumb.Min.X, a.lay.Thumb.Max.X-a.paintedThumb.Max.X
				if left < right-1 || left > right+1 || a.paintedThumb.Dx() >= a.lay.Thumb.Dx() {
					t.Fatalf("%dx%d: vertical shot %v not centred in %v", c.w, c.h, a.paintedThumb, a.lay.Thumb)
				}
			}
		}
		if texts["wide"] != texts["tall"] || texts["tall"] != texts["slow"] || texts["wide"] != a.lay.PaneText {
			t.Fatalf("%dx%d inset %d: the text moved: %v", c.w, c.h, c.inset, texts)
		}
	}
}

// columnImages draws every picture at once, except one that stays loading.
type columnImages struct{ loading string }

func (f *columnImages) Get(r ImageReq) (*image.RGBA, ImageState) {
	if r.Key == f.loading {
		return nil, ImageLoading
	}
	w, h := r.W, r.H
	if !r.Stretch {
		w, h = fitBox(240, 320, r.W, r.H)
	}
	c := gfx.New(w, h)
	c.Fill(c.Rect, rgb{R: 200, G: 40, B: 40, A: 255})
	return c.RGBA, ImageReady
}
func (*columnImages) Want([]ImageReq) {}
func (*columnImages) SetPaused(bool)  {}
