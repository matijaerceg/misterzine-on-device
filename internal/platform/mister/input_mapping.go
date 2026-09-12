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

// padMapping is what the app reads from a gamepad node itself. A pad with a
// MiSTer map file is read entirely by the slots the user defined in Main's
// define-buttons screen: d-pad, A B X Y, L R, Select, Start, and the menu
// stick. A pad without one only gives up Start (Main never forwards it);
// its other buttons reach the app through Main's virtual keyboard.
//
// Main names the face slots A, B, X and Y but, while a script runs, turns
// the pad into keys by its MENU OK / MENU BACK choice (Enter for OK, Esc
// for back). A user who put OK on B therefore gets "A" and "B" swapped in
// the app if it trusts those keys. Reading the pad directly by slot keeps
// the app's A the button the user called A, on every pad at once.
//
// Map codes: a button is its evdev code (256 and up); an axis edge is
// emuBase + axis*2 + direction (0 = towards minimum, 1 = towards maximum),
// the synthetic code Main gives an axis pushed past its threshold; the
// d-pad hat shows up as 800..803. Slots 28/29 hold the axis numbers of the
// stick that moves Main's own menu; the app follows them the same way.
type padMapping struct {
	Code         uint16 // Start; 0 when unavailable
	Source, Note string
	Mapped       bool                    // a MiSTer map file was read
	Keys         map[uint16]platform.Key // code (button or axis edge) -> app key
	Slots        map[string]uint16       // slot name -> code, for the pad tester
	stickAxes    [4]uint16               // axes in slots 24..27, whose negative edge also counts
	stickAxesN   int
	menuX, menuY uint16 // slots 28/29: the menu stick's axis numbers
	menu         [2]bool
}

const (
	emuBase   = 0x300 // Main's KEY_EMU
	slotRight = iota - 1
	slotLeft
	slotDown
	slotUp
	slotA
	slotB
	slotX
	slotY
	slotL
	slotR
	slotSelect
	slotStart
)

// slotNames is MiSTer's define order (map file indices 0..11).
var slotNames = []string{"Right", "Left", "Down", "Up", "A", "B", "X", "Y", "L", "R", "Select", "Start"}

// slotKeys is what each define-slot does in the app: the same keys Main
// would send for them when OK and back sit on A and B.
var slotKeys = map[string]platform.Key{
	"Right": platform.KeyRight, "Left": platform.KeyLeft, "Down": platform.KeyDown, "Up": platform.KeyUp,
	"A": platform.KeyEnter, "B": platform.KeyBack, "X": platform.KeyTab, "Y": platform.KeySpace,
	"L": platform.KeyPageUp, "R": platform.KeyPageDown, "Start": platform.KeyStart,
}

// priority is the slot order that wins when two slots share a code.
var priority = []string{"Start", "A", "B", "X", "Y", "L", "R", "Right", "Left", "Down", "Up"}

// AxisCode is the synthetic code for an axis edge, as Main forms it.
func AxisCode(axis uint16, positive bool) uint16 {
	c := emuBase + axis*2
	if positive {
		c++
	}
	return c
}

// IsAxisCode reports whether a code names an axis edge; axis and direction
// are those it names.
func IsAxisCode(code uint16) (axis uint16, positive, ok bool) {
	if code < emuBase {
		return 0, false, false
	}
	return (code - emuBase) >> 1, code&1 == 1, true
}

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
	var abs [8]byte // EVIOCGBIT(EV_ABS, 64 axes)
	ioctl(f.Fd(), uintptr(0x80000000)|uintptr(len(abs))<<16|uintptr(0x45)<<8|uintptr(0x20+3), unsafe.Pointer(&abs[0]))
	return loadPadMapping("/media/fat/config", fmt.Sprintf("%04x_%04x", id[1], id[2]), bits, abs)
}

func hasButton(bits [96]byte, code uint32) bool {
	return code > 0 && code < uint32(len(bits)*8) && bits[code/8]&(1<<uint(code%8)) != 0
}

