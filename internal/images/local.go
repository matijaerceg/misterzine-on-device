package images

import (
	"image"
	"os"
	"path/filepath"

	"image/png"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/fetch"
	"github.com/matijaerceg/misterzine-on-device/internal/media"
)

// Local serves pictures synchronously from a directory laid out like the
// site's docs/images (title/, snap/, ingame/, systems/). It is what the PC
// harness and the golden tests use; nothing here is asynchronous.
type Local struct {
	dir       string
	cache     map[scaledKey]*image.RGBA
	raw       map[Pic]*image.RGBA
	clips     map[Pic]*media.Clip
	clipOrder []Pic
	miss      map[Pic]bool
}

// NewLocal makes a provider over dir.
func NewLocal(dir string) *Local {
	return &Local{dir: dir, cache: map[scaledKey]*image.RGBA{}, raw: map[Pic]*image.RGBA{}, clips: map[Pic]*media.Clip{}, miss: map[Pic]bool{}}
}

func (l *Local) path(p Pic) string {
	if p.Slot == "system" {
		return filepath.Join(l.dir, "systems", p.Key+".png")
	}
	if p.Slot == fetch.SlotVideo {
		return filepath.Join(l.dir, fetch.SlotVideo, p.Key+".mzv")
	}
	return filepath.Join(l.dir, p.Slot, p.Key+".png")
}

// SetPaused is a no-op: Local decodes on demand.
func (l *Local) SetPaused(bool) {}

// Get decodes and scales on the spot.
func (l *Local) Get(req app.ImageReq) (*image.RGBA, app.ImageState) {
	k := keyFor(req)
	p := k.Pic
	if !validPic(p) {
		return nil, app.ImageMissing
	}
	if p.Slot == fetch.SlotVideo {
		return l.getVideo(k)
	}
	if img, ok := l.cache[k]; ok {
		return img, app.ImageReady
	}
	if l.miss[p] {
		return nil, app.ImageMissing
	}
	src := l.raw[p]
	if src == nil {
		f, err := os.Open(l.path(p))
		if err != nil {
			l.miss[p] = true
			return nil, app.ImageMissing
		}
		var img image.Image
		if p.Slot == fetch.SlotFlyer {
			img, err = media.ReadFlyer(f)
		} else {
			img, err = png.Decode(f)
		}
		f.Close()
		if err != nil {
			l.miss[p] = true
			return nil, app.ImageMissing
		}
		src = ToRGBA(img)
		l.raw[p] = src
	}
	out := scaleFor(src, k)
	l.cache[k] = out
	return out, app.ImageReady
}

func (l *Local) getVideo(k scaledKey) (*image.RGBA, app.ImageState) {
	p := k.Pic
	if l.miss[p] {
		return nil, app.ImageMissing
	}
	clip := l.clips[p]
	if clip == nil {
		f, err := os.Open(l.path(p))
		if err != nil {
			l.miss[p] = true
			return nil, app.ImageMissing
		}
		clip, err = media.Read(f)
		f.Close()
		if err != nil {
			l.miss[p] = true
			return nil, app.ImageMissing
		}
		l.rememberClip(p, clip)
	} else {
		l.touchClip(p)
	}
	k.Frame = clip.Normalize(k.Frame)
	if img, ok := l.cache[k]; ok {
		return img, app.ImageReady
	}
	img, err := clip.Decode(k.Frame)
	if err != nil {
		l.miss[p] = true
		return nil, app.ImageMissing
	}
	out := scaleFor(ToRGBA(img), k)
	for old := range l.cache {
		if old.Pic == k.Pic && old.W == k.W && old.H == k.H && old.Native == k.Native && old.Stretch == k.Stretch && old.Pixelate == k.Pixelate {
			delete(l.cache, old)
		}
	}
	l.cache[k] = out
	return out, app.ImageReady
}

func (l *Local) touchClip(p Pic) {
	for i, q := range l.clipOrder {
		if q == p {
			copy(l.clipOrder[i:], l.clipOrder[i+1:])
			l.clipOrder[len(l.clipOrder)-1] = p
			return
		}
	}
	l.clipOrder = append(l.clipOrder, p)
}

func (l *Local) rememberClip(p Pic, clip *media.Clip) {
	l.clips[p] = clip
	l.touchClip(p)
	for len(l.clipOrder) > 2 {
		old := l.clipOrder[0]
		l.clipOrder = l.clipOrder[1:]
		delete(l.clips, old)
	}
}

// Want is a no-op: everything is synchronous.
func (l *Local) Want([]app.ImageReq) {}
