//go:build linux

package mister

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// writeSlotMap writes a v3 map with the given define-slot codes (Right Left
// Down Up A B X Y L R Select Start), the OK/back pair Main uses for its
// own menu, and optionally the menu stick's axis numbers.
func writeSlotMap(t *testing.T, dir string, slots [12]uint32, ok, back uint32, menuX, menuY uint32) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 128)
	for i, code := range slots {
		binary.LittleEndian.PutUint32(b[i*4:], code)
	}
	binary.LittleEndian.PutUint32(b[23*4:], back<<16|ok) // SYS_BTN_MENU_FUNC
	binary.LittleEndian.PutUint32(b[28*4:], menuX)       // SYS_AXIS_X
	binary.LittleEndian.PutUint32(b[29*4:], menuY)       // SYS_AXIS_Y
	if err := os.WriteFile(filepath.Join(dir, "input_045e_028e_v3.map"), b, 0644); err != nil {
		t.Fatal(err)
	}
}

func axisBits(axes ...uint16) (abs [8]byte) {
	for _, a := range axes {
		abs[a/8] |= 1 << (a % 8)
	}
	return
}

// The user's 8BitDo in Xbox mode, defined by position (A right, B bottom)
// with MENU OK on the bottom button and MENU BACK on the right one, the
// d-pad on the hat (axes 16/17) and R on the left trigger (axis 2): Main
// sends Enter for the bottom button, but the app must treat the button
// defined as A as A regardless of the OK/back choice, and read the hat and
// trigger edges itself.
func TestPadSlotsFollowTheDefine(t *testing.T) {
	dir := t.TempDir()
	writeSlotMap(t, filepath.Join(dir, "inputs"), [12]uint32{801, 800, 803, 802, 305, 304, 308, 307, 310, 773, 314, 315}, 304, 305, 0x20000, 0x20001)
	bits := buttonBits(304, 305, 307, 308, 310, 314, 315)
	abs := axisBits(0, 1, 2, 5, 16, 17)
	m := loadPadMapping(dir, "045e_028e", bits, abs)
	if !m.Mapped || m.Code != 315 || m.Note != "" {
		t.Fatalf("mapping: %+v", m)
	}
	want := map[uint16]platform.Key{
		305: platform.KeyEnter, 304: platform.KeyBack, 308: platform.KeyTab, 307: platform.KeySpace,
		310: platform.KeyPageUp, 773: platform.KeyPageDown, 314: platform.KeySelect, 315: platform.KeyStart,
		801: platform.KeyRight, 800: platform.KeyLeft, 803: platform.KeyDown, 802: platform.KeyUp,
		AxisCode(0, true): platform.KeyRight, AxisCode(0, false): platform.KeyLeft, AxisCode(1, true): platform.KeyDown, AxisCode(1, false): platform.KeyUp,
	}
	if len(m.Keys) != len(want) {
		t.Fatalf("keys %v", m.Keys)
	}
	for code, key := range want {
		if m.Keys[code] != key {
			t.Fatalf("code %d = %v, want %v", code, m.Keys[code], key)
		}
	}
	if m.Slots["R"] != 773 || m.Slots["Select"] != 314 || m.Slots["Up"] != 802 || !m.bothEdges(0) || m.bothEdges(2) {
		t.Fatalf("slots %v menu %v", m.Slots, m.menu)
	}
	p := m.info("event3", "pad", 0x045e, 0x028e)
	if p.Slot(305) != "A" || p.Slot(304) != "B" || p.Slot(773) != "R" || p.Slot(802) != "Up" || p.Slot(999) != "" || p.MenuStick != "axes 0/1" {
		t.Fatalf("pad info %+v", p)
	}
	d := &device{pad: true, mapping: m, name: "pad", abs: map[uint16]absInfo{}, axisEdge: map[uint16]uint8{}}
	if k, ok := d.inputKey(314); !ok || k != platform.KeySelect {
		t.Fatal("Select should arrive as the quick-toggle modifier:", k, ok)
	}
	// a slot whose code this node cannot report is left to Main
	writeSlotMap(t, filepath.Join(dir, "inputs"), [12]uint32{801, 800, 803, 802, 305, 304, 308, 307, 310, 773, 314, 315}, 0, 0, 0, 0)
	if m := loadPadMapping(dir, "045e_028e", bits, axisBits(16, 17)); m.Keys[773] != platform.KeyNone || len(m.Keys) != 11 {
		t.Fatalf("trigger without the axis: %v", m.Keys)
	}
	// keyboard-coded slots (an encoder) are not raw pad buttons
	writeSlotMap(t, filepath.Join(dir, "inputs"), [12]uint32{106, 105, 108, 103, 28, 1, 15, 57, 0, 0, 0, 315}, 0, 0, 0, 0)
	if m := loadPadMapping(dir, "045e_028e", bits, abs); len(m.Keys) != 1 || m.Keys[315] != platform.KeyStart {
		t.Fatalf("keyboard-coded map: %+v", m)
	}
}

func absEvent(typ, code uint16, val int32) []byte {
	b := make([]byte, 16)
	binary.LittleEndian.PutUint16(b[8:], typ)
	binary.LittleEndian.PutUint16(b[10:], code)
	binary.LittleEndian.PutUint32(b[12:], uint32(val))
	return b
}

