//go:build linux

// Phase-0 spike for misterzine-on-device. Throwaway code: it proves the
// MiSTer framebuffer, the fb_cmd1 handshake, vsync, evdev input and
// load_core plumbing on the real device, and logs numbers. Nothing in here
// is meant to survive into the app; the app's platform layer is written
// from the measurements this produces.
//
// Usage (from /media/fat/Scripts/spike.sh, args in spike.args):
//
//	spike all                      full battery, ~70 s, logs to spike.log
//	spike fbinfo                   geometry + sysfs + channel offsets
//	spike fbcmd W H                fb_cmd1 handshake, poll until it lands
//	spike restore                  fb_cmd1 back to the geometry saved by fbinfo/all
//	spike pattern SECS             corner colours, grey ramp, inset frames, up arrow
//	spike vsync N                  N vsync waits, interval stats
//	spike present N                N full-frame copies, timing
//	spike keys SECS                log every EV_KEY event from every device
//	spike png OUT                  dump the current shadow canvas as PNG
//	spike load PATH                write load_core PATH to /dev/MiSTer_cmd
package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	dir      = "/media/fat/misterzine"
	logPath  = dir + "/spike.log"
	origPath = dir + "/spike.orig"
	fbPath   = "/dev/fb0"
	cmdPath  = "/dev/MiSTer_cmd"
	ttyPath  = "/dev/tty0"

	fbiogetVscreeninfo = 0x4600
	fbiogetFscreeninfo = 0x4602
	fbioWaitforvsync   = 0x40044620
	kdgetmode          = 0x4B3B
	kdsetmode          = 0x4B3A
	kdText             = 0
	kdGraphics         = 1
)

// linux/fb.h layouts. Sizes checked at runtime: 160 and 68 bytes on arm32.
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

var logf *os.File

func logln(format string, a ...any) {
	line := time.Now().Format("15:04:05.000") + " " + fmt.Sprintf(format, a...)
	fmt.Println(line)
	if logf != nil {
		logf.WriteString(line + "\n")
	}
}

func ioctl(fd uintptr, req uintptr, arg unsafe.Pointer) error {
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(arg))
	if e != 0 {
		return e
	}
	return nil
}

// ---- framebuffer ----------------------------------------------------------

type fb struct {
	f      *os.File
	v      fbVarScreeninfo
	x      fbFixScreeninfo
	mem    []byte
	canvas *image.RGBA // shadow, same size as the fb
	// byte index of R, G, B within a 4-byte pixel, from the reported offsets
	ri, gi, bi int
}

