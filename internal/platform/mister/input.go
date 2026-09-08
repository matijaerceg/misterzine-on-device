//go:build linux

package mister

import (
	"encoding/binary"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// Main holds an exclusive grab on every input device and lets go only while
// a script runs, re-grabbing blindly when it ends. This reader therefore
// NEVER grabs: a grab held here would leave the user's controller dead until
// a reboot. Main also turns the gamepad into keyboard events through its
// "MiSTer virtual input" device while a script runs, so reading keyboards is
// enough to support a controller. Kernel autorepeat (value 2) is dropped;
// the app repeats on its own.

const (
	evKey      = 1
	keyEsc     = 1
	keyTab     = 15
	keyEnter   = 28
	keySpace   = 57
	keyF12     = 88
	keyKPEnter = 96
	keyHome    = 102
	keyUp      = 103
	keyPageUp  = 104
	keyLeft    = 105
	keyRight   = 106
	keyEnd     = 107
	keyDown    = 108
	keyPageDn  = 109
)

var keyMap = map[uint16]platform.Key{
	keyUp: platform.KeyUp, keyDown: platform.KeyDown, keyLeft: platform.KeyLeft, keyRight: platform.KeyRight,
	keyEnter: platform.KeyEnter, keyKPEnter: platform.KeyEnter, keyEsc: platform.KeyBack, keySpace: platform.KeySpace,
	keyTab: platform.KeyTab, keyPageUp: platform.KeyPageUp, keyPageDn: platform.KeyPageDown,
	keyHome: platform.KeyHome, keyEnd: platform.KeyEnd, keyF12: platform.KeyScreenshot,
}

// EVIOCGNAME(len) and EVIOCGBIT(EV_KEY, len).
func eviocgname(n int) uintptr { return uintptr(0x80000000) | uintptr(n)<<16 | uintptr(0x45)<<8 | 0x06 }
func eviocgbitKey(n int) uintptr {
	return uintptr(0x80000000) | uintptr(n)<<16 | uintptr(0x45)<<8 | uintptr(0x20+evKey)
}

type device struct {
	path string
	name string
	f    *os.File
}

// Input reads every keyboard-class evdev device, rescanning for hotplug.
type Input struct {
	ch   chan platform.Event
	log  *log.Logger
	mu   sync.Mutex
	devs map[string]*device
	stop chan struct{}
	wg   sync.WaitGroup
}

// OpenInput starts reading. It never fails hard: with no devices it just
// delivers nothing until one appears.
func OpenInput(lg *log.Logger) *Input {
	in := &Input{ch: make(chan platform.Event, 256), log: lg, devs: map[string]*device{}, stop: make(chan struct{})}
	in.rescan()
	in.wg.Add(1)
	go func() {
		defer in.wg.Done()
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-in.stop:
				return
			case <-t.C:
				in.rescan()
			}
		}
	}()
	return in
}

// Devices lists the device names being read.
func (in *Input) Devices() []string {
	in.mu.Lock()
	defer in.mu.Unlock()
	var out []string
	for _, d := range in.devs {
		out = append(out, d.path+" "+d.name)
	}
	sort.Strings(out)
	return out
}

func (in *Input) rescan() {
	paths, _ := filepath.Glob("/dev/input/event*")
	seen := map[string]bool{}
	for _, p := range paths {
		seen[p] = true
		in.mu.Lock()
		_, have := in.devs[p]
		in.mu.Unlock()
		if have {
			continue
		}
		// O_NONBLOCK puts the fd under Go's poller, so Close unblocks a
		// pending Read; a plain blocking read would pin the goroutine until
		// the next key press and hang shutdown (seen on the device).
		f, err := os.OpenFile(p, os.O_RDONLY|syscall.O_NOCTTY|syscall.O_CLOEXEC|syscall.O_NONBLOCK, 0)
		if err != nil {
			continue
		}
		if !isKeyboard(f) {
			f.Close()
			continue
		}
		name := devName(f)
		if name == "misterzine launcher" { // our own console-opening keyboard
			f.Close()
			continue
		}
		d := &device{path: p, name: name, f: f}
		in.mu.Lock()
		in.devs[p] = d
		in.mu.Unlock()
		in.log.Printf("input: reading %s (%s)", p, d.name)
		in.wg.Add(1)
		go in.read(d)
	}
	in.mu.Lock()
	for p, d := range in.devs {
		if !seen[p] {
			d.f.Close() // unblocks the reader, which removes the entry
		}
	}
	in.mu.Unlock()
}

