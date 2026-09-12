//go:build linux

package mister

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// writeSlotMap writes a v3 map with the given define-slot codes (A B X Y L R
// Select Start) and the OK/back pair Main uses for its own menu.
func writeSlotMap(t *testing.T, dir string, slots [8]uint32, ok, back uint32) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 128)
	for i, code := range slots {
		binary.LittleEndian.PutUint32(b[(4+i)*4:], code)
	}
	binary.LittleEndian.PutUint32(b[23*4:], back<<16|ok) // SYS_BTN_MENU_FUNC
	if err := os.WriteFile(filepath.Join(dir, "input_045e_028e_v3.map"), b, 0644); err != nil {
		t.Fatal(err)
	}
}

// The user's 8BitDo in Xbox mode, defined by position (A right, B bottom)
// with MENU OK on the bottom button and MENU BACK on the right one: Main
// sends Enter for the bottom button, but the app must treat the button
// defined as A as A regardless of the OK/back choice.
func TestFaceButtonsFollowDefineSlots(t *testing.T) {
	dir := t.TempDir()
	writeSlotMap(t, filepath.Join(dir, "inputs"), [8]uint32{305, 304, 308, 307, 310, 773, 314, 315}, 304, 305)
	bits := buttonBits(304, 305, 307, 308, 310, 314, 315)
	m := loadPadMapping(dir, "045e_028e", bits)
	if !m.Mapped || m.Code != 315 || m.Note != "" {
		t.Fatalf("mapping: %+v", m)
	}
	want := map[uint16]platform.Key{305: platform.KeyEnter, 304: platform.KeyBack, 308: platform.KeyTab, 307: platform.KeySpace}
	if len(m.Face) != len(want) {
		t.Fatalf("face buttons %v", m.Face)
	}
	for code, key := range want {
		if m.Face[code] != key {
			t.Fatalf("code %d = %v, want %v", code, m.Face[code], key)
		}
	}
	// 773 (R) is an axis-style code this node lacks: left to Main
	if _, ok := m.Slots["R"]; ok || m.Slots["L"] != 310 || m.Slots["Select"] != 314 {
		t.Fatalf("slots %v", m.Slots)
	}
	d := &device{pad: true, mapping: m, name: "pad"}
	for code, key := range want {
		if k, ok := d.inputKey(code); !ok || k != key {
			t.Fatalf("code %d -> %v %v", code, k, ok)
		}
	}
	if k, ok := d.inputKey(315); !ok || k != platform.KeyStart {
		t.Fatal(k, ok)
	}
	if k, ok := d.inputKey(310); !ok || k != platform.KeyOther {
		t.Fatal("L should arrive as an actionless button:", k, ok)
	}
	p := m.info("event3", "pad", 0x045e, 0x028e)
	if p.Slot(305) != "A" || p.Slot(304) != "B" || p.Slot(315) != "Start" || p.Slot(999) != "" || !p.Mapped {
		t.Fatalf("pad info %+v", p)
	}
	// a map with no usable face button (an encoder defined with keyboard
	// codes) keeps the pad on Main's translation
	writeSlotMap(t, filepath.Join(dir, "inputs"), [8]uint32{28, 1, 15, 57, 0, 0, 0, 315}, 0, 0)
	if m := loadPadMapping(dir, "045e_028e", bits); len(m.Face) != 0 || !m.Mapped || m.Code != 315 {
		t.Fatalf("keyboard-coded map: %+v", m)
	}
}

// While a mapped pad is being read, Main's translated Enter/Esc/Tab/Space
// are dropped (the pad delivered the press itself) but arrows still pass;
// with no mapped pad everything from the virtual keyboard is used.
func TestVirtualFaceKeysDroppedWhileMappedPadPresent(t *testing.T) {
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
			b := make([]byte, 16)
			binary.LittleEndian.PutUint16(b[8:], 1)
			binary.LittleEndian.PutUint16(b[10:], pair[0])
			binary.LittleEndian.PutUint32(b[12:], uint32(pair[1]))
			if _, err := w.Write(b); err != nil {
				t.Fatal(err)
			}
		}
		w.Close()
		in.wg.Wait()
		var keys []platform.Key
		for len(in.ch) > 0 {
			keys = append(keys, (<-in.ch).Key)
		}
		if raw {
			if len(keys) != 2 || keys[0] != platform.KeyDown || keys[1] != platform.KeyDown {
				t.Fatalf("raw mode: got %v, want only the arrow press and release", keys)
			}
		} else if len(keys) != 10 { // 8 events + the disconnect releases of the held Tab and Space
			t.Fatalf("translated mode: got %d events %v", len(keys), keys)
		}
	}
}

// updateRawFace flips only when a pad with face buttons is connected and
// logs each change once.
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
	in.devs["event1"] = &device{pad: true, mapping: padMapping{Mapped: true, Face: map[uint16]platform.Key{305: platform.KeyEnter}}}
	in.updateRawFace()
	if !in.rawFace.Load() {
		t.Fatal("mapped pad should switch to raw face buttons")
	}
	delete(in.devs, "event1")
	in.updateRawFace()
	if in.rawFace.Load() {
		t.Fatal("unplugging the mapped pad should restore the translation")
	}
	_ = time.Now
}
