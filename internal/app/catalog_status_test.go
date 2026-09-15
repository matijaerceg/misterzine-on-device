package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"testing"
	"time"
)

func TestCatalogStatusUnknownAndNarrowFit(t *testing.T) {
	if catalogStamp(time.Time{}, "Unknown") != "Unknown" || catalogStamp(time.Time{}, "Not yet") != "Not yet" {
		t.Fatal("unknown times must not show a year-one timestamp")
	}
	now := time.Date(2026, 9, 15, 18, 37, 0, 0, time.Local)
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(nil, "", now), nil)
		a.SetRotation(rot)
		a.SetInset(40, 40)
		a.SetCatalogChecked(now)
		for _, line := range []string{"Catalog: " + catalogStamp(now, "Unknown"), "Last checked: " + catalogStamp(a.CatalogChecked(), "Not yet")} {
			if a.sm.Width(line) > a.lay.Body.Dx()-4 {
				t.Fatalf("footer clips at rotation %v: %s", rot, line)
			}
		}
	}
}
