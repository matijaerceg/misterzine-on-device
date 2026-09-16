//go:build linux

package main

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform/mister"
)

func (h *host) canvasReady() bool { return !h.canvasDue.IsZero() && !time.Now().Before(h.canvasDue) }

func (h *host) applyCanvasChange() bool {
	h.canvasDue = time.Time{}
	choice := h.a.Canvas()
	if choice == h.appliedCanvas {
		return true
	}
	nw, nh := h.fb.NativeSize()
	w, ht := mister.FitCanvas(nw, nh)
	if choice == "full" {
		w, ht = mister.FullCanvas(nw, nh)
	} else if choice == "320x240" {
		w, ht = 320, 240
	}
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
