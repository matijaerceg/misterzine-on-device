//go:build linux

package mister

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// setOSD writes the MiSTer menu button (slot 21) and its combo partner
// (slot 22) into the map writeSlotMap produced.
func setOSD(t *testing.T, dir string, one, two uint32) {
	t.Helper()
	p := filepath.Join(dir, "inputs", "input_045e_028e_v3.map")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint32(b[21*4:], one)
	binary.LittleEndian.PutUint32(b[22*4:], two)
	if err := os.WriteFile(p, b, 0644); err != nil {
		t.Fatal(err)
	}
}

// The Xbox pad's logo button (316) in slot 21 is the app's Menu button;
// an arcade board's Select+Start combo becomes Menu on the second press
// while the first keeps its slot.
func TestMenuButtonFromTheDefine(t *testing.T) {
	dir := t.TempDir()
	slots := [12]uint32{801, 800, 803, 802, 305, 304, 308, 307, 310, 311, 314, 315}
	writeSlotMap(t, filepath.Join(dir, "inputs"), slots, 0, 0, 0, 0)
	setOSD(t, dir, 316, 0)
	bits := buttonBits(304, 305, 307, 308, 310, 311, 314, 315, 316)
	m := loadPadMapping(dir, "045e_028e", bits, axisBits(16, 17))
	if !m.direct || m.osd != [2]uint16{316, 316} || m.info("", "", 0, 0).Menu != "316" {
		t.Fatalf("mapping: %+v", m)
	}
	d := &device{pad: true, mapping: m, held: map[uint16]platform.Key{}}
	for _, pressed := range []bool{true, false} {
		if k, ok := d.osdKey(316, pressed); !ok || k != platform.KeyMenu {
			t.Fatalf("logo button pressed=%v: %v %v", pressed, k, ok)
		}
	}
	if _, ok := d.osdKey(305, true); ok {
		t.Fatal("A is not the menu button")
	}

	setOSD(t, dir, 314, 315)
	m = loadPadMapping(dir, "045e_028e", bits, axisBits(16, 17))
	if m.osd != [2]uint16{314, 315} || m.info("", "", 0, 0).Menu != "314+315" {
		t.Fatalf("combo: %+v", m)
	}
	d = &device{pad: true, mapping: m, held: map[uint16]platform.Key{}}
	if _, ok := d.osdKey(314, true); ok {
		t.Fatal("Select alone keeps its slot")
	}
	d.held[314] = platform.KeySelect
	if k, ok := d.osdKey(315, true); !ok || k != platform.KeyMenu {
		t.Fatalf("Start with Select held should be Menu: %v %v", k, ok)
	}
	d.held[315] = platform.KeyMenu
	if _, ok := d.osdKey(314, false); ok {
		t.Fatal("releasing the first button keeps its slot")
	}
	if k, ok := d.osdKey(315, false); !ok || k != platform.KeyMenu {
		t.Fatalf("releasing the second button ends Menu: %v %v", k, ok)
	}
	delete(d.held, 314)
	if _, ok := d.osdKey(315, true); ok {
		t.Fatal("Start alone keeps its slot")
	}
}

// A map without a readable A or B is not taken over: nothing but Start is
// read here and Main's translation stays in use for the pad.
func TestPadWithoutABStaysOnMain(t *testing.T) {
	dir := t.TempDir()
	writeSlotMap(t, filepath.Join(dir, "inputs"), [12]uint32{801, 800, 803, 802, 305, 0, 308, 307, 310, 311, 314, 315}, 0, 0, 0, 0)
	setOSD(t, dir, 316, 0)
	m := loadPadMapping(dir, "045e_028e", buttonBits(305, 307, 308, 310, 311, 314, 315, 316), axisBits(16, 17))
	if m.direct || !m.Mapped || m.Code != 315 || len(m.Keys) != 1 || m.Keys[315] != platform.KeyStart || m.osd[0] != 0 || m.info("", "", 0, 0).Direct {
		t.Fatalf("mapping: %+v", m)
	}
	d := &device{pad: true, mapping: m, held: map[uint16]platform.Key{}, abs: map[uint16]absInfo{}, axisEdge: map[uint16]uint8{}}
	if k, ok := d.inputKey(305); !ok || k != platform.KeyOther {
		t.Fatal("A should be left to Main:", k, ok)
	}
	if k, ok := d.inputKey(315); !ok || k != platform.KeyStart {
		t.Fatal("Start is still read here:", k, ok)
	}
	if _, ok := d.osdKey(316, true); ok {
		t.Fatal("the menu button belongs to Main for a pad that is not held")
	}
}
