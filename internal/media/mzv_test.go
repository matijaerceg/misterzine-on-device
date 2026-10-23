package media

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func jpegFrame(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, nil); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func makeClip(t *testing.T, w, h int, frames ...[]byte) []byte {
	t.Helper()
	var out bytes.Buffer
	out.WriteString("MZV1")
	for _, n := range []int{w, h, FPS, len(frames)} {
		if err := binary.Write(&out, binary.LittleEndian, uint16(n)); err != nil {
			t.Fatal(err)
		}
	}
	for _, frame := range frames {
		if err := binary.Write(&out, binary.LittleEndian, uint32(len(frame))); err != nil {
			t.Fatal(err)
		}
		out.Write(frame)
	}
	return out.Bytes()
}

func TestParseAndSelectFrames(t *testing.T) {
	red := jpegFrame(t, 3, 2, color.RGBA{R: 255, A: 255})
	blue := jpegFrame(t, 3, 2, color.RGBA{B: 255, A: 255})
	c, err := Parse(makeClip(t, 3, 2, red, blue))
	if err != nil {
		t.Fatal(err)
	}
	if c.Width() != 3 || c.Height() != 2 || c.Len() != 2 || c.Normalize(5) != 1 || c.Normalize(-1) != 1 {
		t.Fatalf("clip metadata or normalization is wrong: %dx%d len=%d", c.Width(), c.Height(), c.Len())
	}
	img, err := c.Decode(3)
	if err != nil {
		t.Fatal(err)
	}
	r, _, b, _ := img.At(1, 1).RGBA()
	if b <= r {
		t.Fatal("wrapped frame did not select the blue JPEG")
	}
}

func TestParseRejectsMalformedClips(t *testing.T) {
	good := jpegFrame(t, 2, 2, color.White)
	base := makeClip(t, 2, 2, good)
	tests := map[string][]byte{
		"short":        []byte("MZV1"),
		"magic":        append([]byte("NOPE"), base[4:]...),
		"dimensions":   makeClip(t, MaxDimension+1, 2, good),
		"wrong fps":    append([]byte(nil), base...),
		"zero frames":  makeClip(t, 2, 2),
		"short record": append([]byte(nil), base[:len(base)-1]...),
		"trailing":     append(append([]byte(nil), base...), 0),
		"jpeg size":    makeClip(t, 3, 2, good),
		"bad jpeg":     makeClip(t, 2, 2, []byte("not jpeg")),
	}
	tests["wrong fps"][8] = 14
	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(data); err == nil {
				t.Fatal("malformed clip accepted")
			}
		})
	}
	if _, err := Parse(make([]byte, MaxBytes+1)); err == nil {
		t.Fatal("oversize clip accepted")
	}
}

func TestSyntheticHarnessFixture(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "images", "video", "media-demo.mzv"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if c.Width() != 320 || c.Height() != 240 || c.Len() != 3 {
		t.Fatalf("fixture is %dx%d with %d frames", c.Width(), c.Height(), c.Len())
	}
	seen := map[[32]byte]bool{}
	for i := 0; i < c.Len(); i++ {
		img, err := c.Decode(i)
		if err != nil {
			t.Fatal(err)
		}
		pixels := make([]byte, 0, c.Width()*c.Height()*3)
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				r, g, b, _ := img.At(x, y).RGBA()
				pixels = append(pixels, byte(r>>8), byte(g>>8), byte(b>>8))
			}
		}
		seen[sha256.Sum256(pixels)] = true
	}
	if len(seen) != c.Len() {
		t.Fatal("fixture frames are not visually distinct")
	}
}
