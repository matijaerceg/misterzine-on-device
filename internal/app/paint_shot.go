package app

import (
	"image"

	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// paintShot is the screenshot view: the picture on the whole canvas (the
// safe zone does not apply, so a 240p shot shows pixel for pixel on a 240p
// screen) with the slot count overlaid in a corner. Left/Right cycle the
// slots.
func (a *App) paintShot(c *gfx.Canvas) {
	l := &a.lay
	row, d, _ := a.current()
	if row == nil {
		a.screen = ScreenList
		a.paintList(c)
		return
	}
	slots := shotSlots(row)
	area := a.shotArea()
	c.Fill(area, gen.Eva.Bg)
	if len(slots) == 0 {
		a.placeholder(c, area, "no shot for this row")
	} else {
		if a.slot >= len(slots) {
			a.slot = 0
		}
		slot := slots[a.slot]
		key := row.Img
		if slot == "system" {
			key = row.Core
		}
		req := ImageReq{Key: key, Slot: slot, W: area.Dx(), H: area.Dy(), Native: true}
		img, st := a.cfg.Images.Get(req)
		if img == nil {
			a.want(req)
			text := "loading"
			if st == ImageMissing {
				text = "no shot"
			} else if st == ImageOffline {
				text = "no connection"
			}
			a.placeholder(c, area, text)
		} else {
			p := image.Pt(area.Min.X+(area.Dx()-img.Rect.Dx())/2, area.Min.Y+(area.Dy()-img.Rect.Dy())/2)
			if slot == "system" {
				c.BlitAlpha(p, img)
			} else {
				c.Blit(p, img)
			}
		}
	}
	_ = d
	if len(slots) > 1 {
		// the count, on a small dark tab inside the safe zone's corner
		s := itoa(a.slot+1) + "/" + itoa(len(slots))
		w := a.sm.Width(s) + 6
		tab := image.Rect(l.Root.Max.X-w, l.Root.Max.Y-a.sm.H-4, l.Root.Max.X, l.Root.Max.Y)
		c.Fill(tab, rgb{R: 0, G: 0, B: 0, A: 255})
		c.Text(tab.Min.X+3, tab.Min.Y+2, a.sm, s, gen.Eva.Fg)
	}
}

// shotArea is the picture area of the screen view: the whole canvas.
func (a *App) shotArea() image.Rectangle {
	return image.Rect(0, 0, a.lay.W, a.lay.H)
}

func (a *App) actShot(k platform.Key) bool {
	switch k {
	case platform.KeyBack, platform.KeyEnter, platform.KeyTab:
		// a leaf view: every way out goes back to the details it came from
		a.screen = ScreenDetails
		a.detail = detailState{from: ScreenList, opened: a.cfg.Now()}
	case platform.KeyLeft, platform.KeyRight:
		row, _, _ := a.current()
		if row != nil {
			n := len(shotSlots(row))
			if n > 0 {
				if k == platform.KeyRight {
					a.slot = (a.slot + 1) % n
				} else {
					a.slot = (a.slot + n - 1) % n
				}
				a.slotName = shotSlots(row)[a.slot]
			}
		}
	default:
		return false
	}
	a.all = true
	return true
}

// pickSlot chooses the slot for the current row: the slot the viewer chose
// last when the row has it, else the list's preference (snap, title, ingame).
func (a *App) pickSlot() {
	a.slot = 0
	row, _, _ := a.current()
	if row == nil {
		return
	}
	slots := shotSlots(row)
	want := a.slotName
	if want == "" {
		_, want = thumbSlot(row)
	}
	for i, s := range slots {
		if s == want {
			a.slot = i
			return
		}
	}
}
