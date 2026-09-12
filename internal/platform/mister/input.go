//go:build linux

package mister

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

// Main holds an exclusive grab on every input device and lets go only while
// a script runs, re-grabbing blindly when it ends. This reader therefore
// NEVER grabs: a grab held here would leave the user's controller dead until
// a reboot. Main also turns the gamepad into keyboard events through its
// "MiSTer virtual input" device while a script runs. Start is read directly
// using the saved global controller map. Navigation repeats in the app; printable
// keyboard characters keep their normal kernel autorepeat.

const (
	evKey        = 1
	keyEsc       = 1
	keyBackspace = 14
	keyTab       = 15
	keyEnter     = 28
	keySpace     = 57
	keyF12       = 88
	keyKPEnter   = 96
	keyHome      = 102
	keyUp        = 103
	keyPageUp    = 104
	keyLeft      = 105
	keyRight     = 106
	keyEnd       = 107
	keyDown      = 108
	keyPageDn    = 109
	btnStart     = 315 // BTN_START on a gamepad
)

var keyMap = map[uint16]platform.Key{
	keyBackspace: platform.KeyBackspace,
	keyUp:        platform.KeyUp, keyDown: platform.KeyDown, keyLeft: platform.KeyLeft, keyRight: platform.KeyRight,
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
	path            string
	name            string
	f               *os.File
	held            map[uint16]platform.Key
	pad             bool // a gamepad node: read by MiSTer define-slot, see padMapping
	mapping         padMapping
	vendor, product uint16
	abs             map[uint16]absInfo // axis ranges, read on first use
	axisEdge        map[uint16]uint8   // 0 centred, 1 at the minimum, 2 at the maximum
	grabbed         bool               // held exclusively (EVIOCGRAB): Main sees nothing from it
	menuCode        uint16             // the combo button whose press became Menu, until it is released
}

type absInfo struct {
	min, max int32
	ok       bool
}

type axisEvent struct {
	code    uint16
	pressed bool
}

const evAbs = 3

// Input reads every keyboard-class evdev device, rescanning for hotplug.
type Input struct {
	ch   chan platform.Event
	log  *log.Logger
	mu   sync.Mutex
	devs map[string]*device
	stop chan struct{}
	wg   sync.WaitGroup
	// probe is a keyboard of our own that never types: Main grabs it with
	// the rest when it takes the screen, so ScreenLost has something to
	// test even when every real pad is held by us.
	probe *VKeyboard
	// rawFace is set while a pad with a MiSTer map is connected: it is read
	// from the pad itself by define-slot, so everything Main types for pads
	// through its virtual keyboard is dropped. A pad without a map still
	// works through Main's translation, but only while no mapped pad is
	// present, since the translated keys carry no pad name.
	rawFace atomic.Bool
}

// OpenInput starts reading. It never fails hard: with no devices it just
// delivers nothing until one appears.
func OpenInput(lg *log.Logger) *Input {
	in := &Input{ch: make(chan platform.Event, 256), log: lg, devs: map[string]*device{}, stop: make(chan struct{})}
	if k, err := NewVKeyboard("misterzine probe"); err == nil {
		in.probe = k
	} else {
		lg.Printf("input: probe keyboard: %v", err)
	}
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
		name := devName(f)
		if name == "misterzine launcher" || name == "misterzine probe" { // our own keyboards: the console opener and the ScreenLost probe
			f.Close()
			continue
		}
		m := padMapping{}
		if name != "MiSTer virtual input" {
			m = devicePad(f)
		}
		pad := false
		if !isKeyboard(f) {
			// gamepad nodes are read for Start (Main never forwards it)
			// and, with a MiSTer map, for the face buttons by define-slot
			if m.Code == 0 && len(m.Keys) == 0 {
				f.Close()
				continue
			}
			pad = true
		}
		d := &device{path: p, name: name, f: f, pad: pad, mapping: m, held: map[uint16]platform.Key{}, abs: map[uint16]absInfo{}, axisEdge: map[uint16]uint8{}}
		d.vendor, d.product = devID(f)
		if pad && m.direct {
			// held exclusively: Main sees nothing from it, so its MiSTer menu
			// button is the app's and cannot hand the screen to Main
			if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), eviocgrab, 1); e == 0 {
				d.grabbed = true
			} else {
				in.log.Printf("input: %s (%s): could not hold it exclusively: %v", p, name, e)
			}
		}
		in.mu.Lock()
		in.devs[p] = d
		in.mu.Unlock()
		if pad {
			held := ""
			if d.grabbed {
				held = "; held exclusively, Menu button " + m.info("", "", 0, 0).Menu
			}
			in.log.Printf("input: reading %s (%s) Start button %d via %s; slots %s%s", p, d.name, m.Code, m.Source, m.slotText(), held)
		} else {
			in.log.Printf("input: reading %s (%s)", p, d.name)
		}
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
	in.updateRawFace()
}

