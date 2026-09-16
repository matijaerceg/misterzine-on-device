//go:build linux

package mister

import (
	"fmt"
	"syscall"
	"time"
)

// NativeSize retains the original output size across live canvas changes.
func (b *FB) NativeSize() (int, int) { return b.orig.W, b.orig.H }
func (b *FB) Ready() bool            { return b.mem != nil }

// Resize runs on the presentation thread, with no concurrent framebuffer writes.
// Every request is confirmed before mapping, including recovery after failure.
func (b *FB) Resize(cmd *Cmd, w, h int) error {
	if w == b.CanvasW && h == b.CanvasH {
		return nil
	}
	return resizeWithRollback(b.CanvasW, b.CanvasH, w, h, func(w, h int) error {
		if b.mem != nil {
			if err := syscall.Munmap(b.mem); err != nil {
				return err
			}
			b.mem = nil
		}
		if err := b.request(cmd, w, h, 2*time.Second); err != nil {
			return err
		}
		if err := b.refresh(); err != nil {
			return err
		}
		if b.geom.W != w || b.geom.H != h {
			return fmt.Errorf("framebuffer changed before mapping: %s", b.geom)
		}
		if err := b.mapMem(); err != nil {
			return err
		}
		b.CanvasW, b.CanvasH = w, h
		b.sx, b.sy, b.ox, b.oy = 1, 1, 0, 0
		b.row = make([]byte, w*b.geom.BPP/8)
		b.Clear()
		return nil
	})
}

func resizeWithRollback(oldW, oldH, w, h int, configure func(int, int) error) error {
	if err := configure(w, h); err != nil {
		if restore := configure(oldW, oldH); restore != nil {
			return fmt.Errorf("resize: %v; restore: %w", err, restore)
		}
		return fmt.Errorf("resize: %w (previous picture restored)", err)
	}
	return nil
}
