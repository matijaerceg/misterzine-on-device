package media

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
)

const (
	MaxFlyerDimension = 640
	MaxFlyerBytes     = 4 << 20
)

// DecodeFlyer validates a bounded PNG before allocating its pixel image.
func DecodeFlyer(data []byte) (image.Image, error) {
	if len(data) > MaxFlyerBytes {
		return nil, fmt.Errorf("flyer: file exceeds %d bytes", MaxFlyerBytes)
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("flyer: %w", err)
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > MaxFlyerDimension || cfg.Height > MaxFlyerDimension {
		return nil, fmt.Errorf("flyer: invalid dimensions %dx%d", cfg.Width, cfg.Height)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("flyer: %w", err)
	}
	if img.Bounds().Dx() != cfg.Width || img.Bounds().Dy() != cfg.Height {
		return nil, fmt.Errorf("flyer: decoded dimensions do not match header")
	}
	return img, nil
}

// ReadFlyer reads and validates a flyer without buffering past its file cap.
func ReadFlyer(r io.Reader) (image.Image, error) {
	b, err := readBounded(r, MaxFlyerBytes, "flyer")
	if err != nil {
		return nil, err
	}
	return DecodeFlyer(b)
}