// updateRawFace decides whether Main's translated face keys are in use.
func (in *Input) updateRawFace() {
	in.mu.Lock()
	raw := false
	for _, d := range in.devs {
		if d.pad && d.mapping.direct {
			raw = true
		}
	}
	in.mu.Unlock()
	if in.rawFace.Swap(raw) != raw {
		if raw {
			in.log.Printf("input: mapped pad(s) read by MiSTer define-slot; MiSTer's translated pad keys are ignored")
		} else {
			in.log.Printf("input: no mapped pad; pad buttons come from MiSTer's translation")
		}
	}
}

// Pads describes the gamepad nodes being read, for the pad tester.
func (in *Input) Pads() []support.Pad {
	in.mu.Lock()
	defer in.mu.Unlock()
	var out []support.Pad
	for _, d := range in.devs {
		if d.pad {
			out = append(out, d.mapping.info(filepath.Base(d.path), d.name, d.vendor, d.product))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Node < out[j].Node })
	return out
}

// slotText is the mapping in one line for the log.
func (m padMapping) slotText() string {
	if !m.Mapped {
		return "via MiSTer (no map file)"
	}
	if !m.direct {
		return "via MiSTer (A or B is not readable here)"
	}
	s := ""
	for _, name := range support.SlotOrder {
		if code, ok := m.Slots[name]; ok {
			s += fmt.Sprintf("%s=%s ", name, support.CodeText(code))
		}
	}
	if m.menu[0] || m.menu[1] {
		s += fmt.Sprintf("menu stick axes %d/%d", m.menuX&0xFFFF, m.menuY&0xFFFF)
	}
	if s == "" {
		return "none in the map"
	}
	return strings.TrimSpace(s)
}

// axisEvents turns an axis value into edge presses and releases the way
// Main does: a hat is pressed at either end; an analogue axis past a
// quarter of its range from centre is pressed, and only towards its
// maximum unless it is a stick axis, so a trigger at rest is not held.
func (d *device) axisEvents(axis uint16, value int32) []axisEvent {
	info, ok := d.abs[axis]
	if !ok {
		info = d.absRange(axis)
		d.abs[axis] = info
	}
	if !info.ok {
		return nil
	}
	var edge uint8
	if (info.max == 1 && info.min == -1) || (info.max == 2 && info.min == 0) {
		if value == info.min {
			edge = 1
		}
		if value == info.max {
			edge = 2
		}
	} else {
		span := info.max - info.min + 1
		centre := info.min + span/2
		threshold := span / 4
		if value < centre-threshold && d.mapping.bothEdges(axis) {
			edge = 1
		}
		if value > centre+threshold {
			edge = 2
		}
	}
	last := d.axisEdge[axis]
	if last == edge {
		return nil
	}
	d.axisEdge[axis] = edge
	var out []axisEvent
	if last != 0 {
		out = append(out, axisEvent{AxisCode(axis, last == 2), false})
	}
	if edge != 0 {
		out = append(out, axisEvent{AxisCode(axis, edge == 2), true})
	}
	return out
}

// absRange asks the kernel for an axis range (EVIOCGABS).
func (d *device) absRange(axis uint16) absInfo {
	var raw [6]int32 // value, minimum, maximum, fuzz, flat, resolution
	if d.f == nil || ioctl(d.f.Fd(), 0x80184540+uintptr(axis), unsafe.Pointer(&raw[0])) != nil {
		return absInfo{}
	}
	if raw[2] <= raw[1] {
		return absInfo{}
	}
	return absInfo{min: raw[1], max: raw[2], ok: true}
}

