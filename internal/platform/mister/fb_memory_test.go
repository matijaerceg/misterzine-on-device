//go:build linux

package mister

import (
	"errors"
	"syscall"
	"testing"
)

func TestPhysicalFramebufferFallback(t *testing.T) {
	x := fbFixScreeninfo{SmemStart: 0x22001000, SmemLen: 320 * 240 * 4}
	copy(x.ID[:], "MiSTer_fb")
	for _, tc := range []struct {
		name   string
		mutate func(*fbFixScreeninfo)
		err    error
		want   bool
	}{
		{"missing mmap", func(*fbFixScreeninfo) {}, syscall.ENODEV, true},
		{"permission denied", func(*fbFixScreeninfo) {}, syscall.EACCES, false},
		{"out of memory", func(*fbFixScreeninfo) {}, syscall.ENOMEM, false},
		{"other driver", func(x *fbFixScreeninfo) { x.ID[0] = 'X' }, syscall.ENODEV, false},
		{"hidden address", func(x *fbFixScreeninfo) { x.SmemStart = 0 }, syscall.ENODEV, false},
		{"unaligned address", func(x *fbFixScreeninfo) { x.SmemStart++ }, syscall.ENODEV, false},
		{"empty memory", func(x *fbFixScreeninfo) { x.SmemLen = 0 }, syscall.ENODEV, false},
		{"address overflow", func(x *fbFixScreeninfo) { x.SmemStart = 0xfffff000 }, syscall.ENODEV, false},
		{"reported address", func(x *fbFixScreeninfo) { x.SmemStart = 0x23001000 }, syscall.ENODEV, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fix := x
			tc.mutate(&fix)
			offset, err := physicalFramebufferOffset(fix, tc.err)
			if tc.want {
				if err != nil || offset != int64(fix.SmemStart) {
					t.Fatalf("offset=%#x err=%v", offset, err)
				}
			} else if !errors.Is(err, tc.err) {
				t.Fatalf("expected original error %v, got %v", tc.err, err)
			}
		})
	}
}

func TestFramebufferMemoryBounds(t *testing.T) {
	for _, tc := range []struct {
		name   string
		geom   Geom
		length uint32
		want   bool
	}{
		{"240p", Geom{W: 320, H: 240, Stride: 1280, BPP: 32}, 307200, true},
		{"padded rows", Geom{W: 320, H: 240, Stride: 1536, BPP: 32}, 368640, true},
		{"16 bit", Geom{W: 320, H: 240, Stride: 640, BPP: 16}, 153600, true},
		{"short memory", Geom{W: 320, H: 240, Stride: 1280, BPP: 32}, 307199, false},
		{"short stride", Geom{W: 320, H: 240, Stride: 1279, BPP: 32}, 307200, false},
		{"empty", Geom{}, 0, false},
		{"negative height", Geom{W: 320, H: -1, Stride: 1280, BPP: 32}, 307200, false},
		{"overflow on ARM", Geom{W: 320, H: 1 << 30, Stride: 1280, BPP: 32}, 307200, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			size, err := framebufferMemorySize(fbFixScreeninfo{SmemLen: tc.length}, tc.geom)
			if (err == nil) != tc.want {
				t.Fatalf("size=%d err=%v", size, err)
			}
			if tc.want && size != int(tc.length) {
				t.Fatalf("size=%d, want %d", size, tc.length)
			}
		})
	}
}
