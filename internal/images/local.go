package images

import (
	"image"
	"os"
	"path/filepath"

	"image/png"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
)

// Local serves pictures synchronously from a directory laid out like the
// site's docs/images (title/, snap/, ingame/, systems/). It is what the PC
// harness and the golden tests use; nothing here is asynchronous.
type Local struct {
	dir   string
	cache map[scaledKey]*image.RGBA
	raw   map[Pic]*image.RGBA
	miss  map[Pic]bool
}

// NewLocal makes a provider over dir.
func NewLocal(dir string) *Local {
	return &Local{dir: dir, cache: map[scaledKey]*image.RGBA{}, raw: map[Pic]*image.RGBA{}, miss: map[Pic]bool{}}
}

func (l *Local) path(p Pic) string {
	if p.Slot == "system" {
		return filepath.Join(l.dir, "systems", p.Key+".png")
	}
	return filepath.Join(l.dir, p.Slot, p.Key+".png")
}

// SetPaused is a no-op: Local decodes on demand.
func (l *Local) SetPaused(bool) {}

// Get decodes and scales on the spot.
func (l *Local) Get(req app.ImageReq) (*image.RGBA, app.ImageState) {
	p := Pic{req.Key, req.Slot}
	k := scaledKey{p, req.W, req.H}
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
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			l.miss[p] = true
			return nil, app.ImageMissing
		}
		src = ToRGBA(img)
		l.raw[p] = src
	}
	fw, fh := FitSize(src.Rect.Dx(), src.Rect.Dy(), req.W, req.H)
	out := Resample(src, fw, fh)
	l.cache[k] = out
	return out, app.ImageReady
}

// Want is a no-op: everything is synchronous.
func (l *Local) Want([]app.ImageReq) {}
