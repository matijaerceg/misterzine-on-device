package images

import (
	"image"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
)

func TestPixelateFlattensBlocks(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 80))
	for i := range img.Pix {
		img.Pix[i] = uint8(i * 7)
	}
	Pixelate(img)
	b := 80 / pixelBlocks
	for by := 0; by < 80; by += b {
		for bx := 0; bx < 100; bx += b {
			want := img.RGBAAt(bx, by)
			for y := by; y < min(by+b, 80); y++ {
				for x := bx; x < min(bx+b, 100); x++ {
					if img.RGBAAt(x, y) != want {
						t.Fatalf("block at %d,%d not flat at %d,%d", bx, by, x, y)
					}
				}
			}
		}
	}
}

func TestPixelatedRequestIsCachedApart(t *testing.T) {
	plain := keyFor(app.ImageReq{Key: "supmodel", Slot: "snap", W: 96, H: 72})
	pix := keyFor(app.ImageReq{Key: "supmodel", Slot: "snap", W: 96, H: 72, Pixelate: true})
	if plain == pix {
		t.Fatal("a pixelated picture shares the plain one's cache entry")
	}
}
