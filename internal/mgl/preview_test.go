package mgl_test

import (
	"os"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/mgl"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// Exercise the real UI's launch callback, not a model of the picker. This
// does not claim that MiSTer hardware loaded the game or its BIOS/ROMs.
func TestImportedRowsLaunchOriginalShortcut(t *testing.T) {
	f, err := os.Open("../../testdata/mgl/arcade-pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	c, err := mgl.ReadCatalogue(f)
	if err != nil {
		t.Fatal(err)
	}
	r, err := mgl.Scan("../../testdata/mgl/card", c)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range r.Rows {
		t.Run(row.SN, func(t *testing.T) {
			now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
			launched := ""
			a := app.New(app.Config{
				PhysW: 320, PhysH: 240, SafeInsetX: 15, SafeInsetY: 15,
				Now: func() time.Time { return now }, ClockTrusted: true,
				Status: func(int) data.Status { return data.StatusUnknown },
				Launch: func(p string) { launched = p },
			}, data.Ingest([]data.Row{row}, "mgl-prototype", now), nil)
			a.Paint()
			a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: now})
			a.Handle(platform.Event{Key: platform.KeyStart, At: now.Add(time.Millisecond)})
			if launched != row.MRA {
				t.Fatalf("launched %q; want untouched MGL path %q", launched, row.MRA)
			}
		})
	}
}