func openFB() (*fb, error) {
	f, err := os.OpenFile(fbPath, os.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	b := &fb{f: f}
	if err := b.refresh(); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *fb) refresh() error {
	if err := ioctl(b.f.Fd(), fbiogetVscreeninfo, unsafe.Pointer(&b.v)); err != nil {
		return fmt.Errorf("FBIOGET_VSCREENINFO: %w", err)
	}
	if err := ioctl(b.f.Fd(), fbiogetFscreeninfo, unsafe.Pointer(&b.x)); err != nil {
		return fmt.Errorf("FBIOGET_FSCREENINFO: %w", err)
	}
	b.ri, b.gi, b.bi = int(b.v.Red.Offset/8), int(b.v.Green.Offset/8), int(b.v.Blue.Offset/8)
	return nil
}

func (b *fb) describe(tag string) {
	v, x := b.v, b.x
	logln("%s: %dx%d virt %dx%d bpp %d stride %d smem_len %d visual %d id %q",
		tag, v.Xres, v.Yres, v.XresVirtual, v.YresVirtual, v.BitsPerPixel, x.LineLength, x.SmemLen, x.Visual,
		strings.TrimRight(string(x.ID[:]), "\x00"))
	logln("%s: red off %d len %d, green off %d len %d, blue off %d len %d, transp off %d len %d",
		tag, v.Red.Offset, v.Red.Length, v.Green.Offset, v.Green.Length, v.Blue.Offset, v.Blue.Length, v.Transp.Offset, v.Transp.Length)
	if s, err := os.ReadFile("/sys/module/MiSTer_fb/parameters/mode"); err == nil {
		logln("%s: sysfs mode = %q", tag, strings.TrimSpace(string(s)))
	}
}

func (b *fb) mapMem() error {
	if b.mem != nil {
		return nil
	}
	m, err := syscall.Mmap(int(b.f.Fd()), 0, int(b.x.SmemLen), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("mmap: %w", err)
	}
	b.mem = m
	b.canvas = image.NewRGBA(image.Rect(0, 0, int(b.v.Xres), int(b.v.Yres)))
	return nil
}

func (b *fb) unmap() {
	if b.mem != nil {
		syscall.Munmap(b.mem)
		b.mem = nil
	}
}

func (b *fb) waitVSync() error {
	var zero uint32
	return ioctl(b.f.Fd(), fbioWaitforvsync, unsafe.Pointer(&zero))
}

// present copies the whole shadow canvas into the framebuffer using the
// reported channel offsets. Returns the copy duration.
func (b *fb) present() time.Duration {
	t0 := time.Now()
	w, h := b.canvas.Rect.Dx(), b.canvas.Rect.Dy()
	stride := int(b.x.LineLength)
	bpp := int(b.v.BitsPerPixel / 8)
	row := make([]byte, w*bpp)
	for y := 0; y < h; y++ {
		src := b.canvas.Pix[y*b.canvas.Stride : y*b.canvas.Stride+w*4]
		if bpp == 4 {
			for x := 0; x < w; x++ {
				p := row[x*4 : x*4+4]
				p[b.ri] = src[x*4]
				p[b.gi] = src[x*4+1]
				p[b.bi] = src[x*4+2]
				p[6-b.ri-b.gi-b.bi] = 0 // the leftover (transp) byte
			}
		} else if bpp == 2 { // RGB565 fallback
			for x := 0; x < w; x++ {
				r, g, bl := uint16(src[x*4]), uint16(src[x*4+1]), uint16(src[x*4+2])
				binary.LittleEndian.PutUint16(row[x*2:], (r>>3)<<11|(g>>2)<<5|(bl>>3))
			}
		}
		off := y * stride
		if off+len(row) <= len(b.mem) {
			copy(b.mem[off:off+len(row)], row)
		}
	}
	return time.Since(t0)
}

func fill(c *image.RGBA, r image.Rectangle, col color.RGBA) {
	r = r.Intersect(c.Rect)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		i := c.PixOffset(r.Min.X, y)
		for x := r.Min.X; x < r.Max.X; x++ {
			c.Pix[i], c.Pix[i+1], c.Pix[i+2], c.Pix[i+3] = col.R, col.G, col.B, 255
			i += 4
		}
	}
}

func frame(c *image.RGBA, inset int, col color.RGBA) {
	r := c.Rect.Inset(inset)
	fill(c, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+1), col)
	fill(c, image.Rect(r.Min.X, r.Max.Y-1, r.Max.X, r.Max.Y), col)
	fill(c, image.Rect(r.Min.X, r.Min.Y, r.Min.X+1, r.Max.Y), col)
	fill(c, image.Rect(r.Max.X-1, r.Min.Y, r.Max.X, r.Max.Y), col)
}

// pattern: physical top-left RED, top-right GREEN, bottom-left BLUE,
// bottom-right WHITE corner blocks; a grey ramp across the middle; 1 px
// frames at 4 (yellow), 8 (cyan) and 12 (magenta) px insets; a white
// triangle pointing at the physical TOP edge. The user reports which corner
// shows which colour: that settles channel order AND tate rotation at once.
func drawPattern(c *image.RGBA) {
	w, h := c.Rect.Dx(), c.Rect.Dy()
	fill(c, c.Rect, color.RGBA{24, 16, 40, 255})
	k := h / 5
	fill(c, image.Rect(0, 0, k, k), color.RGBA{255, 0, 0, 255})
	fill(c, image.Rect(w-k, 0, w, k), color.RGBA{0, 255, 0, 255})
	fill(c, image.Rect(0, h-k, k, h), color.RGBA{0, 0, 255, 255})
	fill(c, image.Rect(w-k, h-k, w, h), color.RGBA{255, 255, 255, 255})
	// grey ramp, 16 steps
	for i := 0; i < 16; i++ {
		v := uint8(i * 17)
		fill(c, image.Rect(w*i/16, h/2-k/3, w*(i+1)/16, h/2+k/3), color.RGBA{v, v, v, 255})
	}
	// up triangle
	top, bot := k+2, h/2-k/3-2
	for y := top; y < bot; y++ {
		half := (y - top) * (w / 6) / (bot - top)
		fill(c, image.Rect(w/2-half, y, w/2+half+1, y+1), color.RGBA{255, 255, 255, 255})
	}
	frame(c, 4, color.RGBA{255, 255, 0, 255})
	frame(c, 8, color.RGBA{0, 255, 255, 255})
	frame(c, 12, color.RGBA{255, 0, 255, 255})
}

