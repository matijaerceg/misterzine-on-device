package app

import (
	"image"

	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// paintShot is the screenshot view: the picture as large as the safe area
// allows above a one-line caption. Left/Right walk the list, Up/Down cycle
// the slots.
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
	cap := image.Rect(l.Root.Min.X, area.Max.Y, l.Root.Max.X, l.Root.Max.Y)
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
		req := ImageReq{Key: key, Slot: slot, W: area.Dx(), H: area.Dy()}
		img, st := a.cfg.Images.Get(req)
		if img == nil {
			a.want(req)
			text := "loading"
			if st == ImageMissing {
				text = "no shot"
			} else if st == ImageOffline {
				text = "offline"
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
	c.Fill(cap, gen.Eva.Surface)
	right := ""
	if len(slots) > 0 {
		right = slots[a.slot] + " " + itoa(a.slot+1) + "/" + itoa(len(slots))
	}
	date := row.Updated
	if right != "" {
		right += "  "
	}
	right += date
	rw := a.sm.Width(right)
	c.TextRight(cap.Max.X-2, cap.Min.Y+2, a.sm, right, gen.Eva.Muted)
	c.Text(cap.Min.X+2, cap.Min.Y+2, a.sm, gfx.Fit(d.Title, a.sm.Cols(cap.Dx()-rw-6)), gen.Eva.Fg)
}

// shotArea is the picture area of the screen view: the safe area above the
// caption band.
func (a *App) shotArea() image.Rectangle {
	l := &a.lay
	return image.Rect(l.Root.Min.X, l.Root.Min.Y, l.Root.Max.X, l.Root.Max.Y-statusH)
}

func (a *App) actShot(k platform.Key) bool {
	switch k {
	case platform.KeyBack:
		a.screen = ScreenList
	case platform.KeyLeft:
		if a.cursor > 0 {
			a.cursor--
			a.pickSlot()
			a.ensureVisible()
		}
	case platform.KeyRight:
		if a.cursor < len(a.view)-1 {
			a.cursor++
			a.pickSlot()
			a.ensureVisible()
		}
	case platform.KeyUp, platform.KeyDown:
		row, _, _ := a.current()
		if row != nil {
			n := len(shotSlots(row))
			if n > 0 {
				if k == platform.KeyDown {
					a.slot = (a.slot + 1) % n
				} else {
					a.slot = (a.slot + n - 1) % n
				}
				a.slotName = shotSlots(row)[a.slot]
			}
		}
	case platform.KeyEnter:
		a.screen = ScreenDetails
		a.detail = detailState{from: ScreenShot}
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
