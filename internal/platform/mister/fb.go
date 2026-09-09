//go:build linux

// Package mister implements the platform contract on a MiSTer: the Linux
// framebuffer the menu core scales to the screen, evdev input, the console
// modes a script must leave clean, and the /dev/MiSTer_cmd FIFO.
package mister

import (
	"encoding/binary"
	"fmt"
	"image"
	"log"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	fbPath = "/dev/fb0"

	fbiogetVscreeninfo = 0x4600
	fbiogetFscreeninfo = 0x4602
	fbioWaitforvsync   = 0x40044620
)

// linux/fb.h layouts; 160 and 68 bytes on arm32 (verified on the device).
type fbBitfield struct{ Offset, Length, MsbRight uint32 }

type fbVarScreeninfo struct {
	Xres, Yres, XresVirtual, YresVirtual, Xoffset, Yoffset uint32
	BitsPerPixel, Grayscale                                uint32
	Red, Green, Blue, Transp                               fbBitfield
	Nonstd, Activate, Height, Width, AccelFlags            uint32
	Pixclock, LeftMargin, RightMargin, UpperMargin         uint32
	LowerMargin, HsyncLen, VsyncLen, Sync, Vmode, Rotate   uint32
	Colorspace                                             uint32
	Reserved                                               [4]uint32
}

type fbFixScreeninfo struct {
	ID                        [16]byte
	SmemStart                 uint32
	SmemLen, Type, TypeAux    uint32
	Visual                    uint32
	Xpanstep, Ypanstep, Ywrap uint16
	LineLength                uint32
	MmioStart                 uint32
	MmioLen, Accel            uint32
	Capabilities              uint16
	Reserved                  [2]uint16
}

func ioctl(fd uintptr, req uintptr, arg unsafe.Pointer) error {
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(arg))
	if e != 0 {
		return e
	}
	return nil
}

// Geom is the framebuffer geometry as the kernel reports it.
type Geom struct {
	W, H, Stride, BPP int
	ROff, GOff, BOff  int // byte index of each channel within a pixel
}

func (g Geom) String() string {
	return fmt.Sprintf("%dx%d %dbpp stride %d rgb@%d,%d,%d", g.W, g.H, g.BPP, g.Stride, g.ROff, g.GOff, g.BOff)
}

// FB is the framebuffer display. The app draws a canvas of CanvasW x
// CanvasH; Present integer-scales it into whatever geometry the framebuffer
// ended up with (1:1 after a successful fb_cmd1, 2x1 on a 640x240 mode, 6x4
// on 1080p) through a shadow copy so the uncached mmap is only written.
type FB struct {
	f                *os.File
	mem              []byte
	geom             Geom
	orig             Geom
	CanvasW, CanvasH int
	sx, sy, ox, oy   int
	row              []byte // one scaled fb row
	log              *log.Logger
	requested        bool
	presents         int
	LastWait         time.Duration // vsync wait inside the last Present
	MeasureTiming    bool          // opt-in performance diagnostics
}

// OpenFB opens /dev/fb0, asks Main for a canvasW x canvasH framebuffer and
// maps it. An unconfirmed sent request aborts startup after a restore attempt;
// an asynchronous mode change must never race a newly mapped framebuffer.
func OpenFB(cmd *Cmd, canvasW, canvasH int, lg *log.Logger) (*FB, error) {
	f, err := os.OpenFile(fbPath, os.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", fbPath, err)
	}
	b := &FB{f: f, CanvasW: canvasW, CanvasH: canvasH, log: lg}
	if err := b.refresh(); err != nil {
		f.Close()
		return nil, err
	}
	b.orig = b.geom
	lg.Printf("fb: native %s", b.geom)
	if b.geom.W != canvasW || b.geom.H != canvasH {
		if err := b.request(cmd, canvasW, canvasH, 2*time.Second); err != nil {
			lg.Printf("fb: request %dx%d: %v; restoring native mode before mapping", canvasW, canvasH, err)
			if b.requested {
				rollback := b.request(cmd, b.orig.W, b.orig.H, 2*time.Second)
				f.Close()
				return nil, fmt.Errorf("framebuffer request unconfirmed: %w (restore attempt: %v); refusing to map pending geometry", err, rollback)
			}
		} else {
			b.requested = true
		}
	}
	if err := b.refresh(); err != nil {
		b.CloseRestore(cmd, true)
		return nil, err
	}
	if err := b.mapMem(); err != nil {
		b.CloseRestore(cmd, true)
		return nil, err
	}
	b.sx, b.sy = b.geom.W/canvasW, b.geom.H/canvasH
	if b.sx < 1 || b.sy < 1 {
		b.CloseRestore(cmd, true)
		return nil, fmt.Errorf("framebuffer %dx%d is smaller than the canvas %dx%d", b.geom.W, b.geom.H, canvasW, canvasH)
	}
	b.ox = (b.geom.W - canvasW*b.sx) / 2
	b.oy = (b.geom.H - canvasH*b.sy) / 2
	b.row = make([]byte, canvasW*b.sx*b.geom.BPP/8)
	lg.Printf("fb: presenting %dx%d at %dx%d scale into %s, offset %d,%d", canvasW, canvasH, b.sx, b.sy, b.geom, b.ox, b.oy)
	return b, nil
}

