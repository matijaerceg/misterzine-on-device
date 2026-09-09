package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

func TestNeighbourThumbnailMatchesDisplayedVariant(t *testing.T) {
	for _, size := range [][2]int{{320, 240}, {240, 320}, {240, 240}} {
		rows := []data.Row{{K: "a", Title: "A", Img: "a", ImgSlots: []string{"snap"}, ImgW: size[0], ImgH: size[1]}, {K: "b", Title: "B", Img: "b", ImgSlots: []string{"snap"}, ImgW: size[0], ImgH: size[1]}}
		a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(rows, "test", time.Now()), nil)
		a.MoveToKey("a")
		a.Paint()
		var warm ImageReq
		for _, r := range a.wants {
			if r.Key == "b" {
				warm = r
			}
		}
		a.MoveToKey("b")
		a.Paint()
		if len(a.wants) == 0 || warm != a.wants[0] {
			t.Fatalf("warm=%+v, displayed=%+v", warm, a.wants)
		}
		a.screen = ScreenShot
		a.all = true
		a.Paint()
		for _, r := range a.wants {
			if r.Key != "b" {
				t.Fatalf("full screen requested unreachable neighbour: %+v", r)
			}
		}
	}
}
