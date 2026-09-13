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
// for back), which carry no pad name. Reading the pad directly by slot
// keeps every pad apart; the OK / BACK choice itself is passed on with the
// pad's description (support.Pad), and the app decides per pad whether to
// follow it (Options -> OK button).
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
	osd          [2]uint16 // slots 21/22: the MiSTer menu button, or the two buttons of its combo; 0 when unreadable here
	okBack       [2]uint16 // slot 23: MENU OK and MENU BACK, Main's own menu confirm and back; 0 when unset
	direct       bool      // A and B are readable: the pad is held exclusively and read here entirely
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

// slotKeys is what each define-slot does in the app. A and B match the
// Enter and back keys Main sends for them; X and Y are the app's own
// assignment. Main's script translation (input.cpp joy_digital) sends
// Backspace for X and Tab for Y, so a pad with no map file, which reaches
// the app only through that translation, gets Filters on Y and erase on X.
var slotKeys = map[string]platform.Key{
	"Right": platform.KeyRight, "Left": platform.KeyLeft, "Down": platform.KeyDown, "Up": platform.KeyUp,
	"A": platform.KeyEnter, "B": platform.KeyBack, "X": platform.KeyTab, "Y": platform.KeySpace,
	"L": platform.KeyPageUp, "R": platform.KeyPageDown, "Select": platform.KeySelect, "Start": platform.KeyStart,
}

// priority is the slot order that wins when two slots share a code.
var priority = []string{"Start", "A", "B", "X", "Y", "L", "R", "Select", "Right", "Left", "Down", "Up"}

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
	m := loadPadMapping("/media/fat/config", fmt.Sprintf("%04x_%04x", id[1], id[2]), bits, abs)
	if !m.Mapped {
		return fallbackPad(bits, abs, id[0] == busVirtual || devPhys(f) == "")
	}
	return m
}

// A uinput device such as the Zaparoo pad is Main's: it may claim any bus
// type (Zaparoo says USB), but it has no physical path, where a real pad
// reports its USB port or Bluetooth address.
const busVirtual = 0x06 // BUS_VIRTUAL

func eviocgphys(n int) uintptr { return uintptr(0x80000000) | uintptr(n)<<16 | uintptr(0x45)<<8 | 0x07 }

// devPhys is the device's physical path (EVIOCGPHYS): "" for a virtual one.
func devPhys(f *os.File) string {
	buf := make([]byte, 256)
	if err := ioctl(f.Fd(), eviocgphys(len(buf)), unsafe.Pointer(&buf[0])); err != nil {
		return ""
	}
	n := 0
	for n < len(buf) && buf[n] != 0 {
		n++
	}
	return string(buf[:n])
}

// Linux gamepad codes (linux/input-event-codes.h), by position: the
// kernel's gamepad drivers name the face buttons by where they sit.
const (
	btnSouth  = 304 // bottom
	btnEast   = 305 // right
	btnNorth  = 307 // top
	btnWest   = 308 // left
	btnTL     = 310
	btnTR     = 311
	btnSelect = 314
	btnMode   = 316 // the home / guide button
	absX      = 0   // the left stick
	absY      = 1
	absHat0X  = 16 // the d-pad
	absHat0Y  = 17
)

// fallbackPad is the mapping for a pad with no MiSTer map file: the Linux
// default layout when the node reports the standard gamepad buttons, else
// Start alone, the rest through Main. A virtual device is never held.
func fallbackPad(bits [96]byte, abs [8]byte, virtual bool) padMapping {
	if !virtual {
		if m, ok := linuxLayout(bits, abs); ok {
			return m
		}
	}
	return defaultStart(bits)
}

