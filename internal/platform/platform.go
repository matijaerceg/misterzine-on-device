// Package platform is the contract between the app and the machine it runs
// on: a display to present frames to, an input event stream, a way to send
// commands to MiSTer Main, and a clock. The MiSTer implementation lives in
// platform/mister (Linux only); platform/headless renders to PNG files for
// the PC harness and the golden tests.
package platform

import (
	"image"
	"time"
)

// Key is a logical input key. Main turns the gamepad into these while a
// script runs: d-pad = arrows, A = Enter, B = Back, Y = Space, X = Tab,
// L = PageUp, R = PageDown. Real keyboards deliver the same set plus more.
type Key int

const (
	KeyNone Key = iota
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeyBack // Esc
	KeySpace
	KeyTab
	KeyPageUp
	KeyPageDown
	KeyHome
	KeyEnd
	KeyScreenshot // F12 on a keyboard, or the debug channel
	KeyStart      // the pad's Start button (read from the pad itself: Main does not forward it)
	KeyOther      // anything else; Event.Code says what
	KeyBackspace
	KeySelect // the pad's Select button: held on the list, a modifier for the quick toggles
)

var keyNames = map[Key]string{
	KeyNone: "none", KeyUp: "up", KeyDown: "down", KeyLeft: "left", KeyRight: "right",
	KeyEnter: "enter", KeyBack: "back", KeySpace: "space", KeyTab: "tab",
	KeyPageUp: "pageup", KeyPageDown: "pagedown", KeyHome: "home", KeyEnd: "end",
	KeyScreenshot: "screenshot", KeyStart: "start", KeyOther: "other", KeyBackspace: "backspace", KeySelect: "select",
}

func (k Key) String() string {
	if s, ok := keyNames[k]; ok {
		return s
	}
	return "key?"
}

// ParseKey maps a key name (as used by scripts and the debug channel) back
// to a Key; KeyNone when unknown.
func ParseKey(name string) Key {
	for k, s := range keyNames {
		if s == name {
			return k
		}
	}
	switch name {
	case "esc", "escape", "b":
		return KeyBack
	case "a", "return":
		return KeyEnter
	case "pgup":
		return KeyPageUp
	case "pgdn", "pgdown":
		return KeyPageDown
	}
	return KeyNone
}

// Event is one key press or release.
type Event struct {
	Key     Key
	Text    rune   // printable keyboard character; zero for gamepad/navigation input
	Code    uint16 // raw evdev code
	Pressed bool
	At      time.Time
	Source  string // device name, "script", "debug"
}

// Display shows frames. Present copies the given rectangles of the canvas
// (nil = everything) to the screen; the canvas is the PHYSICAL frame, already
// rotated for tate by the app.
type Display interface {
	Size() (w, h int)
	Present(canvas *image.RGBA, dirty []image.Rectangle) error
	Close() error
}

// Input delivers key events until Close.
type Input interface {
	Events() <-chan Event
	Close() error
}

// Commander sends one command line to MiSTer Main (/dev/MiSTer_cmd).
type Commander interface {
	Send(line string) error
}

// Clock tells the time and whether it can be trusted: a MiSTer without an
// RTC and without network boots into the past, and relative times must then
// stay hidden.
type Clock struct {
	Now     func() time.Time
	Trusted bool
}

// Platform bundles what the app needs from the machine.
type Platform struct {
	Display Display
	Input   Input
	Cmd     Commander
	Clock   Clock
	Root    string // config dir, /media/fat/misterzine on device
	Card    string // card root, /media/fat on device
	IniPath string // MiSTer.ini path, "" when none
}
