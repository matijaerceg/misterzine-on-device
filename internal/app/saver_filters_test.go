package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"testing"
)

func TestScreensaverFiltersCombine(t *testing.T) {
	a, _, _, _ := shotsApp()
	a.ds.Rows[0].Rot = "Vertical"
	a.ds.Rows[0].Res = "15kHz"
	a.ds.Rows[1].Rot = "Horizontal"
	a.ds.Rows[1].Res = "31kHz"
	a.cfg.Favorites = map[string]bool{"red": true}
	a.cfg.Status = func(i int) data.Status {
		if i == 0 {
			return data.StatusOutdated
		}
		return data.StatusNotFound
	}
	for _, tc := range []struct {
		card, rotation, fav bool
		res                 string
		rot                 gfx.Rotation
		want                int
	}{
		{false, false, false, "", gfx.RotNone, 2},
		{true, false, false, "", gfx.RotNone, 1},
		{false, true, false, "", gfx.RotNone, 1},
		{false, false, true, "", gfx.RotNone, 1},
		{false, false, false, "31kHz", gfx.RotNone, 1},
		{true, true, true, "15kHz", gfx.RotLeft, 1},
		{true, true, true, "31kHz", gfx.RotLeft, 0},
		{true, true, true, "15kHz", gfx.RotNone, 0},
		{false, false, false, "unknown", gfx.RotNone, 0},
	} {
		a.cfg.SaverCard = tc.card
		a.cfg.SaverRotation = tc.rotation
		a.cfg.SaverFavorites = tc.fav
		a.cfg.SaverResolution = tc.res
		a.setRotation(tc.rot)
		if got := len(a.saverPool()); got != tc.want {
			t.Fatalf("%+v: got %d", tc, got)
		}
	}
	// Browsing filters must not affect the independent pool.
	a.cfg.SaverCard = false
	a.cfg.SaverRotation = false
	a.cfg.SaverFavorites = false
	a.cfg.SaverResolution = ""
	a.SetFilters(data.Filters{Install: data.InstallMissing, FavOnly: true})
	if len(a.saverPool()) != 2 {
		t.Fatal("list filters leaked")
	}
}

func TestScreensaverSubpage(t *testing.T) {
	a, _, clock, _ := shotsApp()
	a.openOptions()
	for i, e := range a.panel.entries {
		if e.kind == "saver-options" {
			a.panel.cursor = i
		}
	}
	a.actPanel(platform.KeyEnter)
	if a.Screen() != ScreenSaverOptions {
		t.Fatal("did not open")
	}
	a.cfg.SaverFavorites = true
	a.cfg.SaverResolution = "15kHz"
	a.cfg.SaverStyle = "word"
	a.buildPanel()
	if len(a.panel.entries) != 4 {
		t.Fatal("screenshot settings visible for lettering")
	}
	a.cfg.SaverStyle = "shots"
	a.buildPanel()
	if len(a.panel.entries) != 10 || !a.cfg.SaverFavorites || a.cfg.SaverResolution != "15kHz" {
		t.Fatal("settings lost")
	}
	a.panel.cursor = len(a.panel.entries) - 1
	a.actPanel(platform.KeyEnter)
	if !a.ScreensaverActive() {
		t.Fatal("preview did not start")
	}
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: *clock})
	a.Handle(platform.Event{Key: platform.KeyBack, At: *clock})
	if a.Screen() != ScreenSaverOptions {
		t.Fatal("wake navigated")
	}
	a.actPanel(platform.KeyBack)
	if a.Screen() != ScreenOptions || a.panel.entries[a.panel.cursor].kind != "saver-options" {
		t.Fatal("back lost parent row")
	}
}