func (b *FB) refresh() error {
	var v fbVarScreeninfo
	var x fbFixScreeninfo
	if err := ioctl(b.f.Fd(), fbiogetVscreeninfo, unsafe.Pointer(&v)); err != nil {
		return fmt.Errorf("FBIOGET_VSCREENINFO: %w", err)
	}
	if err := ioctl(b.f.Fd(), fbiogetFscreeninfo, unsafe.Pointer(&x)); err != nil {
		return fmt.Errorf("FBIOGET_FSCREENINFO: %w", err)
	}
	b.geom = Geom{
		W: int(v.Xres), H: int(v.Yres), Stride: int(x.LineLength), BPP: int(v.BitsPerPixel),
		ROff: int(v.Red.Offset / 8), GOff: int(v.Green.Offset / 8), BOff: int(v.Blue.Offset / 8),
	}
	if b.geom.BPP != 32 && b.geom.BPP != 16 {
		return fmt.Errorf("unsupported framebuffer depth %d", b.geom.BPP)
	}
	return nil
}

// request sends fb_cmd1 and polls until the kernel side reports the size.
func (b *FB) request(cmd *Cmd, w, h int, timeout time.Duration) error {
	if cmd == nil {
		return fmt.Errorf("no MiSTer_cmd")
	}
	t0 := time.Now()
	if err := cmd.Send(fmt.Sprintf("fb_cmd1 8888 1 %d %d", w, h)); err != nil {
		return err
	}
	b.requested = true // a sent request may land after its confirmation timeout
	for {
		time.Sleep(10 * time.Millisecond)
		if err := b.refresh(); err != nil {
			return err
		}
		if b.geom.W == w && b.geom.H == h {
			b.log.Printf("fb: %dx%d landed in %v: %s", w, h, time.Since(t0).Round(time.Millisecond), b.geom)
			return nil
		}
		if time.Since(t0) > timeout {
			return fmt.Errorf("timeout, still %dx%d", b.geom.W, b.geom.H)
		}
	}
}

