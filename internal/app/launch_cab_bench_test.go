package app

import (
	"image"
	"testing"
	"time"
)

// The launch animation's frame cost on a board: the spin (many small
// faces) and the end of the push (the textured monitor over the whole
// frame). Cross-compile with go test -c and run on a board.
func BenchmarkLaunchCabFrame(b *testing.B) {
	dst := image.NewRGBA(image.Rect(0, 0, 320, 240))
	tex := image.NewRGBA(image.Rect(0, 0, cabTexW, cabTexH))
	for i := range tex.Pix {
		tex.Pix[i] = byte(i * 7)
	}
	for _, c := range []struct {
		name    string
		elapsed time.Duration
	}{
		{"spin-mid", cabSpinDur / 3},
		{"spin-end", cabSpinDur - time.Millisecond},
		{"push-end", cabSpinDur + cabPushDur - time.Millisecond},
	} {
		b.Run(c.name, func(b *testing.B) {
			angle, focal, bright := cabPose(c.elapsed, 320, 240)
			for i := 0; i < b.N; i++ {
				for j := range dst.Pix {
					dst.Pix[j] = 0
				}
				renderCab(dst, tex, angle, focal, bright)
			}
		})
	}
}
