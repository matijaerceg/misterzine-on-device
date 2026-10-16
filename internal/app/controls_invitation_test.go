package app

import (
	"bytes"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"testing"
	"time"
)

func TestControlInvitationConsumesOpeningAndStartingHolds(t *testing.T) {
	a, c, pads, saves := mappingApp()
	a.screen = ScreenList
	e := rawControl((*pads)[0], 304, platform.KeyEnter)
	e.Pressed = true
	a.Handle(e)
	*c = c.Add(2 * time.Second)
	a.Tick(*c)
	if a.mapping.mode != "invitation" {
		t.Fatal("opening hold began setup")
	}
	e.Pressed = false
	a.Handle(e)
	checkPress(a, c, 304, true)
	*c = c.Add(time.Second)
	a.Tick(*c)
	if a.mapping.mode != "basic" {
		t.Fatal("hold did not begin")
	}
	checkPress(a, c, 304, true)
	checkPress(a, c, 304, false)
	if a.mapping.check.step != 0 || *saves != 0 {
		t.Fatal("starting hold assigned")
	}
	mappingTap(a, c, rawControl((*pads)[0], 801, platform.KeyOther))
	if a.mapping.check.step != 1 {
		t.Fatal("first fresh input ignored")
	}
}
func TestControlInvitationChordNeverDowngradesAndCanReinvite(t *testing.T) {
	a, c, pads, saves := mappingApp()
	a.screen = ScreenList
	mappingTap(a, c, rawControl((*pads)[0], 304, platform.KeyEnter))
	checkPress(a, c, 304, true)
	*c = c.Add(800 * time.Millisecond)
	checkPress(a, c, 305, true)
	checkPress(a, c, 305, false)
	*c = c.Add(2 * time.Second)
	a.Tick(*c)
	if a.mapping.mode != "invitation" {
		t.Fatal("partial chord began setup")
	}
	checkPress(a, c, 304, false)
	checkPress(a, c, 304, true)
	checkPress(a, c, 305, true)
	*c = c.Add(time.Second)
	a.Tick(*c)
	if a.screen == ScreenControls || *saves != 0 {
		t.Fatal("chord did not defer")
	}
	checkPress(a, c, 304, true)
	if a.screen == ScreenControls {
		t.Fatal("repeat reinvited")
	}
	checkPress(a, c, 305, false)
	mappingTap(a, c, rawControl((*pads)[0], 306, platform.KeyOther))
	if a.screen == ScreenControls {
		t.Fatal("invited before all released")
	}
	checkPress(a, c, 304, false)
	mappingTap(a, c, rawControl((*pads)[0], 304, platform.KeyEnter))
	if a.mapping.mode != "invitation" || a.screen != ScreenControls {
		t.Fatal("fresh press not invited")
	}
}
func TestControlInvitationForeignChordAndKeyboardEscape(t *testing.T) {
	a, c, pads, _ := mappingApp()
	a.screen = ScreenList
	mappingTap(a, c, rawControl((*pads)[0], 304, platform.KeyEnter))
	for _, code := range []uint16{30, 31} {
		e := rawControl((*pads)[1], code, platform.KeyOther)
		e.Pressed = true
		a.Handle(e)
	}
	*c = c.Add(time.Second)
	a.Tick(*c)
	if a.screen == ScreenControls {
		t.Fatal("other device could not defer")
	}
	for _, code := range []uint16{30, 31} {
		e := rawControl((*pads)[1], code, platform.KeyOther)
		a.Handle(e)
	}
	a.controlInput.current = controlID((*pads)[0])
	a.controlInput.node = (*pads)[0].Node
	a.OpenControls()
	mappingTap(a, c, rawControl((*pads)[1], 1, platform.KeyBack))
	if a.screen == ScreenControls {
		t.Fatal("separate Esc could not cancel basics")
	}
	a.controlInput.current = controlID((*pads)[1])
	a.controlInput.node = (*pads)[1].Node
	a.OpenControls()
	mappingTap(a, c, rawControl((*pads)[1], 1, platform.KeyBack))
	if a.mapping.check.step != 1 {
		t.Fatal("target Esc could not be assigned")
	}
}
func TestControlKeyboardFallbackTapHoldAndChord(t *testing.T) {
	a, c, pads, saves := mappingApp()
	a.screen = ScreenList
	mappingTap(a, c, rawControl((*pads)[1], 30, platform.KeyOther))
	if a.mapping.cursor != 0 {
		t.Fatal("trigger changed role")
	}
	mappingTap(a, c, rawControl((*pads)[1], 30, platform.KeyOther))
	if a.mapping.cursor != 1 {
		t.Fatal("tap did not switch role")
	}
	checkHold(a, c, 30)
	if a.mapping.mode != "basic" || a.mapping.check.step != 0 || *saves != 0 {
		t.Fatal("fallback did not start controller setup")
	}
}
func TestControlInvitationCancelledAndMissingReleaseCannotBegin(t *testing.T) {
	a, c, pads, _ := mappingApp()
	a.screen = ScreenList
	mappingTap(a, c, rawControl((*pads)[0], 304, platform.KeyEnter))
	checkPress(a, c, 304, true)
	e := rawControl((*pads)[0], 304, platform.KeyEnter)
	e.Cancelled = true
	a.Handle(e)
	*c = c.Add(2 * time.Second)
	a.Tick(*c)
	if a.mapping.mode != "invitation" {
		t.Fatal("synthetic release began")
	}
	(*pads)[0].HeldKnown = true
	checkPress(a, c, 305, true)
	*c = c.Add(100 * time.Millisecond)
	a.Tick(*c)
	*c = c.Add(200 * time.Millisecond)
	a.Tick(*c)
	*c = c.Add(2 * time.Second)
	a.Tick(*c)
	if a.mapping.mode != "invitation" {
		t.Fatal("missing release began")
	}
}

