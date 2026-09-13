package app

import (
	"bytes"
	"image/color"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// Options -> Screensaver info: title only cuts the caption to the title,
// typed as before.
func TestSaverShotsInfoTitleOnly(t *testing.T) {
	a, _, clock, _ := shotsApp()
	a.cfg.SaverInfo = "title"
	a.startSaver(*clock)
	s := a.saver.shots
	frames(a, clock, saverShotWipe)
	row := &a.ds.Rows[s.cur.row]
	titled := len(gfx.Wrap(a.ds.Der[s.cur.row].Title, a.saverShotCols(), 2))
	if len(s.lines) != titled || s.lines[0].text != row.Title || s.lines[0].col != s.curCol || len(s.title) != titled {
		t.Fatalf("title only: %+v (title %+v, %d lines)", s.lines, s.title, titled)
	}
	frames(a, clock, 3)
	if got := saverShotTypingAt(s.lines, s.typed); got != (saverShotTyping{chars: 3, cursor: true}) {
		t.Fatalf("after three frames: %+v", got)
	}
	total := saverShotBeat * (titled - 1)
	for _, ln := range s.lines {
		total += len(ln.text)
	}
	frames(a, clock, total-3)
	if got := saverShotTypingAt(s.lines, s.typed); !got.done {
		t.Fatalf("after %d frames: %+v", total, got)
	}
}

// Options -> Screensaver info: none leaves the picture alone, and the hold
// puts the title up whole over its line for as long as it lasts.
func TestSaverShotsInfoNoneHoldShowsTheTitle(t *testing.T) {
	a, _, clock, _ := shotsApp()
	a.cfg.SaverInfo = "none"
	a.startSaver(*clock)
	s := a.saver.shots
	frames(a, clock, saverShotWipe+5)
	row := &a.ds.Rows[s.cur.row]
	if len(s.lines) != 0 || len(s.title) == 0 || s.title[0].text != row.Title || s.title[0].col != s.curCol {
		t.Fatalf("none: lines %+v title %+v", s.lines, s.title)
	}
	if !bytes.Equal(a.logical.Pix, s.lit) {
		t.Fatal("something is painted over the picture with no caption")
	}
	count := func(want color.RGBA) int {
		n := 0
		for y := 0; y < 240; y++ {
			for x := 0; x < 320; x++ {
				if a.logical.RGBAAt(x, y) == want {
					n++
				}
			}
		}
		return n
	}
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	if !a.saverShotsHolding() || !s.holdOK {
		t.Fatal("Start did not hold")
	}
	frames(a, clock, saverShotRamp+2)
	if s.bright != 256 {
		t.Fatalf("bright %d under the hold", s.bright)
	}
	// the title at full brightness, whole, on its strip, with the hold's
	// line under it
	if n := count(color.RGBA(s.curCol)); n < a.sm.W*len(row.Title)/2 {
		t.Fatalf("the title is not up under the hold: %d pixels in its hue", n)
	}
	if n := count(color.RGBA{A: 255}); n < a.sm.W*a.sm.H*len(s.title)+a.sm.W {
		t.Fatalf("the title's strip and the hold's are not there: %d black pixels", n)
	}
	a.Handle(platform.Event{Key: platform.KeyStart, At: *clock})
	frames(a, clock, saverShotRamp+2)
	if !bytes.Equal(a.logical.Pix, s.lit) {
		t.Fatal("the title stayed after the hold")
	}
}

// Options: the info row sits under the brightness row with the
// screenshots only, Left/Right pick and save its value, A previews.
func TestSaverShotsInfoOptionRow(t *testing.T) {
	a, _, clock, _ := shotsApp()
	a.cfg.SaverStyle = ""
	saved := 0
	a.cfg.SettingsChanged = func() { saved++ }
	a.openPanel(ScreenOptions)
	find := func(kind string) int {
		for i, e := range a.panel.entries {
			if e.kind == kind {
				return i
			}
		}
		return -1
	}
	if find("saver-info") != -1 {
		t.Fatal("the info row shows with the lettering")
	}
	a.panel.cursor = find("saver-style")
	a.actPanel(platform.KeyRight)
	info := find("saver-info")
	if info != find("saver-bright")+1 || a.panel.entries[info].vals[a.panel.entries[info].idx] != "full" {
		t.Fatalf("info row %d (bright %d): %+v", info, find("saver-bright"), a.panel.entries[info])
	}
	a.panel.cursor = info
	a.actPanel(platform.KeyRight)
	if a.SaverInfo() != "title" || saved != 2 {
		t.Fatalf("info %q saved %d", a.SaverInfo(), saved)
	}
	a.actPanel(platform.KeyRight)
	if a.SaverInfo() != "none" || a.actPanel(platform.KeyRight) {
		t.Fatalf("info %q, or Right went past the end", a.SaverInfo())
	}
	a.Handle(platform.Event{Key: platform.KeyEnter, Pressed: true, At: *clock})
	a.Handle(platform.Event{Key: platform.KeyEnter, At: *clock})
	if !a.ScreensaverActive() || a.saver.shots == nil {
		t.Fatal("A on the info row did not preview the screenshots")
	}
	frames(a, clock, saverShotWipe+1)
	if len(a.saver.shots.lines) != 0 {
		t.Fatalf("the preview has a caption with info none: %+v", a.saver.shots.lines)
	}
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: *clock})
	if a.Screen() != ScreenOptions || a.ScreensaverActive() {
		t.Fatal("the preview's wake navigated away")
	}
}