// ---- MiSTer_cmd -----------------------------------------------------------

func sendCmd(line string) error {
	f, err := os.OpenFile(cmdPath, os.O_WRONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line + "\n")
	logln("MiSTer_cmd <- %q (err=%v)", line, err)
	return err
}

// fbcmd requests a custom framebuffer and waits for the kernel side to follow.
func fbcmd(b *fb, w, h int) error {
	b.refresh()
	if int(b.v.Xres) == w && int(b.v.Yres) == h {
		logln("fbcmd: already %dx%d", w, h)
		return nil
	}
	b.unmap()
	t0 := time.Now()
	if err := sendCmd(fmt.Sprintf("fb_cmd1 8888 1 %d %d", w, h)); err != nil {
		return err
	}
	for {
		time.Sleep(10 * time.Millisecond)
		b.refresh()
		if int(b.v.Xres) == w && int(b.v.Yres) == h {
			logln("fbcmd: landed %dx%d after %v", w, h, time.Since(t0).Round(time.Millisecond))
			break
		}
		if time.Since(t0) > 3*time.Second {
			logln("fbcmd: TIMEOUT, still %dx%d after %v", b.v.Xres, b.v.Yres, time.Since(t0).Round(time.Millisecond))
			break
		}
	}
	b.describe("after fbcmd")
	return b.mapMem()
}

func saveOrig(b *fb) {
	os.WriteFile(origPath, []byte(fmt.Sprintf("%d %d\n", b.v.Xres, b.v.Yres)), 0644)
}

func restore(b *fb) {
	s, err := os.ReadFile(origPath)
	if err != nil {
		logln("restore: no %s (%v)", origPath, err)
		return
	}
	var w, h int
	fmt.Sscanf(string(s), "%d %d", &w, &h)
	if w > 0 && h > 0 {
		fbcmd(b, w, h)
	}
}

// ---- console ---------------------------------------------------------------

type console struct {
	f    *os.File
	mode uint32
}

func openConsole() *console {
	f, err := os.OpenFile(ttyPath, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		logln("console: open %s: %v", ttyPath, err)
		return nil
	}
	c := &console{f: f}
	if err := ioctl(f.Fd(), kdgetmode, unsafe.Pointer(&c.mode)); err != nil {
		logln("console: KDGETMODE: %v (not a virtual console?)", err)
		f.Close()
		return nil
	}
	logln("console: current KD mode %d", c.mode)
	return c
}

func (c *console) graphics() {
	if c == nil {
		return
	}
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, c.f.Fd(), kdsetmode, kdGraphics)
	logln("console: KD_GRAPHICS (errno %d)", e)
	c.f.WriteString("\x1b[?25l")
}

func (c *console) text() {
	if c == nil {
		return
	}
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, c.f.Fd(), kdsetmode, uintptr(c.mode))
	logln("console: KD mode restored to %d (errno %d)", c.mode, e)
	c.f.WriteString("\x1b[?25h")
}

// ---- input ------------------------------------------------------------------

type inputEvent struct {
	Sec, Usec  int32
	Type, Code uint16
	Value      int32
}

func eviocgname(n int) uintptr { return uintptr(0x80000000) | uintptr(n)<<16 | uintptr(0x45)<<8 | 0x06 }

func devName(f *os.File) string {
	buf := make([]byte, 256)
	if err := ioctl(f.Fd(), eviocgname(len(buf)), unsafe.Pointer(&buf[0])); err != nil {
		return "?"
	}
	return strings.TrimRight(string(buf), "\x00")
}

