//go:build linux

package mister

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

// padMapping is what the app reads from a gamepad node itself: the Start
// button (Main never forwards it) and, when the pad has a MiSTer map file,
// the four face buttons by the slot the user defined them in.
//
// Main names the slots A, B, X and Y in its define-buttons screen but, while
// a script runs, turns the pad into keys by its MENU OK / MENU BACK choice
// (Enter for OK, Esc for back, Tab and Space for X and Y). A user who put
// OK on B therefore gets "A" and "B" swapped in the app if it trusts those
// keys. Reading the pad directly by define-slot keeps the app's A the button
// the user called A, on every pad at once, since each pad has its own map.
type padMapping struct {
	Code         uint16 // Start; 0 when unavailable
	Source, Note string
	Mapped       bool                    // a MiSTer map file was read
	Face         map[uint16]platform.Key // face button code -> app key
	Slots        map[string]uint16       // slot name -> code, for the pad tester
}

// Slot names in MiSTer's define order (map file indices 4..11).
var slotNames = []string{"A", "B", "X", "Y", "L", "R", "Select", "Start"}

// faceKeys is what each define-slot does in the app: the same keys Main
// would send for them when OK and back sit on A and B.
var faceKeys = map[string]platform.Key{"A": platform.KeyEnter, "B": platform.KeyBack, "X": platform.KeyTab, "Y": platform.KeySpace}

// devicePad reads the ordinary global v3 controller map for the device.
// Do not guess at per-core, hashed per-device or alternate-mode map files.
func devicePad(f *os.File) padMapping {
	var bits [96]byte
	if ioctl(f.Fd(), eviocgbitKey(len(bits)), unsafe.Pointer(&bits[0])) != nil {
		return padMapping{}
	}
	var id [4]uint16
	if ioctl(f.Fd(), 0x80084502, unsafe.Pointer(&id[0])) != nil { // EVIOCGID
		return defaultStart(bits)
	}
	return loadPadMapping("/media/fat/config", fmt.Sprintf("%04x_%04x", id[1], id[2]), bits)
}

func hasButton(bits [96]byte, code uint32) bool {
	return code > 0 && code < uint32(len(bits)*8) && bits[code/8]&(1<<uint(code%8)) != 0
}

func defaultStart(bits [96]byte) padMapping {
	if hasButton(bits, btnStart) {
		return padMapping{Code: btnStart, Source: "Linux default"}
	}
	return padMapping{}
}

func loadPadMapping(configDir, id string, bits [96]byte) padMapping {
	name := "input_" + id + "_v3.map"
	for _, dir := range []string{filepath.Join(configDir, "inputs"), configDir} {
		path := filepath.Join(dir, name)
		f, err := os.Open(path)
		if os.IsNotExist(err) {
			continue
		}
		m := padMapping{Source: path}
		if err != nil {
			m.Note = "Cannot read MiSTer Start mapping"
			return m
		}
		b, err := io.ReadAll(io.LimitReader(f, 129))
		f.Close()
		if err != nil || len(b) != 128 { // NUMBUTTONS=32, uint32_t entries
			m.Note = "Invalid MiSTer controller mapping"
			return m
		}
		m.Mapped = true
		m.Face, m.Slots = map[uint16]platform.Key{}, map[string]uint16{}
		for i, slot := range slotNames {
			code := binary.LittleEndian.Uint32(b[(4+i)*4:])
			// only a raw button this node has counts; keyboard codes, axes
			// and buttons of another interface are left to Main
			if code < 256 || !hasButton(bits, code) {
				continue
			}
			m.Slots[slot] = uint16(code)
			if k, ok := faceKeys[slot]; ok {
				m.Face[uint16(code)] = k
			}
		}
		code := binary.LittleEndian.Uint32(b[11*4:]) // SYS_BTN_START
		if code == 0 {
			m.Note = "Start is unassigned in MiSTer"
		} else if code < 256 || !hasButton(bits, code) {
			m.Note = "Mapped Start is unavailable on this input device"
		} else {
			m.Code = uint16(code)
		}
		// A saved assignment replaces the default, including unassigned or
		// unsupported entries. Never launch from somebody's reassigned Select.
		return m
	}
	return defaultStart(bits)
}

// info describes the mapping for the pad tester.
func (m padMapping) info(node, name string, vendor, product uint16) support.Pad {
	p := support.Pad{Node: node, Name: name, Vendor: vendor, Product: product, Map: m.Source, Note: m.Note, Mapped: m.Mapped, Slots: map[string]uint16{}}
	for k, v := range m.Slots {
		p.Slots[k] = v
	}
	if m.Code != 0 {
		p.Slots["Start"] = m.Code
	}
	return p
}
