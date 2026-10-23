package app

import (
	"image"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

var listArtTypes = []string{"gameplay", "title", "video", "flyer"}

// Only the visible preview plays. Decoding runs through Images on its worker;
// while a frame is pending we keep the preceding frame, never a loading flash.
type videoState struct {
	req            ImageReq
	row            string
	screen         Screen
	box            image.Rectangle
	center         bool
	visible, dirty bool
	last           *image.RGBA
	next           time.Time
}

const videoInterval = time.Second / 15

func (a *App) mediaImage(req ImageReq, box image.Rectangle, center bool) (*image.RGBA, ImageState) {
	if req.Slot != "video" {
		return a.cfg.Images.Get(req)
	}
	v := &a.video
	base := v.req
	base.Frame = 0
	if base != req || v.row != a.CursorKey() || v.screen != a.screen {
		*v = videoState{req: req, row: a.CursorKey(), screen: a.screen}
	}
	v.box, v.center, v.visible = box, center, true
	img, st := a.cfg.Images.Get(v.req)
	a.want(v.req)
	if img != nil {
		v.last = img
		if v.next.IsZero() {
			v.next = a.cfg.TimerNow().Add(videoInterval)
		}
		return img, st
	}
	if st == ImageMissing || st == ImageOffline {
		v.next = time.Time{}
		v.last = nil
		return nil, st
	}
	if v.last != nil {
		return v.last, ImageReady
	}
	return nil, st
}

func (a *App) nextVideoTick() time.Time {
	v := &a.video
	if !v.visible || v.row != a.CursorKey() || v.screen != a.screen || a.saver.active || a.lock != nil || a.cab.active || a.RepeatActive() || a.PageTransitionRunning() || a.LayoutTransitionRunning() {
		return time.Time{}
	}
	return v.next
}

func (a *App) tickVideo(now time.Time) bool {
	next := a.nextVideoTick()
	if next.IsZero() || now.Before(next) {
		return false
	}
	a.video.req.Frame++
	a.video.next = time.Time{}
	a.video.dirty = true
	return true
}

func (a *App) paintVideo(c *gfx.Canvas) {
	v := &a.video
	v.dirty = false
	if a.screen == ScreenShot {
		a.paintShot(c) // redraw the count/notice above the new frame
	} else {
		req := v.req
		req.Frame = 0
		img, _ := a.mediaImage(req, v.box, v.center)
		if img != nil {
			c.Fill(v.box, pal.Bg)
			p := image.Pt(v.box.Min.X, v.box.Min.Y+(v.box.Dy()-img.Rect.Dy())/2)
			if v.center {
				p.X += (v.box.Dx() - img.Rect.Dx()) / 2
			}
			c.Blit(p, img)
			if a.screen == ScreenList {
				a.paintedThumbImage = img
			}
		} else {
			// A terminal decode/network failure needs the normal view's
			// screenshot fallback or placeholder, not the last video pixels.
			a.all = true
		}
	}
	a.cfg.Images.Want(a.wants)
}