func keys(b *fb, secs int) {
	paths, _ := filepath.Glob("/dev/input/event*")
	sort.Strings(paths)
	type dev struct {
		f    *os.File
		name string
		path string
	}
	var devs []dev
	for _, p := range paths {
		f, err := os.OpenFile(p, os.O_RDONLY|syscall.O_NOCTTY|syscall.O_CLOEXEC, 0)
		if err != nil {
			logln("keys: %s: %v", p, err)
			continue
		}
		d := dev{f: f, name: devName(f), path: p}
		devs = append(devs, d)
		logln("keys: reading %s (%s)", p, d.name)
	}
	if len(devs) == 0 {
		return
	}
	ch := make(chan string, 256)
	for _, d := range devs {
		d := d
		go func() {
			buf := make([]byte, 16*32)
			for {
				n, err := d.f.Read(buf)
				if err != nil {
					ch <- fmt.Sprintf("%s: read error %v", d.path, err)
					return
				}
				for i := 0; i+16 <= n; i += 16 {
					var ev inputEvent
					ev.Sec = int32(binary.LittleEndian.Uint32(buf[i:]))
					ev.Usec = int32(binary.LittleEndian.Uint32(buf[i+4:]))
					ev.Type = binary.LittleEndian.Uint16(buf[i+8:])
					ev.Code = binary.LittleEndian.Uint16(buf[i+10:])
					ev.Value = int32(binary.LittleEndian.Uint32(buf[i+12:]))
					if ev.Type == 1 { // EV_KEY
						ch <- fmt.Sprintf("%s %-24s KEY code=%d value=%d t=%d.%06d", d.path, d.name, ev.Code, ev.Value, ev.Sec, ev.Usec)
					}
				}
			}
		}()
	}
	deadline := time.After(time.Duration(secs) * time.Second)
	count := 0
	for {
		select {
		case s := <-ch:
			count++
			logln("keys: %s", s)
			if b != nil && b.canvas != nil && b.mem != nil {
				v := uint8(count * 37)
				fill(b.canvas, b.canvas.Rect, color.RGBA{v, 255 - v, 128, 255})
				frame(b.canvas, 8, color.RGBA{255, 255, 255, 255})
				b.present()
			}
		case <-deadline:
			logln("keys: done, %d key events in %ds", count, secs)
			for _, d := range devs {
				d.f.Close()
			}
			return
		}
	}
}

// keysPhase paints a plain dark-blue screen ("press keys now"), then flashes
// a new colour on every key event so the user sees input arriving on the tube.
func keysPhase(b *fb, secs int) {
	if b.canvas != nil && b.mem != nil {
		fill(b.canvas, b.canvas.Rect, color.RGBA{0, 0, 96, 255})
		frame(b.canvas, 8, color.RGBA{255, 255, 255, 255})
		b.present()
	}
	logln("--- keys phase: press buttons (%d s); the screen changes colour on each event", secs)
	keys(b, secs)
}

// ---- commands ----------------------------------------------------------------

func mustFB() *fb {
	b, err := openFB()
	if err != nil {
		logln("fb: %v", err)
		os.Exit(1)
	}
	return b
}

func cmdFbinfo(b *fb) {
	logln("struct sizes: var=%d fix=%d (expect 160, 68)", unsafe.Sizeof(fbVarScreeninfo{}), unsafe.Sizeof(fbFixScreeninfo{}))
	b.describe("fbinfo")
	if s, err := os.ReadFile("/sys/class/tty/tty0/active"); err == nil {
		logln("active tty: %s", strings.TrimSpace(string(s)))
	}
	saveOrig(b)
}

func cmdPattern(b *fb, con *console, secs int) {
	if err := b.mapMem(); err != nil {
		logln("pattern: %v", err)
		return
	}
	con.graphics()
	drawPattern(b.canvas)
	b.waitVSync()
	d := b.present()
	logln("pattern: %dx%d presented in %v, showing for %ds", b.v.Xres, b.v.Yres, d.Round(100*time.Microsecond), secs)
	time.Sleep(time.Duration(secs) * time.Second)
}

func cmdVsync(b *fb, n int) {
	var lo, hi, sum time.Duration
	lo = time.Hour
	timeouts := 0
	last := time.Now()
	for i := 0; i < n; i++ {
		if err := b.waitVSync(); err != nil {
			timeouts++
		}
		now := time.Now()
		d := now.Sub(last)
		last = now
		if i == 0 {
			continue
		}
		sum += d
		if d < lo {
			lo = d
		}
		if d > hi {
			hi = d
		}
	}
	logln("vsync: %d waits, min %v avg %v max %v, errors %d", n, lo.Round(10*time.Microsecond), (sum / time.Duration(n-1)).Round(10*time.Microsecond), hi.Round(10*time.Microsecond), timeouts)
}

