// Package media reads MisterZine's small, self-contained preview video format.
package media

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/jpeg"
	"io"
)

// ValidKey reports whether key is safe as one path component and agrees with
// the preparation and catalogue tools' media-key policy.
func ValidKey(key string) bool {
	if key == "" {
		return false
	}
	if c := key[0]; (c < 'a' || c > 'z') && (c < '0' || c > '9') {
		return false
	}
	for i := 0; i < len(key); i++ {
		c := key[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func readBounded(r io.Reader, limit int, kind string) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if err != nil {
		return nil, fmt.Errorf("%s: read: %w", kind, err)
	}
	if len(b) > limit {
		return nil, fmt.Errorf("%s: file exceeds %d bytes", kind, limit)
	}
	return b, nil
}

// Read reads and validates one MZV1 clip without ever buffering more than the
// format limit plus the byte needed to detect an oversized file.
func Read(r io.Reader) (*Clip, error) {
	b, err := readBounded(r, MaxBytes, "mzv")
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

const (
	// FPS is the fixed playback rate of MZV1 clips.
	FPS = 15
	// MaxDimension keeps decoding and display work suitable for the device.
	MaxDimension = 320
	// MaxFrames is twenty seconds at FPS.
	MaxFrames = 300
	// MaxBytes is the largest accepted or downloaded MZV1 file.
	MaxBytes   = 8 << 20
	headerSize = 12
)

type frame struct {
	start int
	end   int
}

// Clip is a validated MZV1 clip. It retains only the compressed file and its
// frame offsets; decoded frames are deliberately short-lived.
type Clip struct {
	data          []byte
	frames        []frame
	width, height int
}

// Parse validates a complete MZV1 file and indexes its JPEG records.
func Parse(data []byte) (*Clip, error) {
	if len(data) > MaxBytes {
		return nil, fmt.Errorf("mzv: file is %d bytes (maximum %d)", len(data), MaxBytes)
	}
	if len(data) < headerSize || string(data[:4]) != "MZV1" {
		return nil, fmt.Errorf("mzv: invalid header")
	}
	w := int(binary.LittleEndian.Uint16(data[4:6]))
	h := int(binary.LittleEndian.Uint16(data[6:8]))
	fps := int(binary.LittleEndian.Uint16(data[8:10]))
	n := int(binary.LittleEndian.Uint16(data[10:12]))
	if w < 1 || h < 1 || w > MaxDimension || h > MaxDimension {
		return nil, fmt.Errorf("mzv: invalid dimensions %dx%d", w, h)
	}
	if fps != FPS {
		return nil, fmt.Errorf("mzv: frame rate is %d (expected %d)", fps, FPS)
	}
	if n < 1 || n > MaxFrames {
		return nil, fmt.Errorf("mzv: invalid frame count %d", n)
	}

	c := &Clip{data: data, frames: make([]frame, 0, n), width: w, height: h}
	pos := headerSize
	for i := 0; i < n; i++ {
		if len(data)-pos < 4 {
			return nil, fmt.Errorf("mzv: frame %d has no length", i)
		}
		sz := uint64(binary.LittleEndian.Uint32(data[pos : pos+4]))
		pos += 4
		if sz == 0 || sz > uint64(len(data)-pos) {
			return nil, fmt.Errorf("mzv: frame %d length is invalid", i)
		}
		end := pos + int(sz)
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(data[pos:end]))
		if err != nil {
			return nil, fmt.Errorf("mzv: frame %d: %w", i, err)
		}
		if cfg.Width != w || cfg.Height != h {
			return nil, fmt.Errorf("mzv: frame %d is %dx%d (header says %dx%d)", i, cfg.Width, cfg.Height, w, h)
		}
		c.frames = append(c.frames, frame{start: pos, end: end})
		pos = end
	}
	if pos != len(data) {
		return nil, fmt.Errorf("mzv: %d trailing bytes", len(data)-pos)
	}
	return c, nil
}

// Width and Height report the dimensions shared by every frame.
func (c *Clip) Width() int  { return c.width }
func (c *Clip) Height() int { return c.height }

// Len reports the number of frames.
func (c *Clip) Len() int { return len(c.frames) }

// Normalize wraps any frame number into this clip's range.
func (c *Clip) Normalize(n int) int {
	n %= len(c.frames)
	if n < 0 {
		n += len(c.frames)
	}
	return n
}

// Decode decompresses one frame. n wraps so playback can use a monotonically
// increasing tick without knowing the clip length.
func (c *Clip) Decode(n int) (image.Image, error) {
	f := c.frames[c.Normalize(n)]
	img, err := jpeg.Decode(bytes.NewReader(c.data[f.start:f.end]))
	if err != nil {
		return nil, fmt.Errorf("mzv: decode frame %d: %w", n, err)
	}
	return img, nil
}
