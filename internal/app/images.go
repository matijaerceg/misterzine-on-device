package app

import (
	"image"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// ImageReq names one scaled picture: an arcade shot (Img + slot) or a system
// photo (Core + slot "system"), scaled to fit W x H.
type ImageReq struct {
	Key  string // row.Img for shots, row.Core for system photos
	Slot string // "title", "snap", "ingame", "system"
	W, H int    // box to fit into, aspect kept
	// Native: a picture close to the box (up to 15% larger) shows pixel
	// for pixel, cropped centrally; only a much bigger one is scaled down
	Native bool
}

// ImageState says why Get returned nil.
type ImageState int

const (
	ImageReady   ImageState = iota // Get returns the bitmap
	ImageLoading                   // fetching or decoding
	ImageMissing                   // the site has no such picture
	ImageOffline                   // not cached and no network
)

// Images is the app's view of the picture pipeline. Get never blocks: a miss
// draws a placeholder and the app registers what it wants next.
type Images interface {
	Get(req ImageReq) (*image.RGBA, ImageState)
	Want(reqs []ImageReq)
	// SetPaused stops background decoding while the UI scrolls.
	SetPaused(paused bool)
}

// noImages is the provider used before the pipeline exists, and in tests.
type noImages struct{}

func (noImages) Get(ImageReq) (*image.RGBA, ImageState) { return nil, ImageMissing }
func (noImages) SetPaused(bool)                         {}
func (noImages) Want([]ImageReq)                        {}

// thumbSlot picks the slot the list thumbnail shows: snap, then title, then
// ingame; system photos for rows without shots.
func thumbSlot(r *data.Row) (key, slot string) {
	if r.Img != "" && len(r.ImgSlots) > 0 {
		for _, want := range []string{"snap", "title", "ingame"} {
			for _, s := range r.ImgSlots {
				if s == want {
					return r.Img, s
				}
			}
		}
		return r.Img, r.ImgSlots[0]
	}
	if r.Core != "" && !r.IsArcade() {
		return r.Core, "system"
	}
	return "", ""
}

// fitBox scales (w, h) to fit inside (bw, bh) keeping aspect; never upscales
// beyond 2x so tiny system photos stay crisp.
func fitBox(w, h, bw, bh int) (int, int) {
	if w <= 0 || h <= 0 {
		return 0, 0
	}
	fw, fh := bw, h*bw/w
	if fh > bh {
		fh = bh
		fw = w * bh / h
	}
	if fw > 2*w || fh > 2*h {
		fw, fh = 2*w, 2*h
	}
	if fw < 1 {
		fw = 1
	}
	if fh < 1 {
		fh = 1
	}
	return fw, fh
}