// Axis edges: the hat presses at either end, the trigger only towards its
// maximum, the stick both ways past a quarter of its range, with releases
// when they return; a fixture supplies the ranges the kernel would.
func TestAxisEdgesBecomeSlotPresses(t *testing.T) {
	dir := t.TempDir()
	writeSlotMap(t, filepath.Join(dir, "inputs"), [12]uint32{801, 800, 803, 802, 305, 304, 308, 307, 310, 773, 314, 315}, 0, 0, 0x20000, 0x20001)
	m := loadPadMapping(dir, "045e_028e", buttonBits(304, 305, 307, 308, 310, 314, 315), axisBits(0, 1, 2, 5, 16, 17))
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	in := &Input{ch: make(chan platform.Event, 64), log: log.New(io.Discard, "", 0), stop: make(chan struct{}), devs: map[string]*device{}}
	d := &device{f: r, pad: true, mapping: m, name: "pad", held: map[uint16]platform.Key{},
		abs:      map[uint16]absInfo{16: {-1, 1, true}, 17: {-1, 1, true}, 2: {0, 255, true}, 5: {0, 255, true}, 0: {-32768, 32767, true}, 1: {-32768, 32767, true}},
		axisEdge: map[uint16]uint8{}}
	in.wg.Add(1)
	go in.read(d)
	for _, e := range [][3]int32{
		{evAbs, 16, 1}, {evAbs, 16, 0}, // hat right, release
		{evAbs, 17, -1}, {evAbs, 17, 1}, {evAbs, 17, 0}, // hat up, straight to down, release
		{evAbs, 2, 40}, {evAbs, 2, 250}, {evAbs, 2, 100}, {evAbs, 2, 0}, // trigger: below threshold, pressed, released once back inside
		{evAbs, 5, 255}, {evAbs, 5, 0}, // unmapped trigger: reported for the tester only
		{evAbs, 0, -30000}, {evAbs, 0, 0}, {evAbs, 0, 30000}, {evAbs, 0, 0}, // stick left, centre, right, centre
		{evKey, 305, 1}, {evKey, 305, 0},
	} {
		if _, err := w.Write(absEvent(uint16(e[0]), uint16(e[1]), e[2])); err != nil {
			t.Fatal(err)
		}
	}
	w.Close()
	in.wg.Wait()
	type got struct {
		key     platform.Key
		code    uint16
		pressed bool
	}
	var events []got
	for len(in.ch) > 0 {
		e := <-in.ch
		events = append(events, got{e.Key, e.Code, e.Pressed})
	}
	want := []got{
		{platform.KeyRight, 801, true}, {platform.KeyRight, 801, false},
		{platform.KeyUp, 802, true}, {platform.KeyUp, 802, false}, {platform.KeyDown, 803, true}, {platform.KeyDown, 803, false},
		{platform.KeyPageDown, 773, true}, {platform.KeyPageDown, 773, false},
		{platform.KeyOther, AxisCode(5, true), true}, {platform.KeyOther, AxisCode(5, true), false},
		{platform.KeyLeft, AxisCode(0, false), true}, {platform.KeyLeft, AxisCode(0, false), false}, {platform.KeyRight, AxisCode(0, true), true}, {platform.KeyRight, AxisCode(0, true), false},
		{platform.KeyEnter, 305, true}, {platform.KeyEnter, 305, false},
	}
	if len(events) != len(want) {
		t.Fatalf("got %d events %v, want %d", len(events), events, len(want))
	}
	for i := range want {
		if events[i] != want[i] {
			t.Fatalf("event %d = %v, want %v", i, events[i], want[i])
		}
	}
}

// While a mapped pad is being read, everything Main types for pads through
// its virtual keyboard is dropped; with no mapped pad it is all used.
func TestVirtualKeysDroppedWhileMappedPadPresent(t *testing.T) {
	for _, raw := range []bool{true, false} {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		in := &Input{ch: make(chan platform.Event, 16), log: log.New(io.Discard, "", 0), stop: make(chan struct{}), devs: map[string]*device{}}
		in.rawFace.Store(raw)
		d := &device{f: r, name: "MiSTer virtual input", held: map[uint16]platform.Key{}}
		in.wg.Add(1)
		go in.read(d)
		for _, pair := range [][2]uint16{{keyEnter, 1}, {keyEnter, 0}, {keyDown, 1}, {keyDown, 0}, {keyEsc, 1}, {keyEsc, 0}, {keyTab, 1}, {keySpace, 1}} {
			if _, err := w.Write(absEvent(evKey, pair[0], int32(pair[1]))); err != nil {
				t.Fatal(err)
			}
		}
		w.Close()
		in.wg.Wait()
		n := len(in.ch)
		if raw && n != 0 {
			t.Fatalf("raw mode: %d events passed", n)
		}
		if !raw && n != 10 { // 8 events + the disconnect releases of the held Tab and Space
			t.Fatalf("translated mode: got %d events", n)
		}
	}
}

// updateRawFace flips only when a mapped pad with usable slots is
// connected.
func TestRawFaceFollowsMappedPads(t *testing.T) {
	in := &Input{log: log.New(io.Discard, "", 0), devs: map[string]*device{}}
	in.updateRawFace()
	if in.rawFace.Load() {
		t.Fatal("no devices")
	}
	in.devs["event0"] = &device{pad: true, mapping: padMapping{Code: 315}}
	in.updateRawFace()
	if in.rawFace.Load() {
		t.Fatal("a Start-only pad must keep Main's translation")
	}
	in.devs["event1"] = &device{pad: true, mapping: padMapping{Mapped: true, Keys: map[uint16]platform.Key{305: platform.KeyEnter}}}
	in.updateRawFace()
	if !in.rawFace.Load() {
		t.Fatal("mapped pad should switch to raw reading")
	}
	delete(in.devs, "event1")
	in.updateRawFace()
	if in.rawFace.Load() {
		t.Fatal("unplugging the mapped pad should restore the translation")
	}
}
