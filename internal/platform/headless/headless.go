// Package headless implements the platform contract without a device: the
// display keeps the last frame and can save it as PNG, the input is fed by a
// script, and commands are recorded. The PC harness and the golden tests
// run the whole app on it.
package headless

import (
	"image"
	"image/png"
	"os"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// Display keeps the last presented frame.
type Display struct {
	W, H     int
	Frame    *image.RGBA
	Presents int
}

// NewDisplay makes a W x H display.
func NewDisplay(w, h int) *Display {
	return &Display{W: w, H: h, Frame: image.NewRGBA(image.Rect(0, 0, w, h))}
}

func (d *Display) Size() (int, int) { return d.W, d.H }

// Present copies the dirty rectangles (or everything) into the frame.
func (d *Display) Present(c *image.RGBA, dirty []image.Rectangle) error {
	d.Presents++
	if dirty == nil {
		dirty = []image.Rectangle{c.Rect}
	}
	for _, r := range dirty {
		r = r.Intersect(d.Frame.Rect).Intersect(c.Rect)
		w := r.Dx() * 4
		for y := r.Min.Y; y < r.Max.Y; y++ {
			so := c.PixOffset(r.Min.X, y)
			do := d.Frame.PixOffset(r.Min.X, y)
			copy(d.Frame.Pix[do:do+w], c.Pix[so:so+w])
		}
	}
	return nil
}

func (d *Display) Close() error { return nil }

// SavePNG writes the frame.
func (d *Display) SavePNG(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, d.Frame)
}

// Input is a scripted event source.
type Input struct {
	ch chan platform.Event
}

// NewInput makes a buffered scripted input.
func NewInput() *Input { return &Input{ch: make(chan platform.Event, 1024)} }

// Push queues one event.
func (i *Input) Push(k platform.Key, pressed bool, at time.Time) {
	i.ch <- platform.Event{Key: k, Pressed: pressed, At: at, Source: "script"}
}

func (i *Input) Events() <-chan platform.Event { return i.ch }
func (i *Input) Close() error                  { close(i.ch); return nil }

// Cmd records command lines instead of sending them.
type Cmd struct{ Lines []string }

func (c *Cmd) Send(line string) error { c.Lines = append(c.Lines, line); return nil }
