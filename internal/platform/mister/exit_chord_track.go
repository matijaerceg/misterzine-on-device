package mister

import "time"

// The exit chord: buttons held together on a pad, inside a game MisterZine
// launched, to leave it for the MiSTer menu (and, with Return after game
// on, for MisterZine). The buttons are named by MiSTer define-slot, so the
// chord is the same on every pad the user has defined.
const (
	ExitChordOff             = ""                // Options -> Exit chord: off
	ExitChordSelectStart     = "select-start"    // Select+Start
	ExitChordShouldersSelect = "lr-select-start" // L+R+Select+Start

	// ExitChordHold is how long the chord is held before it fires: long
	// enough that a pass over the buttons does nothing, short enough to
	// stay clear of the power-off hold of wireless pads.
	ExitChordHold = time.Second
)

// ExitChordSlots names the define-slots the variant needs, or nil when off
// or unknown.
func ExitChordSlots(variant string) []string {
	switch variant {
	case ExitChordSelectStart:
		return []string{"Select", "Start"}
	case ExitChordShouldersSelect:
		return []string{"L", "R", "Select", "Start"}
	}
	return nil
}

// ExitChordName is the Options value for a variant.
func ExitChordName(variant string) string {
	switch variant {
	case ExitChordSelectStart:
		return "Select+Start"
	case ExitChordShouldersSelect:
		return "L+R+Select+Start"
	}
	return "off"
}

// chordHold turns a run of "all chord buttons are down" samples into one
// firing per hold: it fires once the buttons have stayed down for hold,
// then not again until they have all been released.
type chordHold struct {
	hold  time.Duration
	since time.Time // zero while not all down
	fired bool
}

// update takes one sample and reports whether the chord fires on it.
func (c *chordHold) update(allDown bool, now time.Time) bool {
	if !allDown {
		c.since, c.fired = time.Time{}, false
		return false
	}
	if c.since.IsZero() {
		c.since = now
	}
	if c.fired || now.Sub(c.since) < c.hold {
		return false
	}
	c.fired = true
	return true
}
