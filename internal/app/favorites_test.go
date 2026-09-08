package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestUnavailableFavoritesCannotBeToggled(t *testing.T) {
	changes := 0
	a := New(Config{PhysW: 320, PhysH: 240, FavoritesUnavailable: true, FavChanged: func() { changes++ }},
		data.Ingest([]data.Row{{K: "game", Title: "Game"}}, "test", time.Now()), nil)
	a.screen = ScreenDetails
	a.actDetails(platform.KeySpace)
	if len(a.FavoriteSet()) != 0 || changes != 0 {
		t.Fatal("favorite toggle was accepted after a load error")
	}
	if a.notice != FavoritesUnavailableNotice {
		t.Fatal("favorite toggle did not explain that the file was unreadable")
	}
}