func devID(f *os.File) (vendor, product uint16) {
	var id [4]uint16
	if ioctl(f.Fd(), 0x80084502, unsafe.Pointer(&id[0])) != nil { // EVIOCGID
		return 0, 0
	}
	return id[1], id[2]
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

// isPad keeps devices whose EV_KEY bitmap has BTN_START (gamepads).
func isPad(f *os.File) bool {
	var bits [96]byte
	if err := ioctl(f.Fd(), eviocgbitKey(len(bits)), unsafe.Pointer(&bits[0])); err != nil {
		return false
	}
	return bits[btnStart/8]&(1<<uint(btnStart%8)) != 0
}

func (in *Input) read(d *device) {
	defer in.wg.Done()
	defer func() {
		in.releaseHeld(d)
		in.mu.Lock()
		if cur, ok := in.devs[d.path]; ok && cur == d {
			delete(in.devs, d.path)
		}
		in.mu.Unlock()
		d.f.Close()
	}()
	buf := make([]byte, 16*64)
	dropped := false
	for {
		n, err := d.f.Read(buf)
		if err != nil || n == 0 {
			if errors.Is(err, syscall.EAGAIN) {
				time.Sleep(5 * time.Millisecond)
				continue
			}
			return
		}
		for i := 0; i+16 <= n; i += 16 {
			typ := binary.LittleEndian.Uint16(buf[i+8:])
			code := binary.LittleEndian.Uint16(buf[i+10:])
			if typ == 0 && code == 3 {
				dropped = true
				continue
			} // SYN_DROPPED
			if dropped {
				if typ == 0 && code == 0 { // SYN_REPORT ends the invalid batch
					in.releaseHeld(d)
					dropped = false
				}
				continue
			}
			val := int32(binary.LittleEndian.Uint32(buf[i+12:]))
			sec := int64(int32(binary.LittleEndian.Uint32(buf[i:])))
			usec := int64(int32(binary.LittleEndian.Uint32(buf[i+4:])))
			at := time.Unix(sec, usec*1000)
			if typ == evAbs {
				if !d.pad || !d.mapping.direct {
					continue
				}
				for _, e := range d.axisEvents(code, val) {
					if !in.emit(d, e.code, e.pressed, 0, at) {
						return
					}
				}
				continue
			}
			if typ != evKey {
				continue
			}
			text := d.keyboardText(code)
			if val != 0 && val != 1 && !(val == 2 && text != 0) {
				continue // navigation repeats in the app; typing uses keyboard repeat
			}
			if !in.emit(d, code, val != 0, text, at) {
				return
			}
		}
	}
}

// emit delivers one press or release from a device; false once the reader
// should stop.
func (in *Input) emit(d *device, code uint16, pressed bool, text rune, at time.Time) bool {
	k, accept := d.inputKey(code)
	if !accept {
		return true
	}
	if mk, ok := d.osdKey(code, pressed); ok {
		k = mk
	}
	if d.name == "MiSTer virtual input" && in.rawFace.Load() {
		return true // the mapped pads deliver their own presses
	}
	if k == platform.KeyStart {
		text = 0
	}
	ev := platform.Event{Key: k, Text: text, Code: code, Pressed: pressed, At: at, Source: d.name}
	if pressed {
		d.held[code] = k
	} else {
		delete(d.held, code)
	}
	return in.deliver(ev)
}

func (in *Input) Events() <-chan platform.Event { return in.ch }

// osdKey turns the MiSTer menu button into Menu while the pad is held. A
// dedicated button is Menu on every press; with a combo (Select+Start on
// many arcade boards) the second button pressed while the first is down
// becomes Menu instead of its own slot, and its release ends Menu, while
// the first keeps its slot meaning, as it was already delivered.
func (d *device) osdKey(code uint16, pressed bool) (platform.Key, bool) {
	m := &d.mapping
	if !d.pad || !m.direct || m.osd[0] == 0 || (code != m.osd[0] && code != m.osd[1]) {
		return platform.KeyNone, false
	}
	if m.osd[0] == m.osd[1] {
		return platform.KeyMenu, true
	}
	if pressed {
		other := m.osd[0]
		if code == other {
			other = m.osd[1]
		}
		if _, down := d.held[other]; down && d.menuCode == 0 {
			d.menuCode = code
			return platform.KeyMenu, true
		}
		return platform.KeyNone, false
	}
	if d.menuCode == code {
		d.menuCode = 0
		return platform.KeyMenu, true
	}
	return platform.KeyNone, false
}

func (d *device) inputKey(code uint16) (platform.Key, bool) {
	if d.mapping.Code != 0 && code == d.mapping.Code {
		return platform.KeyStart, true
	}
	if d.pad {
		if k, ok := d.mapping.Keys[code]; ok {
			return k, true
		}
		// other pad buttons have no action but are delivered so the pad
		// tester can name them and the screensaver wakes
		return platform.KeyOther, true
	}
	if k, ok := keyMap[code]; ok {
		return k, true
	}
	return platform.KeyOther, true
}

const eviocgrab = 0x40044590 // _IOW('E', 0x90, int)

// ScreenLost reports whether Main has taken the screen back. When it hides
// the framebuffer it grabs every input device it has open exclusively
// (never its own virtual keyboard, but our probe keyboard too), so a
// momentary grab attempt on any of them fails with EBUSY. The pads we hold
// ourselves are skipped: Main's grab fails on those. Our own grab is
// released at once; it exists only as a probe. Devices are opened fresh
// each time because Main re-creates them when it restarts itself.
func (in *Input) ScreenLost() bool {
	paths, _ := filepath.Glob("/dev/input/event*")
	in.mu.Lock()
	ours := map[string]bool{}
	for p, d := range in.devs {
		if d.grabbed {
			ours[p] = true
		}
	}
	in.mu.Unlock()
	for _, p := range paths {
		if ours[p] {
			continue
		}
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
	if in.probe != nil {
		in.probe.Close()
	}
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

func (in *Input) deliver(ev platform.Event) bool {
	select {
	case in.ch <- ev:
		return true
	case <-in.stop:
		return false
	}
}

// Disconnect or kernel queue overflow releases held keys. After overflow the
// user must press again; never invent a press from an asynchronous state query.
func (in *Input) releaseHeld(d *device) {
	for axis := range d.axisEdge {
		delete(d.axisEdge, axis)
	}
	for code, key := range d.held {
		delete(d.held, code)
		if !in.deliver(platform.Event{Key: key, Code: code, At: time.Now(), Source: d.name}) {
			return
		}
	}
}
