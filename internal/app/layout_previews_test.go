package app

import (
	"image"
	"reflect"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

type previewImages struct{ requests []ImageReq }

func (p *previewImages) Get(r ImageReq) (*image.RGBA, ImageState) {
	p.requests = append(p.requests, r)
	return nil, ImageLoading
}
func (*previewImages) Want([]ImageReq) {}
func (*previewImages) SetPaused(bool)  {}

func TestLayoutPreviewsOrientationAndState(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
		for _, inset := range []int{15, 40} {
			for _, fallback := range []bool{false, true} {
				pictures := &previewImages{}
				rows := []data.Row{
					{K: "h", Title: "Horizontal", Img: "h", ImgW: 320, ImgH: 240, ImgSlots: []string{"snap"}},
					{K: "t", Title: "Tate", Img: "t", ImgW: 240, ImgH: 320, ImgSlots: []string{"snap"}},
				}
				a := New(Config{PhysW: 320, PhysH: 240, Rotation: rot, SafeInsetX: inset, SafeInsetY: inset, Images: pictures}, data.Ingest(rows, "", time.Now()), nil)
				if fallback {
					a.view = nil
					a.cursor = 0
				}
				a.screen = ScreenOptions
				a.buildPanel()
				for i, e := range a.panel.entries {
					if e.kind == "list-layout" {
						a.panel.cursor = i
					}
				}
				cursor, top, layout, view := a.cursor, a.top, a.lay, append([]int(nil), a.view...)
				a.Paint()
				if a.cursor != cursor || a.top != top || a.lay != layout || !reflect.DeepEqual(a.view, view) || a.ListLayout() != "list" || a.screen != ScreenOptions {
					t.Fatal("preview changed live navigation or settings")
				}
				if len(pictures.requests) != 0 || len(a.wants) != 0 {
					t.Fatal("diagrams requested artwork")
				}
				pictures.requests = nil
				a.panel.cursor++
				a.all = true
				a.Paint()
				if len(pictures.requests) != 0 {
					t.Fatal("previews remain after leaving the row")
				}
			}
		}
	}
}
