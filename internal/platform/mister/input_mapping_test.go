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

func buttonBits(codes ...uint16) (bits [96]byte) {
	for _, code := range codes {
		bits[code/8] |= 1 << (code % 8)
	}
	return
}

func writeStartMap(t *testing.T, dir, name string, code uint32) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 128)
	binary.LittleEndian.PutUint32(b[10*4:], 315) // reproduced Brook Select
	binary.LittleEndian.PutUint32(b[11*4:], code)
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, b, 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMiSTerStartMapping(t *testing.T) {
	for _, tt := range []struct {
		name string
		code uint32
		bits [96]byte
		want uint16
		note bool
	}{
		{"brook start replaces select", 314, buttonBits(314, 315), 314, false},
		{"encoder without standard start", 299, buttonBits(299), 299, false},
		{"explicitly unassigned", 0, buttonBits(315), 0, true},
		{"different interface", 314, buttonBits(315), 0, true},
		{"axis mapping", 800, buttonBits(315), 0, true},
		{"keyboard mapping is not a raw pad button", 28, buttonBits(28, 315), 0, true},
		{"wide invalid code must not truncate", 0x100013b, buttonBits(315), 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			p := writeStartMap(t, filepath.Join(dir, "inputs"), "input_0f0d_00aa_v3.map", tt.code)
			m := loadPadMapping(dir, "0f0d_00aa", tt.bits, [8]byte{})
			if m.Code != tt.want || m.Source != p || (m.Note != "") != tt.note {
				t.Fatalf("mapping: %+v", m)
			}
		})
	}
}

func TestMiSTerStartFileSelection(t *testing.T) {
	dir := t.TempDir()
	bits := buttonBits(299, 314, 315)
	if m := loadPadMapping(dir, "0f0d_00aa", bits, [8]byte{}); m.Code != 315 || m.Source != "Linux default" {
		t.Fatal(m)
	}
	if m := loadPadMapping(dir, "0f0d_00aa", buttonBits(299), [8]byte{}); m.Code != 0 {
		t.Fatal(m)
	}
	writeStartMap(t, filepath.Join(dir, "inputs"), "input_1234_5678_v3.map", 299)
	writeStartMap(t, filepath.Join(dir, "inputs"), "galaga_input_0f0d_00aa_v3.map", 299)
	writeStartMap(t, filepath.Join(dir, "inputs"), "input_0f0d_00aa_deadbeef_v3.map", 299)
	if m := loadPadMapping(dir, "0f0d_00aa", bits, [8]byte{}); m.Code != 315 {
		t.Fatal(m)
	}
	legacy := writeStartMap(t, dir, "input_0f0d_00aa_v3.map", 299)
	if m := loadPadMapping(dir, "0f0d_00aa", bits, [8]byte{}); m.Code != 299 || m.Source != legacy {
		t.Fatal(m)
	}
	primary := writeStartMap(t, filepath.Join(dir, "inputs"), "input_0f0d_00aa_v3.map", 314)
	if m := loadPadMapping(dir, "0f0d_00aa", bits, [8]byte{}); m.Code != 314 || m.Source != primary {
		t.Fatal(m)
	}
	for _, size := range []int{0, 44, 127, 129, 4096} {
		if err := os.WriteFile(primary, make([]byte, size), 0644); err != nil {
			t.Fatal(err)
		}
		if m := loadPadMapping(dir, "0f0d_00aa", bits, [8]byte{}); m.Code != 0 || m.Note == "" || m.Source != primary {
			t.Fatal(m)
		}
	}
}

// Exercise the actual raw reader: Select and face buttons must not produce a
// launch or duplicate Main's translated actions. MiSTer uses 16-byte events.
func TestMappedStartRawEvents(t *testing.T) {
	dir := t.TempDir()
	writeStartMap(t, filepath.Join(dir, "inputs"), "input_0f0d_00aa_v3.map", 314)
	m := loadPadMapping(dir, "0f0d_00aa", buttonBits(305, 314, 315), [8]byte{})
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	in := &Input{ch: make(chan platform.Event, 16), log: log.New(io.Discard, "", 0), stop: make(chan struct{}), devs: map[string]*device{}}
	d := &device{f: r, pad: true, mapping: m, held: map[uint16]platform.Key{}, name: "Brook fixture"}
	in.wg.Add(1)
	go in.read(d)
	for _, pair := range [][2]uint16{{315, 1}, {315, 0}, {305, 1}, {305, 0}, {314, 1}, {314, 2}, {314, 0}, {314, 1}} {
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
	// 305 is not in the map, so it arrives as an actionless pad button (for
	// the pad tester); 315 is the reassigned Select, the quick-toggle
	// modifier, and must not be Start
	var starts, others []platform.Event
	for len(in.ch) > 0 {
		e := <-in.ch
		if e.Key == platform.KeyStart {
			starts = append(starts, e)
		} else {
			others = append(others, e)
		}
	}
	if len(starts) != 4 {
		t.Fatalf("got %d Start events, want two presses/releases including disconnect release", len(starts))
	}
	for i, e := range starts {
		if e.Code != 314 || e.Pressed != (i%2 == 0) {
			t.Fatal(e)
		}
	}
	for _, e := range others {
		want := platform.KeyOther
		if e.Code == 315 {
			want = platform.KeySelect
		}
		if e.Key != want || (e.Code != 305 && e.Code != 315) {
			t.Fatal(e)
		}
	}
	virtual := device{name: "MiSTer virtual input"}
	if key, ok := virtual.inputKey(keyEnter); !ok || key != platform.KeyEnter {
		t.Fatal(key, ok)
	}
	standard := device{pad: true, mapping: defaultStart(buttonBits(314, 315))}
	if key, ok := standard.inputKey(315); !ok || key != platform.KeyStart {
		t.Fatal(key, ok)
	}
	if key, _ := standard.inputKey(314); key == platform.KeyStart {
		t.Fatal("unmapped314 must not launch")
	}
}