func devName(f *os.File) string {
	buf := make([]byte, 256)
	if err := ioctl(f.Fd(), eviocgname(len(buf)), unsafe.Pointer(&buf[0])); err != nil {
		return "?"
	}
	n := 0
	for n < len(buf) && buf[n] != 0 {
		n++
	}
	return string(buf[:n])
}

// isKeyboard keeps devices whose EV_KEY bitmap has KEY_ENTER and KEY_UP.
func isKeyboard(f *os.File) bool {
	var bits [96]byte // 768 key codes
	if err := ioctl(f.Fd(), eviocgbitKey(len(bits)), unsafe.Pointer(&bits[0])); err != nil {
		return false
	}
	has := func(code int) bool { return bits[code/8]&(1<<uint(code%8)) != 0 }
	return has(keyEnter) && has(keyUp)
}

func (in *Input) read(d *device) {
	defer in.wg.Done()
	defer func() {
		in.mu.Lock()
		if cur, ok := in.devs[d.path]; ok && cur == d {
			delete(in.devs, d.path)
		}
		in.mu.Unlock()
		d.f.Close()
	}()
	buf := make([]byte, 16*64)
	for {
		n, err := d.f.Read(buf)
		if err != nil {
			if errors.Is(err, syscall.EAGAIN) {
				time.Sleep(5 * time.Millisecond)
				continue
			}
			return
		}
		for i := 0; i+16 <= n; i += 16 {
			typ := binary.LittleEndian.Uint16(buf[i+8:])
			if typ != evKey {
				continue
			}
			code := binary.LittleEndian.Uint16(buf[i+10:])
			val := int32(binary.LittleEndian.Uint32(buf[i+12:]))
			if val != 0 && val != 1 {
				continue // autorepeat
			}
			sec := int64(int32(binary.LittleEndian.Uint32(buf[i:])))
			usec := int64(int32(binary.LittleEndian.Uint32(buf[i+4:])))
			k, ok := keyMap[code]
			if !ok {
				k = platform.KeyOther
			}
			ev := platform.Event{Key: k, Code: code, Pressed: val == 1, At: time.Unix(sec, usec*1000), Source: d.name}
			select {
			case in.ch <- ev:
			default: // a flooded queue drops the oldest-first semantics; fine for keys
			}
		}
	}
}

func (in *Input) Events() <-chan platform.Event { return in.ch }

const eviocgrab = 0x40044590 // _IOW('E', 0x90, int)

// ScreenLost reports whether Main has taken the screen back. When it hides
// the framebuffer it grabs every real input device exclusively (never its
// own virtual keyboard), so a momentary grab attempt on any real device
// fails with EBUSY. Our own grab is released at once; it exists only as a
// probe. Devices are opened fresh each time because Main re-creates them
// when it restarts itself.
func (in *Input) ScreenLost() bool {
	paths, _ := filepath.Glob("/dev/input/event*")
	for _, p := range paths {
		f, err := os.OpenFile(p, os.O_RDONLY|syscall.O_NOCTTY|syscall.O_CLOEXEC|syscall.O_NONBLOCK, 0)
		if err != nil {
			continue
		}
		name := devName(f)
		if name == "MiSTer virtual input" || name == "misterzine launcher" {
			f.Close()
			continue
		}
		_, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), eviocgrab, 1)
		if e == 0 {
			syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), eviocgrab, 0)
		}
		f.Close()
		if e == syscall.EBUSY {
			in.log.Printf("input: %s (%s) is grabbed: Main has the screen", p, name)
			return true
		}
	}
	return false
}

// Close stops the readers, waiting at most half a second for them.
func (in *Input) Close() error {
	close(in.stop)
	in.mu.Lock()
	for _, d := range in.devs {
		d.f.Close()
	}
	in.mu.Unlock()
	done := make(chan struct{})
	go func() { in.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		in.log.Printf("input: readers did not stop in time")
	}
	return nil
}
