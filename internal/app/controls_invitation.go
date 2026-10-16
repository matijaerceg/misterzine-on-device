package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"time"
)

func (a *App) controlKeyboardAvailable(separate bool) bool {
	for _, p := range a.controlInput.devices {
		if p.Keyboard && (!separate || p.Node != a.mapping.device.Node || controlID(p) != controlID(a.mapping.device)) {
			return true
		}
	}
	return false
}

// Invitation holds retain source identity, so two devices with the same button
// code cannot accidentally release each other's input. Only the target can begin;
// two buttons on any readable devices can dismiss.
func (a *App) handleControlInvitation(raw platform.Event, fresh bool) bool {
	v, d := &a.mapping, &a.mapping.check
	a.all = true
	token := controlToken(raw)
	if raw.Cancelled {
		if _, held := d.introHeld[token]; !held {
			return true
		}
		delete(d.introHeld, token)
		d.introCancelled = true
		d.introAt = time.Time{}
		return true
	}
	if !raw.At.IsZero() && raw.At.Before(d.opened) {
		return true
	}
	known := false
	for _, p := range a.controlInput.devices {
		if controlID(p) == raw.DeviceID && p.Node == raw.Node {
			known = true
			break
		}
	}
	if !known || raw.Code == 0 {
		return true
	}
	own := raw.DeviceID == controlID(v.device) && raw.Node == v.device.Node
	if fresh && raw.Keyboard && raw.Code == 1 {
		a.exitControlCheck()
		return true
	}
	if raw.Pressed {
		if !fresh {
			return true
		}
		if len(d.introHeld) == 0 {
			d.introAt = a.controlNow()
			d.introChord = false
			d.introCancelled = false
			d.introBlocked = len(d.ignored) > 0 || len(d.entryHeld) > 0
		}
		before := a.invitationControlCount()
		d.introHeld[token] = raw
		count := a.invitationControlCount()
		if count == 2 && before != count {
			d.introAt = a.controlNow()
			d.introChord = true
		}
		if v.mode == "initial-role" && own && !d.introBlocked && count == 1 {
			switch raw.Code {
			case 103:
				v.cursor = 0
			case 108:
				v.cursor = 1
			}
		}
	} else {
		press, held := d.introHeld[token]
		delete(d.introHeld, token)
		if !held {
			return true
		}
		if len(d.introHeld) > 0 {
			for _, other := range d.introHeld {
				if a.controlGestureToken(other) != a.controlGestureToken(raw) {
					d.introCancelled = true
				}
			}
			return true
		}
		if v.mode == "initial-role" && own && !d.introBlocked && !d.introChord && !d.introCancelled {
			if press.Code == 28 {
				a.actControlPopup(platform.KeyEnter)
			} else if press.Code != 103 && press.Code != 108 {
				v.cursor = 1 - v.cursor
			}
		}
		d.introAt = time.Time{}
	}
	a.all = true
	return true
}

func (a *App) tickControlInvitation(now time.Time) {
	d := &a.mapping.check
	if len(d.introHeld) == 0 {
		return
	}
	missing := []string{}
	for token, ev := range d.introHeld {
		found := false
		for _, p := range a.controlInput.devices {
			if controlID(p) != ev.DeviceID || p.Node != ev.Node {
				continue
			}
			found = !p.HeldKnown || ev.Code >= 0x300
			for _, code := range p.Held {
				found = found || code == ev.Code
			}
		}
		if !found {
			missing = append(missing, token)
		}
	}
	if len(missing) > 0 {
		if d.introMissing.IsZero() {
			d.introMissing = now
		}
		if now.Sub(d.introMissing) >= 150*time.Millisecond {
			d.introCancelled = true
			d.introAt = time.Time{}
			for _, token := range missing {
				delete(d.introHeld, token)
				delete(a.controlInput.held, token)
				delete(a.controlInput.keys, token)
			}
		}
		return
	}
	d.introMissing = time.Time{}
	if d.introAt.IsZero() || d.introCancelled {
		return
	}
	if now.Sub(d.introAt) < controlHoldDuration {
		return
	}
	if d.introChord {
		if a.invitationControlCount() >= 2 {
			a.exitControlCheck()
		}
		return
	}
	if a.invitationControlCount() != 1 || d.introBlocked {
		return
	}
	for _, ev := range d.introHeld {
		if ev.DeviceID != controlID(a.mapping.device) || ev.Node != a.mapping.device.Node {
			return
		}
	}
	if a.mapping.mode == "initial-role" {
		a.actControlPopup(platform.KeyEnter)
		if a.screen == ScreenControls && a.mapping.mode == "initial-role" {
			d.introCancelled = true
			d.introAt = time.Time{}
		}
	} else {
		a.startBasicSetup()
	}
}

func (a *App) controlInvitationProgress() (time.Time, time.Time) {
	d := &a.mapping.check
	if d.introCancelled {
		return time.Time{}, time.Time{}
	}
	if d.introChord {
		return time.Time{}, d.introAt
	}
	if d.introBlocked {
		return time.Time{}, time.Time{}
	}
	for _, ev := range d.introHeld {
		if ev.DeviceID == controlID(a.mapping.device) && ev.Node == a.mapping.device.Node {
			return d.introAt, time.Time{}
		}
	}
	return time.Time{}, time.Time{}
}