func TestControlInvitationEscapeWorksWithStuckOpeningInput(t *testing.T) {
	a, c, pads, _ := mappingApp()
	a.screen = ScreenList
	e := rawControl((*pads)[0], 304, platform.KeyEnter)
	e.Pressed = true
	a.Handle(e)
	mappingTap(a, c, rawControl((*pads)[1], 1, platform.KeyBack))
	if a.screen == ScreenControls {
		t.Fatal("stuck opening input blocked keyboard escape")
	}
}

func TestControlInvitationLostReleaseAfterPartialChordRecovers(t *testing.T) {
	a, c, pads, _ := mappingApp()
	a.screen = ScreenList
	mappingTap(a, c, rawControl((*pads)[0], 304, platform.KeyEnter))
	checkPress(a, c, 304, true)
	checkPress(a, c, 305, true)
	checkPress(a, c, 305, false)
	(*pads)[0].HeldKnown = true
	*c = c.Add(100 * time.Millisecond)
	a.Tick(*c)
	*c = c.Add(200 * time.Millisecond)
	a.Tick(*c)
	(*pads)[0].HeldKnown = false
	checkHold(a, c, 304)
	if a.mapping.mode != "basic" {
		t.Fatal("interrupted chord prevented retry")
	}
}
func TestControlKeyboardEnterChordDoesNotAccept(t *testing.T) {
	a, c, pads, saves := mappingApp()
	a.screen = ScreenList
	mappingTap(a, c, rawControl((*pads)[1], 30, platform.KeyOther))
	checkPress(a, c, 28, true)
	checkPress(a, c, 30, true)
	*c = c.Add(time.Second)
	a.Tick(*c)
	if a.screen == ScreenControls || *saves != 0 {
		t.Fatal("Enter chord accepted keyboard")
	}
}

func TestControlInvitationDeferralBarrierOnlyBlocksTarget(t *testing.T) {
	a, c, pads, _ := mappingApp()
	a.screen = ScreenList
	mappingTap(a, c, rawControl((*pads)[0], 304, platform.KeyEnter))
	checkPress(a, c, 304, true)
	checkPress(a, c, 305, true)
	*c = c.Add(time.Second)
	a.Tick(*c)
	mappingTap(a, c, rawControl((*pads)[1], 30, platform.KeyOther))
	if a.screen != ScreenControls || a.mapping.mode != "initial-role" {
		t.Fatal("defer blocked unrelated controller")
	}
}

// Paint after each tick, like the host's vsync loop. A forced harness screenshot
// alone would hide a missing dirty flag by painting just once at the end.
func TestControlInvitationHoldRepaintsBetweenInputEvents(t *testing.T) {
	for _, tc := range []struct {
		name            string
		keyboard, chord bool
	}{
		{"begin", false, false}, {"defer", false, true}, {"keyboard choice", true, false}, {"keyboard defer", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, c, pads, _ := mappingApp()
			a.screen = ScreenList
			index := 0
			code := uint16(304)
			if tc.keyboard {
				index = 1
				code = 30
			}
			mappingTap(a, c, rawControl((*pads)[index], code, platform.KeyOther))
			checkPress(a, c, code, true)
			if tc.chord {
				checkPress(a, c, code+1, true)
			}
			frame, _ := a.Paint()
			before := append([]byte(nil), frame.Pix...)
			for i := 0; i < 3; i++ {
				*c = c.Add(200 * time.Millisecond)
				if !a.ControlsAnimating() {
					t.Fatal("hold is not in animation loop")
				}
				a.Tick(*c)
				frame, dirty := a.Paint()
				if len(dirty) == 0 || bytes.Equal(before, frame.Pix) {
					t.Fatal("hold text did not advance between input events", i)
				}
				before = append(before[:0], frame.Pix...)
			}
			if tc.chord {
				checkPress(a, c, code+1, false)
			} else {
				checkPress(a, c, code, false)
			}
			frame, dirty := a.Paint()
			if len(dirty) == 0 || bytes.Equal(before, frame.Pix) || a.ControlsAnimating() {
				t.Fatal("released hold retained its progress")
			}
		})
	}
}