func hasAxis(abs [8]byte, axis uint32) bool {
	return axis < uint32(len(abs)*8) && abs[axis/8]&(1<<uint(axis%8)) != 0
}

func defaultStart(bits [96]byte) padMapping {
	if hasButton(bits, btnStart) {
		return padMapping{Code: btnStart, Source: "Linux default"}
	}
	return padMapping{}
}

// usable reports whether a map code names something this node can report:
// a raw button it has, or an edge of an axis it has. Keyboard codes and
// buttons of another interface are left to Main.
func usable(code uint32, bits [96]byte, abs [8]byte) bool {
	if code >= emuBase {
		return code < 0x10000 && hasAxis(abs, (code-emuBase)>>1)
	}
	return code >= 256 && hasButton(bits, code)
}

func loadPadMapping(configDir, id string, bits [96]byte, abs [8]byte) padMapping {
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
		m.Keys, m.Slots = map[uint16]platform.Key{}, map[string]uint16{}
		for i, slot := range slotNames {
			code := binary.LittleEndian.Uint32(b[i*4:])
			if usable(code, bits, abs) {
				m.Slots[slot] = uint16(code)
			}
		}
		for i := len(priority) - 1; i >= 0; i-- { // lowest priority first, so higher ones overwrite
			slot := priority[i]
			if code, ok := m.Slots[slot]; ok {
				m.Keys[code] = slotKeys[slot]
			}
		}
		for i := 0; i < 4; i++ { // SYS_AXIS1_X..SYS_AXIS2_Y: the sticks' axis numbers
			if v := binary.LittleEndian.Uint32(b[(24+i)*4:]); v != 0 {
				m.stickAxes[m.stickAxesN] = uint16(v)
				m.stickAxesN++
			}
		}
		if v := binary.LittleEndian.Uint32(b[28*4:]); v != 0 && hasAxis(abs, v&0xFFFF) { // SYS_AXIS_X: the menu stick
			m.menuX, m.menu[0] = uint16(v), true
			m.setDefault(AxisCode(uint16(v), true), platform.KeyRight)
			m.setDefault(AxisCode(uint16(v), false), platform.KeyLeft)
		}
		if v := binary.LittleEndian.Uint32(b[29*4:]); v != 0 && hasAxis(abs, v&0xFFFF) { // SYS_AXIS_Y
			m.menuY, m.menu[1] = uint16(v), true
			m.setDefault(AxisCode(uint16(v), true), platform.KeyDown)
			m.setDefault(AxisCode(uint16(v), false), platform.KeyUp)
		}
		code := binary.LittleEndian.Uint32(b[slotStart*4:])
		if code == 0 {
			m.Note = "Start is unassigned in MiSTer"
		} else if code < 256 || !hasButton(bits, code) {
			m.Note = "Mapped Start is unavailable on this input device"
			delete(m.Slots, "Start") // an axis edge as Start is not a launch button
		} else {
			m.Code = uint16(code)
		}
		// A saved assignment replaces the default, including unassigned or
		// unsupported entries. Never launch from somebody's reassigned Select.
		return m
	}
	return defaultStart(bits)
}

func (m *padMapping) setDefault(code uint16, k platform.Key) {
	if _, taken := m.Keys[code]; !taken {
		m.Keys[code] = k
	}
}

// bothEdges reports whether an axis reports its negative edge as well as
// its positive one: Main counts the low end only for stick axes, so a
// trigger resting at its minimum is not a held button.
func (m padMapping) bothEdges(axis uint16) bool {
	for i := 0; i < m.stickAxesN; i++ {
		if m.stickAxes[i]&0xFFFF == axis {
			return true
		}
	}
	return (m.menu[0] && m.menuX&0xFFFF == axis) || (m.menu[1] && m.menuY&0xFFFF == axis)
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
	if m.menu[0] {
		p.MenuStick = fmt.Sprintf("axes %d/%d", m.menuX&0xFFFF, m.menuY&0xFFFF)
	}
	return p
}
