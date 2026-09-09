package debugsrv

import (
	"image"
	"image/color"
	"testing"
)

func TestScreenshotRemainsStableAfterNextPaint(t *testing.T) {
	canvas := image.NewRGBA(image.Rect(0, 0, 4, 4))
	before := color.RGBA{R: 255, A: 255}
	canvas.SetRGBA(0, 0, before)
	captured := captureShot(Hooks{Shot: func() *image.RGBA { return canvas }, Run: func(f func()) {
		f()
		// The UI is free to repaint as soon as Run returns.
		canvas.SetRGBA(0, 0, color.RGBA{B: 255, A: 255})
	}})
	if captured.RGBAAt(0, 0) != before {
		t.Fatal("next paint changed the captured frame")
	}
	if captureShot(Hooks{Shot: func() *image.RGBA { return nil }, Run: func(f func()) { f() }}) != nil {
		t.Fatal("nil canvas became an image")
	}
}