func cmdPresent(b *fb, n int) {
	if err := b.mapMem(); err != nil {
		logln("present: %v", err)
		return
	}
	var sum, max time.Duration
	t0 := time.Now()
	for i := 0; i < n; i++ {
		v := uint8(i * 255 / n)
		fill(b.canvas, b.canvas.Rect, color.RGBA{v, uint8(255 - int(v)), 80, 255})
		frame(b.canvas, 8, color.RGBA{255, 255, 255, 255})
		b.waitVSync()
		d := b.present()
		sum += d
		if d > max {
			max = d
		}
	}
	total := time.Since(t0)
	logln("present: %d frames of %dx%d: copy avg %v max %v, loop period avg %v", n, b.v.Xres, b.v.Yres,
		(sum / time.Duration(n)).Round(10*time.Microsecond), max.Round(10*time.Microsecond), (total / time.Duration(n)).Round(10*time.Microsecond))
}

func cmdPng(b *fb, out string) {
	if b.canvas == nil {
		if err := b.mapMem(); err != nil {
			logln("png: %v", err)
			return
		}
		drawPattern(b.canvas)
	}
	f, err := os.Create(out)
	if err != nil {
		logln("png: %v", err)
		return
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	if err := png.Encode(w, b.canvas); err != nil {
		logln("png: encode: %v", err)
		return
	}
	w.Flush()
	st, _ := f.Stat()
	logln("png: wrote %s (%d bytes)", out, st.Size())
}

func main() {
	os.MkdirAll(dir, 0755)
	logf, _ = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	defer func() {
		if logf != nil {
			logf.Close()
		}
	}()
	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"all"}
	}
	logln("==== spike %s (pid %d)", strings.Join(args, " "), os.Getpid())

	con := openConsole()
	b := mustFB()
	// Whatever happens, put the console back and unmap.
	done := func() {
		b.unmap()
		con.text()
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		s := <-sig
		logln("signal %v, restoring", s)
		done()
		os.Exit(130)
	}()
	defer done()

	num := func(i int, def int) int {
		if i < len(args) {
			if n, err := strconv.Atoi(args[i]); err == nil {
				return n
			}
		}
		return def
	}

	switch args[0] {
	case "fbinfo":
		cmdFbinfo(b)
	case "fbcmd":
		fbcmd(b, num(1, 320), num(2, 240))
	case "restore":
		restore(b)
	case "pattern":
		cmdPattern(b, con, num(1, 10))
	case "vsync":
		cmdVsync(b, num(1, 300))
	case "present":
		cmdPresent(b, num(1, 200))
	case "keys":
		keys(b, num(1, 20))
	case "png":
		out := dir + "/spike.png"
		if len(args) > 1 {
			out = args[1]
		}
		cmdPng(b, out)
	case "load":
		if len(args) < 2 {
			logln("load: need a path")
			return
		}
		done()
		sendCmd("load_core " + args[1])
		os.Exit(0)
	case "look":
		cmdFbinfo(b)
		if err := fbcmd(b, 320, 240); err != nil {
			logln("fbcmd: %v", err)
		}
		cmdPattern(b, con, num(1, 45))
		keysPhase(b, 30)
		logln("--- restoring native geometry")
		restore(b)
		logln("==== look done")
	case "all":
		cmdFbinfo(b)
		logln("--- native geometry pattern")
		cmdPattern(b, con, 10)
		cmdVsync(b, 120)
		cmdPresent(b, 60)
		logln("--- requesting 320x240")
		if err := fbcmd(b, 320, 240); err != nil {
			logln("fbcmd: %v", err)
		}
		cmdPattern(b, con, 15)
		cmdVsync(b, 120)
		cmdPresent(b, 120)
		cmdPng(b, dir+"/spike-320x240.png")
		logln("--- keys: press d-pad, A/B/X/Y, shoulders, then any keyboard keys (20 s)")
		keysPhase(b, 20)
		logln("--- restoring native geometry")
		restore(b)
		cmdPattern(b, con, 3)
		logln("==== all done")
	default:
		logln("unknown command %q", args[0])
	}
}
