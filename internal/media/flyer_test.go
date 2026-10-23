package media

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"strings"
	"testing"
)

type countingReader struct {
	r io.Reader
	n int
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	r.n += n
	return n, err
}

func pngImage(t *testing.T, width, height int) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestFlyerValidation(t *testing.T) {
	if _, err := DecodeFlyer(pngImage(t, MaxFlyerDimension, MaxFlyerDimension)); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{
		"malformed": []byte("not png"),
		"too wide":  pngImage(t, MaxFlyerDimension+1, 1),
		"too tall":  pngImage(t, 1, MaxFlyerDimension+1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeFlyer(data); err == nil {
				t.Fatal("invalid flyer accepted")
			}
		})
	}
}

func TestBoundedReadersStopAtLimit(t *testing.T) {
	video := &countingReader{r: strings.NewReader(strings.Repeat("x", MaxBytes+100))}
	if _, err := Read(video); err == nil || video.n != MaxBytes+1 {
		t.Fatalf("video err=%v bytes=%d", err, video.n)
	}
	flyer := &countingReader{r: strings.NewReader(strings.Repeat("x", MaxFlyerBytes+100))}
	if _, err := ReadFlyer(flyer); err == nil || flyer.n != MaxFlyerBytes+1 {
		t.Fatalf("flyer err=%v bytes=%d", err, flyer.n)
	}
}

func TestValidKey(t *testing.T) {
	for _, key := range []string{"pacman", "1942", "game-name_2"} {
		if !ValidKey(key) {
			t.Errorf("valid key %q rejected", key)
		}
	}
	for _, key := range []string{"", "../game", "/tmp", "Upper", ".hidden", "_prefix", "has.dot"} {
		if ValidKey(key) {
			t.Errorf("invalid key %q accepted", key)
		}
	}
}
