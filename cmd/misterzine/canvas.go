//go:build linux

package main

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform/mister"
)

// canvasChooser is the canvas rule for a picture choice. Fit display and
// Full display both hand a direct-video mode under 300 lines to
// DirectVideoCanvas, since Main scales such a framebuffer per axis; Full
// display already keeps FitCanvas below 480 lines otherwise.
func (h *host) canvasChooser(choice string) func(nw, nh int) (int, int) {
	fit := mister.FitCanvas
	if h.directVideo {
		fit = mister.DirectVideoCanvas
	}
	switch choice {
	case "320x240":
		return func(int, int) (int, int) { return 320, 240 }
	case "full":
		return func(nw, nh int) (int, int) {
			if nh < 480 {
				return fit(nw, nh)
			}
			return mister.FullCanvas(nw, nh)
		}
	}
	return fit
}

func (h *host) canvasReady() bool { return !h.canvasDue.IsZero() && !time.Now().Before(h.canvasDue) }

func (h *host) applyCanvasChange() bool {
	h.canvasDue = time.Time{}
	choice := h.a.Canvas()
	if choice == h.appliedCanvas {
		return true
	}
	nw, nh := h.fb.NativeSize()
	w, ht := h.canvasChooser(choice)(nw, nh)
	if err := h.fb.Resize(h.cmd, w, ht); err != nil {
		h.lg.Printf("live picture: %v", err)
		h.a.RestoreCanvasChoice(h.appliedCanvas)
		h.a.Invalidate()
		if !h.fb.Ready() {
			h.saveAll(true)
			return false
		}
		h.a.Notice("Could not change picture; previous size restored", 8*time.Second)
		return true
	}
	h.a.SetCanvasSize(w, ht)
	h.appliedCanvas = choice
	h.lg.Printf("live picture: %s, canvas %dx%d", choice, w, ht)
	return true
}
