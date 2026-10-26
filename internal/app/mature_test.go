package app

import (
	"image"
	"testing"
)

// matureApp is shotsApp with the red game marked mature.
func matureApp(t *testing.T, mode string) *App {
	t.Helper()
	a, _, _, _ := shotsApp()
	for i := range a.ds.Rows {
		if a.ds.Rows[i].K == "red" {
			a.ds.Rows[i].Mat = 1
		}
	}
	a.cfg.Mature = mode
	a.Refilter()
	return a
}

func listed(a *App, k string) bool {
	for _, i := range a.view {
		if a.ds.Rows[i].K == k {
			return true
		}
	}
	return false
}

func poolHas(a *App, k string) bool {
	for _, p := range a.saverPool() {
		if a.ds.Rows[p.row].K == k {
			return true
		}
	}
	return false
}

func TestMatureDefaultsToHide(t *testing.T) {
	a := matureApp(t, "")
	if a.Mature() != MatureHide || listed(a, "red") || !listed(a, "blue") {
		t.Fatalf("mode %q listed red %v blue %v", a.Mature(), listed(a, "red"), listed(a, "blue"))
	}
}

func TestMatureShowKeepsEverything(t *testing.T) {
	a := matureApp(t, MatureShow)
	if a.Mature() != MatureShow || !listed(a, "red") || !poolHas(a, "red") {
		t.Fatalf("mode %q listed %v pool %v", a.Mature(), listed(a, "red"), poolHas(a, "red"))
	}
	row := &a.ds.Rows[0]
	if a.pixelated(row) {
		t.Fatal("show pixelates")
	}
}

func TestMatureHideLeavesListCountsAndSaver(t *testing.T) {
	a := matureApp(t, MatureHide)
	if listed(a, "red") || !listed(a, "blue") {
		t.Fatal("hide should drop only the mature game from the list")
	}
	if poolHas(a, "red") || !poolHas(a, "blue") {
		t.Fatal("hide should drop only the mature game from the screensaver")
	}
	if a.filtersActive() {
		t.Fatal("the Options rule is not a Filters narrowing")
	}
	shown := matureApp(t, MatureShow).total
	if a.total != shown-1 {
		t.Fatalf("total %d, want %d", a.total, shown-1)
	}
}

func TestMaturePixelateKeepsRowButNotSaver(t *testing.T) {
	a := matureApp(t, MaturePixelate)
	if !listed(a, "red") {
		t.Fatal("pixelate should keep the game listed")
	}
	if poolHas(a, "red") {
		t.Fatal("a pixelated game is not screensaver décor")
	}
	rec := &recordImages{Images: a.cfg.Images}
	a.cfg.Images = rec
	for i := range a.ds.Rows {
		r := &a.ds.Rows[i]
		if r.K != "red" && r.K != "blue" {
			continue
		}
		rec.gets = nil
		a.paintThumb(a.logical, image.Rect(0, 0, 96, 72), r)
		if len(rec.gets) == 0 {
			t.Fatalf("%s: no picture asked for", r.K)
		}
		for _, g := range rec.gets {
			if g.Pixelate != (r.K == "red") {
				t.Fatalf("%s asked Pixelate=%v", r.K, g.Pixelate)
			}
		}
	}
}

type recordImages struct {
	Images
	gets []ImageReq
}

func (r *recordImages) Get(req ImageReq) (*image.RGBA, ImageState) {
	r.gets = append(r.gets, req)
	return r.Images.Get(req)
}
