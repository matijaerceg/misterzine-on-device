// Package support holds opt-in, bounded controller-test evidence. Nothing is
// collected until a test starts; reports contain no typed text or credentials.
package support

import (
	"fmt"
	"sync"
	"time"
)

const (
	MaxDevices = 32
	MaxSignals = 64
)

type Device struct {
	Node, Name                       string
	Bus, Vendor, Product, Version    uint16
	Keyboard, StandardStart, Virtual bool
	Error                            string
	StartCode                        uint16 `json:",omitempty"`
	StartMapping, StartNote          string `json:",omitempty"`
}

// Route describes the normal reader, not the more inclusive diagnostic reader.
func (d Device) Route() string {
	if d.Error != "" {
		return "Cannot read: " + d.Error
	}
	if d.Virtual {
		return "MiSTer translated input"
	}
	if d.StartMapping != "" {
		if d.StartCode == 0 {
			return d.StartNote
		}
		return fmt.Sprintf("Start button: %d", d.StartCode)
	}
	if d.Keyboard {
		return "Keyboard input"
	}
	if d.StandardStart {
		return "Standard Start only"
	}
	return "Skipped by normal input reader"
}

// Pad is a gamepad as the normal input reader sees it, for the pad tester:
// which raw button code sits in each MiSTer define-slot (A, B, X, Y, L, R,
// Select, Start) and where that came from.
type Pad struct {
	Node, Name      string
	Vendor, Product uint16
	Mapped          bool              // a MiSTer map file was read
	Direct          bool              // A and B are readable: the pad is held and read by define-slot; otherwise only Start is, the rest through Main
	Slots           map[string]uint16 // slot name -> code (a button code, or an axis edge 0x300+axis*2+direction)
	Map, Note       string
	MenuStick       string // the stick that moves Main's menu, when defined ("axes 0/1")
	Menu            string // the MiSTer menu button's code ("316"), or "a+b" for a combo, when the app reads it as its Menu button
}

// SlotOrder is how the tester lists the define-slots.
var SlotOrder = []string{"Up", "Down", "Left", "Right", "A", "B", "X", "Y", "L", "R", "Select", "Start"}

// Slot names the define-slot a code sits in, or "" when it is none of them.
func (p Pad) Slot(code uint16) string {
	for _, name := range SlotOrder {
		if c, ok := p.Slots[name]; ok && c == code {
			return name
		}
	}
	return ""
}

// CodeText prints a map code: a button number, or an axis edge as "ax2+".
func CodeText(code uint16) string {
	if code >= 0x300 {
		dir := "-"
		if code&1 == 1 {
			dir = "+"
		}
		return fmt.Sprintf("ax%d%s", (code-0x300)>>1, dir)
	}
	return fmt.Sprint(code)
}

type Signal struct {
	Node                      string
	Type, Code                uint16
	Down, Up, Repeat, Changes int
	Value                     int32
}

type Launch struct {
	Game, Target, Result, Detail, Version string
}

type Report struct {
	Schema                                  int
	Version, Kernel, Display, Data, Created string
	Tested, Interrupted                     bool
	Devices                                 []Device
	Signals                                 []Signal
	StartDown, StartUp                      int
	Dropped                                 int
	Launch                                  Launch
	Note                                    string
}

func (r Report) HasResult() bool { return r.Tested || r.Launch.Result != "" || r.Note != "" }

func (r Report) ButtonReceived() bool {
	for _, s := range r.Signals {
		if s.Type == 1 && s.Down > 0 {
			return true
		}
	}
	return false
}

func (r Report) Conclusion() string {
	if r.Interrupted {
		return "Test stopped early. Please retry."
	}
	if r.Dropped > 0 {
		return "Capture incomplete. Please retry."
	}
	if r.StartDown > 0 && r.StartUp == 0 {
		return "Start received; release not seen."
	}
	if r.StartDown > 0 {
		return "Start recognised. Try the launch test."
	}
	if r.ButtonReceived() {
		return "Button received; not recognised as Start."
	}
	for _, s := range r.Signals {
		if s.Type == 1 && s.Up > 0 {
			return "Release received; press again after GO."
		}
	}
	if len(r.Signals) > 0 {
		return "Movement received; no Start button seen."
	}
	return "No button received. Check the device pages."
}

// Capture accepts events from separate raw readers and the normal app reader.
// The arming interval discards the button used to enter the test.
type Capture struct {
	mu          sync.Mutex
	from, until time.Time
	report      Report
}

func NewCapture(r Report, from, until time.Time) *Capture {
	r.Schema, r.Tested = 1, true
	return &Capture{from: from, until: until, report: r}
}

func (c *Capture) active(at time.Time) bool { return !at.Before(c.from) && at.Before(c.until) }

func (c *Capture) AddDevice(d Device) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.report.Devices) >= MaxDevices {
		c.report.Dropped++
		return false
	}
	c.report.Devices = append(c.report.Devices, d)
	return true
}

func (c *Capture) DeviceError(node, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.report.Devices {
		if c.report.Devices[i].Node == node {
			c.report.Devices[i].Error = message
			return
		}
	}
}

func (c *Capture) Record(node string, typ, code uint16, value int32, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.active(at) {
		return
	}
	if typ == 0 && code == 3 {
		c.report.Dropped++
		return
	}
	if typ != 1 && typ != 3 {
		return
	}
	if typ == 1 && (value < 0 || value > 2) {
		return
	}
	index := -1
	for i, s := range c.report.Signals {
		if s.Node == node && s.Type == typ && s.Code == code {
			index = i
			break
		}
	}
	if index < 0 {
		if len(c.report.Signals) == MaxSignals {
			c.report.Dropped++
			return
		}
		index = len(c.report.Signals)
		c.report.Signals = append(c.report.Signals, Signal{Node: node, Type: typ, Code: code})
	}
	s := &c.report.Signals[index]
	s.Value = value
	if typ == 3 {
		s.Changes++
		return
	}
	switch value {
	case 0:
		s.Up++
	case 1:
		s.Down++
	case 2:
		s.Repeat++
	}
}

func (c *Capture) RecordStart(pressed bool, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.active(at) {
		return
	}
	if pressed {
		c.report.StartDown++
	} else {
		c.report.StartUp++
	}
}

func (c *Capture) Snapshot() Report {
	c.mu.Lock()
	defer c.mu.Unlock()
	r := c.report
	r.Devices = append([]Device(nil), r.Devices...)
	r.Signals = append([]Signal(nil), r.Signals...)
	return r
}