func (b *FB) mapMem() error {
	var x fbFixScreeninfo
	if err := ioctl(b.f.Fd(), fbiogetFscreeninfo, unsafe.Pointer(&x)); err != nil {
		return err
	}
	size, err := framebufferMemorySize(x, b.geom)
	if err != nil {
		return err
	}
	m, err := syscall.Mmap(int(b.f.Fd()), 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		// Linux 20260907 omitted MiSTer_fb's mmap operation. Older
		// kernels supplied a default; Linux 6.8+ returns ENODEV. Map
		// only this driver's reported pixel memory until a fixed kernel
		// is installed. Normal /dev/fb0 mapping remains the first choice.
		offset, fallbackErr := physicalFramebufferOffset(x, err)
		if fallbackErr != nil {
			return fmt.Errorf("mmap: %w", fallbackErr)
		}
		mf, openErr := os.OpenFile("/dev/mem", os.O_RDWR|syscall.O_SYNC|syscall.O_CLOEXEC, 0)
		if openErr != nil {
			return fmt.Errorf("mmap: %v; open framebuffer memory: %w", err, openErr)
		}
		// O_SYNC gives the ARM mapping write-combining attributes, like
		// fb_io_mmap. The mmap owns its lifetime after the fd is closed.
		m, fallbackErr = syscall.Mmap(int(mf.Fd()), offset, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
		mf.Close()
		if fallbackErr != nil {
			return fmt.Errorf("mmap: %v; map framebuffer memory: %w", err, fallbackErr)
		}
		b.log.Printf("fb: /dev/fb0 mmap unavailable (%v); mapped MiSTer_fb pixel memory at %#x (%d bytes)", err, offset, size)
	}
	b.mem = m
	return nil
}

// Size is the canvas size the app should draw at.
func (b *FB) Size() (int, int) { return b.CanvasW, b.CanvasH }

// Geometry reports the live framebuffer geometry.
func (b *FB) Geometry() Geom { return b.geom }

// WaitVSync blocks until the next vertical blank (about 16 ms at most).
func (b *FB) WaitVSync() error {
	var zero uint32
	return ioctl(b.f.Fd(), fbioWaitforvsync, unsafe.Pointer(&zero))
}

// Present copies the dirty rectangles of the canvas into the framebuffer.
// Large updates wait for vsync first so a full repaint never tears; small
// ones (a cursor move) go straight in.
func (b *FB) Present(c *image.RGBA, dirty []image.Rectangle) error {
	return b.PresentWait(c, dirty, true)
}

// PresentWait is Present with the vsync wait optional (the frame loop has
// already waited).
func (b *FB) PresentWait(c *image.RGBA, dirty []image.Rectangle, wait bool) error {
	if b.mem == nil {
		return fmt.Errorf("framebuffer closed")
	}
	full := image.Rect(0, 0, b.CanvasW, b.CanvasH)
	if dirty == nil {
		dirty = []image.Rectangle{full}
	}
	area := 0
	for _, r := range dirty {
		r = r.Intersect(full)
		area += r.Dx() * r.Dy()
	}
	b.LastWait = 0
	if wait && area*4 > b.CanvasW*b.CanvasH {
		if b.MeasureTiming {
			t := time.Now()
			b.WaitVSync()
			b.LastWait = time.Since(t)
		} else {
			b.WaitVSync()
		}
	}
	for _, r := range dirty {
		r = r.Intersect(full).Intersect(c.Rect)
		if r.Empty() {
			continue
		}
		b.presentRect(c, r)
	}
	b.presents++
	return nil
}

func (b *FB) presentRect(c *image.RGBA, r image.Rectangle) {
	bpp := b.geom.BPP / 8
	sx := b.sx
	rowW := r.Dx() * sx * bpp
	row := b.row[:rowW]
	for y := r.Min.Y; y < r.Max.Y; y++ {
		src := c.Pix[c.PixOffset(r.Min.X, y):c.PixOffset(r.Max.X, y)]
		o := 0
		if bpp == 4 {
			rs, gs, bs := uint(b.geom.ROff*8), uint(b.geom.GOff*8), uint(b.geom.BOff*8)
			for x := 0; x < len(src); x += 4 {
				v := uint32(src[x])<<rs | uint32(src[x+1])<<gs | uint32(src[x+2])<<bs
				for k := 0; k < sx; k++ {
					binary.LittleEndian.PutUint32(row[o:o+4], v)
					o += 4
				}
			}
		} else { // RGB565 little-endian
			for x := 0; x < len(src); x += 4 {
				v := uint16(src[x]>>3)<<11 | uint16(src[x+1]>>2)<<5 | uint16(src[x+2]>>3)
				for k := 0; k < sx; k++ {
					row[o] = byte(v)
					row[o+1] = byte(v >> 8)
					o += 2
				}
			}
		}
		fbx := (b.ox + r.Min.X*sx) * bpp
		for k := 0; k < b.sy; k++ {
			fy := b.oy + y*b.sy + k
			off := fy*b.geom.Stride + fbx
			if off+rowW <= len(b.mem) {
				copy(b.mem[off:off+rowW], row)
			}
		}
	}
}

// Clear paints the whole framebuffer black.
func (b *FB) Clear() {
	if b.mem == nil {
		return
	}
	for i := range b.mem {
		b.mem[i] = 0
	}
}

// Close clears, unmaps and puts the framebuffer geometry back so Main's
// "Press any key" prompt renders at the size it expects. On a core launch
// Main reprograms the framebuffer itself, so callers pass restore=false.
func (b *FB) Close() error { return b.CloseRestore(nil, true) }

// CloseRestore is Close with control over the geometry restore.
func (b *FB) CloseRestore(cmd *Cmd, restore bool) error {
	if b.mem != nil {
		b.Clear()
		syscall.Munmap(b.mem)
		b.mem = nil
	}
	if restore && b.requested && cmd != nil {
		if err := b.request(cmd, b.orig.W, b.orig.H, time.Second); err != nil {
			b.log.Printf("fb: restore %dx%d: %v", b.orig.W, b.orig.H, err)
		}
	}
	return b.f.Close()
}

// SysfsMode reads the MiSTer_fb module's mode string, for logs.
func SysfsMode() string {
	s, err := os.ReadFile("/sys/module/MiSTer_fb/parameters/mode")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(s))
}