// linuxLayout reads a pad MiSTer has not defined by the standard Linux
// gamepad layout, held exclusively like a defined pad so that its home
// button is the app's Menu button rather than Main's OSD. The slots follow
// MiSTer's positional define (A right, B bottom, X top, Y left), and the
// bottom button is its OK, as in Main's own default and on every pad that
// prints letters, so B confirms unless Options say otherwise. The hat and
// the left stick move; the shoulders page. It needs the bottom and right
// buttons at least; a pad without them stays with Main.
func linuxLayout(bits [96]byte, abs [8]byte) (padMapping, bool) {
	if !hasButton(bits, btnSouth) || !hasButton(bits, btnEast) {
		return padMapping{}, false
	}
	m := padMapping{Source: "Linux default layout", Keys: map[uint16]platform.Key{}, Slots: map[string]uint16{}, direct: true}
	for slot, code := range map[string]uint32{"A": btnEast, "B": btnSouth, "X": btnNorth, "Y": btnWest, "L": btnTL, "R": btnTR, "Select": btnSelect, "Start": btnStart} {
		if usable(code, bits, abs) {
			m.Slots[slot] = uint16(code)
		}
	}
	if hasAxis(abs, absHat0X) && hasAxis(abs, absHat0Y) {
		m.Slots["Right"], m.Slots["Left"] = AxisCode(absHat0X, true), AxisCode(absHat0X, false)
		m.Slots["Down"], m.Slots["Up"] = AxisCode(absHat0Y, true), AxisCode(absHat0Y, false)
	} else {
		for slot, code := range map[string]uint32{"Up": 544, "Down": 545, "Left": 546, "Right": 547} { // BTN_DPAD_*
			if usable(code, bits, abs) {
				m.Slots[slot] = uint16(code)
			}
		}
	}
	m.assignKeys()
	if hasButton(bits, btnMode) {
		m.osd = [2]uint16{btnMode, btnMode}
	}
	m.okBack = [2]uint16{btnSouth, btnEast}
	if hasAxis(abs, absX) && hasAxis(abs, absY) {
		m.stickAxes, m.stickAxesN = [4]uint16{absX, absY}, 2
		m.menuX, m.menuY, m.menu = absX, absY, [2]bool{true, true}
		m.setDefault(AxisCode(absX, true), platform.KeyRight)
		m.setDefault(AxisCode(absX, false), platform.KeyLeft)
		m.setDefault(AxisCode(absY, true), platform.KeyDown)
		m.setDefault(AxisCode(absY, false), platform.KeyUp)
	}
	if hasButton(bits, btnStart) {
		m.Code = btnStart
	}
	return m, true
}

// assignKeys gives every slot's code its app key, the higher priority
// slot winning when two share a code.
func (m *padMapping) assignKeys() {
	for i := len(priority) - 1; i >= 0; i-- { // lowest priority first, so higher ones overwrite
		slot := priority[i]
		if code, ok := m.Slots[slot]; ok {
			m.Keys[code] = slotKeys[slot]
		}
	}
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
		m.assignKeys()
		// The MiSTer menu (OSD) button: slot 21, and slot 22 for the second
		// button of a combo (Main sets it equal to the first when there is
		// none). While the pad is held it is the app's Menu button.
		for i := 0; i < 2; i++ {
			if v := binary.LittleEndian.Uint32(b[(21+i)*4:]); v != 0 && v < emuBase && usable(v, bits, abs) {
				m.osd[i] = uint16(v)
			}
		}
		if m.osd[1] == 0 {
			m.osd[1] = m.osd[0]
		}
		// Slot 23 (SYS_BTN_MENU_FUNC) packs Main's own menu buttons: MENU OK
		// in the low half, MENU BACK in the high half. They are kept as the
		// user's word for which button confirms, not acted on here.
		if v := binary.LittleEndian.Uint32(b[23*4:]); v != 0 {
			m.okBack = [2]uint16{uint16(v & 0xFFFF), uint16(v >> 16)}
			if m.okBack[0] == m.okBack[1] { // one button for both means nothing was chosen
				m.okBack = [2]uint16{}
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
		// Only a pad whose A and B are both readable here is taken over
		// (held exclusively, Main's translation dropped): with either left to
		// Main the pad would lose its back button, so such a pad stays on
		// Main's translation like one without a map, giving up Start only.
		_, hasA := m.Slots["A"]
		_, hasB := m.Slots["B"]
		m.direct = hasA && hasB
		if !m.direct {
			m.Keys = map[uint16]platform.Key{}
			if m.Code != 0 {
				m.Keys[m.Code] = platform.KeyStart
			}
			m.osd = [2]uint16{}
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
	p := support.Pad{Node: node, Name: name, Vendor: vendor, Product: product, Map: m.Source, Note: m.Note, Mapped: m.Mapped, Direct: m.direct, Slots: map[string]uint16{}}
	for k, v := range m.Slots {
		p.Slots[k] = v
	}
	if m.Code != 0 {
		p.Slots["Start"] = m.Code
	}
	if m.menu[0] {
		p.MenuStick = fmt.Sprintf("axes %d/%d", m.menuX&0xFFFF, m.menuY&0xFFFF)
	}
	if m.osd[0] != 0 {
		p.Menu = fmt.Sprint(m.osd[0])
		if m.osd[1] != m.osd[0] {
			p.Menu += "+" + fmt.Sprint(m.osd[1])
		}
	}
	p.OK, p.Back = m.okBackName(p, 0), m.okBackName(p, 1)
	return p
}

// okBackName names MENU OK (0) or MENU BACK (1) by the slot it sits in, by
// its code when it is in none, or "" when unset.
func (m padMapping) okBackName(p support.Pad, i int) string {
	code := m.okBack[i]
	if code == 0 {
		return ""
	}
	if slot := p.Slot(code); slot != "" {
		return slot
	}
	if code < emuBase {
		return fmt.Sprintf("btn %d", code)
	}
	return support.CodeText(code)
}
