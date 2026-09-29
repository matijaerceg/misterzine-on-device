package app

import (
	"errors"
	"image"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/beta"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

// The Patreon beta's lock screen. A locked beta build (Config.BetaUnlock
// set) opens here instead of the list, and nothing else is reachable: no
// list, Options, search or launch. The member enters the six-digit code
// from the Patreon post and the list follows at once, with no restart.
//
// On a pad, Up and Down turn the chosen digit, Left and Right move between
// the boxes, and A or Start unlocks; on a keyboard the digit keys type,
// Backspace erases and Enter unlocks. B clears a started entry, and on an
// empty one returns to the MiSTer menu, as the Menu button always does.
//
// On a card that can go back to the free MisterZine by itself
// (Config.CanSwitchToFree), X pressed twice switches to it: the run's
// update screen shows in place of the lock screen, and when it has
// finished A restarts into the free version and B comes back here.

// codeLen is the number of digits in a code.
const codeLen = 6

// betaLock is the lock screen's state; App.lock is nil once unlocked.
type betaLock struct {
	digits    [codeLen]int8 // -1 while a box is empty
	box       int           // the chosen box
	message   string        // the answer to the last Unlock, "" before one
	free      bool          // the card can switch back to the free version
	switching bool          // the switch's update screen shows instead
}

func newBetaLock() *betaLock {
	l := &betaLock{}
	l.clear()
	return l
}

// clear empties the boxes and forgets the last answer.
func (l *betaLock) clear() {
	for i := range l.digits {
		l.digits[i] = -1
	}
	l.box, l.message = 0, ""
}

// Locked reports the lock screen up: a beta build still waiting for its
// code. The host saves nothing while it is.
func (a *App) Locked() bool { return a.lock != nil }

// started reports any box filled.
func (l *betaLock) started() bool {
	for _, d := range l.digits {
		if d >= 0 {
			return true
		}
	}
	return false
}

// code is the entry as typed, and false while a box is still empty.
func (l *betaLock) code() (string, bool) {
	b := make([]byte, codeLen)
	for i, d := range l.digits {
		if d < 0 {
			return "", false
		}
		b[i] = '0' + byte(d)
	}
	return string(b), true
}

// handleLock takes every key while the app is locked: Handle sends it
// nothing else, so no other screen can be reached.
func (a *App) handleLock(ev platform.Event) bool {
	if a.bounced(ev) {
		return false
	}
	if ev.Pressed && ev.Text >= '0' && ev.Text <= '9' {
		// the keyboard's own repeat types again, as in a search
		return a.lockType(int8(ev.Text - '0'))
	}
	if ev.Key == platform.KeyNone || ev.Key == platform.KeyOther {
		return false
	}
	if !ev.Pressed {
		if a.down[ev.Key] {
			delete(a.down, ev.Key)
			a.rep.release(ev.Key)
		}
		return false
	}
	if a.down[ev.Key] {
		return false
	}
	a.down[ev.Key] = true
	a.rep.press(ev.Key, ev.At)
	if a.lockRepeats(ev.Key) {
		a.rep.next = ev.At.Add(time.Duration(a.HoldDelay()) * time.Millisecond)
	}
	return a.actLock(ev.Key)
}

// lockRepeats reports the keys that repeat while held on the lock screen:
// turning a digit, moving between boxes and erasing.
func (a *App) lockRepeats(k platform.Key) bool {
	switch k {
	case platform.KeyUp, platform.KeyDown, platform.KeyLeft, platform.KeyRight, platform.KeyBackspace:
		return true
	}
	return false
}

// lockType puts a typed digit in the chosen box and moves to the next.
func (a *App) lockType(d int8) bool {
	l := a.lock
	l.digits[l.box] = d
	l.box = min(codeLen-1, l.box+1)
	l.message = ""
	a.all = true
	return true
}

// actLock carries out one key on the lock screen.
func (a *App) actLock(k platform.Key) bool {
	l := a.lock
	switch k {
	case platform.KeyUp, platform.KeyDown:
		// an empty box starts at 0 going up and at 9 going down
		d := l.digits[l.box]
		switch {
		case d < 0 && k == platform.KeyUp:
			d = 0
		case d < 0:
			d = 9
		case k == platform.KeyUp:
			d = (d + 1) % 10
		default:
			d = (d + 9) % 10
		}
		l.digits[l.box] = d
		l.message = ""
	case platform.KeyLeft:
		l.box = max(0, l.box-1)
	case platform.KeyRight:
		l.box = min(codeLen-1, l.box+1)
	case platform.KeyBackspace:
		// erase the chosen box, or the one before it when that is empty
		if l.digits[l.box] < 0 {
			l.box = max(0, l.box-1)
		}
		l.digits[l.box] = -1
		l.message = ""
	case platform.KeyEnter, platform.KeyStart:
		a.tryUnlock()
	case platform.KeyTab:
		// X asks first, then switches to the free version
		if !l.free {
			return false
		}
		if l.message != a.lockSwitchAsk() {
			l.message = a.lockSwitchAsk()
			break
		}
		a.switchToFree()
		return true
	case platform.KeyBack:
		if l.message == a.lockSwitchAsk() {
			l.message = "" // no switch, and the entry stays
			break
		}
		if !l.started() && l.message == "" {
			a.quit()
			return false
		}
		l.clear()
	case platform.KeyMenu:
		a.quit()
		return false
	default:
		return false
	}
	a.all = true
	return true
}

// quit leaves for the MiSTer menu, the way Options -> Quit MisterZine does.
func (a *App) quit() {
	if a.cfg.Quit != nil {
		a.cfg.Quit()
	}
}

// Lock screen answers. A right code whose unlock the card could not save
// still opens the app for this session: the member has the code.
const (
	lockIncomplete   = "Enter all six digits."
	lockWrong        = "Wrong code. Check the Patreon post."
	lockBadBuild     = "This build cannot be unlocked. Install it again."
	lockUnlocked     = "Unlocked. Thank you!"
	lockUnsaved      = "Unlocked, but not saved on the card"
	lockNoticeLength = 6 * time.Second
)

// tryUnlock hands a complete entry to the host, which checks it and saves
// the batch's receipt; an incomplete one moves to its first empty box.
func (a *App) tryUnlock() {
	l := a.lock
	code, ok := l.code()
	if !ok {
		for i, d := range l.digits {
			if d < 0 {
				l.box = i
				break
			}
		}
		l.message = lockIncomplete
		return
	}
	var err error
	if a.cfg.BetaUnlock != nil {
		err = a.cfg.BetaUnlock(code)
	}
	switch {
	case err == nil:
		a.unlock(lockUnlocked)
	case errors.Is(err, beta.ErrLocked):
		l.message = lockWrong // the entry stays, to correct a digit
	case errors.Is(err, beta.ErrBuild):
		l.message = lockBadBuild
	default:
		a.unlock(lockUnsaved)
	}
}

// unlock takes the lock screen down: the list the app opened on shows,
// with the notice in its status bar.
func (a *App) unlock(notice string) {
	a.lock = nil
	a.rep = repeater{}
	a.saver.lastInput = a.cfg.TimerNow()
	a.Notice(notice, lockNoticeLength)
	a.all = true
}

// lockSwitchAsk is the question the first X puts: a second X switches.
func (a *App) lockSwitchAsk() string {
	return "Press " + a.btn("X") + " again to switch to the free version."
}

// switchToFree starts the run that takes the card back to the free
// MisterZine (updater.ModeFree) and shows its update screen, still locked:
// the host saves nothing, and the screen's keys are handleLockSwitch's.
func (a *App) switchToFree() {
	a.lock.switching = true
	a.lock.message = ""
	a.OpenUpdate(updater.ModeFree)
}

// handleLockSwitch takes every key while the lock screen shows its switch
// to the free version. The update screen's own keys work (hold B cancels
// the run, the arrows scroll its log, A restarts into the free version
// once it is installed), but nothing leads past the lock: once the run has
// finished, B comes back to the lock screen and Menu returns to the MiSTer
// menu, where the update screen would open Options.
func (a *App) handleLockSwitch(ev platform.Event) bool {
	if a.bounced(ev) {
		return false
	}
	if ev.Pressed && !a.down[ev.Key] && !a.update.Active() {
		switch ev.Key {
		case platform.KeyBack:
			a.down[ev.Key] = true
			if a.update.ResultNotice() && a.cfg.Action != nil {
				a.cfg.Action("update-dismiss", a.update.ID)
			}
			a.lock.switching = false
			a.screen = ScreenList // what the unlock opens onto
			a.rep = repeater{}
			a.all = true
			return true
		case platform.KeyMenu:
			a.down[ev.Key] = true
			a.quit()
			return false
		}
	}
	return a.handleUpdate(ev)
}

// lockHint is the lock screen's legend, shortened where the bar is narrow.
// X for the free version comes last, and goes first when space runs out:
// the line under the code says it too.
func (a *App) lockHint() string {
	back := "B Quit"
	if a.lock.started() || a.lock.message != "" {
		back = "B Clear"
	}
	digits := gfx.ArrowUp + gfx.ArrowDown + " Digit  " + gfx.ArrowLeft + gfx.ArrowRight + " Move  "
	free := ""
	if a.lock.free {
		free = "  X Free version"
	}
	hint := digits + "A Unlock  " + back + free
	for _, h := range []string{digits + "A OK  " + back + free, digits + "A OK  " + back} {
		if a.hintFits(hint) {
			break
		}
		hint = h
	}
	return hint
}

// The code boxes: the body font's digits drawn at twice their size, in
// two groups of three.
const (
	lockBoxW, lockBoxH = 18, 24
	lockBoxGap         = 4
	lockGroupGap       = 10
	lockSwitchGap      = 4 // above the line that offers the free version
)

// lockFooter is the foot of the lock screen's body: where the code comes
// from.
func (a *App) lockFooter() []string {
	return []string{"patreon.com/MisterZine"}
}

// lockSwitchLine is the line under the footer that offers the free
// version, as a legend chunk, in the shorter form where the longer does
// not fit cols characters; "" on a card that cannot switch.
func (a *App) lockSwitchLine(cols int) string {
	if !a.lock.free {
		return ""
	}
	if s := "X Switch to the free version"; len(s) <= cols {
		return s
	}
	return "X Free version"
}

// paintLock draws the lock screen: the title and the BETA mark in the
// status bar, what to do, the six boxes and the answer to the last try.
func (a *App) paintLock(c *gfx.Canvas) {
	l := &a.lay
	c.Fill(l.Status, pal.Surface)
	c.HLine(l.Status.Min.X, l.Status.Max.X-1, l.Status.Max.Y-1, pal.Muted)
	right := a.paintBetaMark(c)
	y := l.Status.Min.Y + 2
	if a.notice != "" {
		c.Text(l.Status.Min.X+2, y, a.sm, gfx.Fit(a.notice, a.sm.Cols(right-l.Status.Min.X-4)), pal.Fg)
	} else {
		c.Text(l.Status.Min.X+2, y, a.sm, gfx.Fit("MisterZine Arcade", a.sm.Cols(right-l.Status.Min.X-4)), pal.Accent)
	}

	c.Box(l.Body, pal.Line)
	box := l.Body.Inset(4)
	intro := evenWrap("Enter the code from the Patreon post.", a.body.Cols(box.Dx()), 3)
	msgCols := a.sm.Cols(box.Dx())
	foot := a.lockFooter()
	sw := a.lockSwitchLine(msgCols)
	footH := len(foot) * (a.sm.H + 1)
	if sw != "" {
		footH += lockSwitchGap + a.sm.H + 1
	}
	// the block, centred on the body: the intro, the boxes, two lines kept
	// for the answer so nothing moves when one appears, and the footer
	h := len(intro)*(a.body.H+1) + 10 + lockBoxH + 8 + 2*(a.sm.H+1) + 6 + footH
	top := box.Min.Y + max(0, (box.Dy()-h)/2)
	y = top
	for _, line := range intro {
		c.Text(box.Min.X+(box.Dx()-a.body.Width(line))/2, y, a.body, line, pal.Fg)
		y += a.body.H + 1
	}
	y += 10
	a.paintCodeBoxes(c, box, y)
	y += lockBoxH + 8
	for _, line := range evenWrap(a.lock.message, msgCols, 2) {
		c.Text(box.Min.X+(box.Dx()-a.sm.Width(line))/2, y, a.sm, line, pal.Warn)
		y += a.sm.H + 1
	}
	y = top + h - footH
	for _, line := range foot {
		line = gfx.Fit(line, msgCols)
		c.Text(box.Min.X+(box.Dx()-a.sm.Width(line))/2, y, a.sm, line, pal.Muted)
		y += a.sm.H + 1
	}
	if sw != "" {
		// painted as the legend paints it: the button in the accent
		y += lockSwitchGap
		w := a.hintWidth(hintChunks(sw), " ")
		a.hintLine(c, box.Min.X+(box.Dx()-w)/2, y, w, sw)
	}
	a.paintHint(c, a.lockHint())
}

// evenWrap wraps centred text into as few lines as gfx.Wrap does, but as
// narrow as they stay that few, so a last line is not left with one word.
func evenWrap(s string, cols, maxLines int) []string {
	lines := gfx.Wrap(s, cols, maxLines)
	n := len(gfx.Wrap(s, cols, len(s)+1))
	if n < 2 || n > maxLines {
		return lines
	}
	for c := cols - 1; c > 0 && len(gfx.Wrap(s, c, len(s)+1)) == n; c-- {
		lines = gfx.Wrap(s, c, maxLines)
	}
	return lines
}

// paintCodeBoxes draws the six boxes centred in box with their tops at y:
// the chosen one framed in the accent on the surface colour, the digits
// at twice the body font's size.
func (a *App) paintCodeBoxes(c *gfx.Canvas, box image.Rectangle, y int) {
	w := codeLen*lockBoxW + (codeLen-2)*lockBoxGap + lockGroupGap
	x := box.Min.X + (box.Dx()-w)/2
	inkL, inkT, inkW, inkH := digitInk(a.body)
	for i, d := range a.lock.digits {
		r := image.Rect(x, y, x+lockBoxW, y+lockBoxH)
		frame, fill := pal.Line, pal.Bg
		if i == a.lock.box {
			frame, fill = pal.Accent, pal.Surface
		}
		c.Fill(r, fill)
		c.Box(r, frame)
		if d >= 0 {
			gx := r.Min.X + (lockBoxW-2*inkW)/2 - 2*inkL
			gy := r.Min.Y + (lockBoxH-2*inkH)/2 - 2*inkT
			bigGlyph(c, gx, gy, a.body, '0'+byte(d), pal.Fg)
		}
		x += lockBoxW + lockBoxGap
		if i == codeLen/2-1 {
			x += lockGroupGap - lockBoxGap
		}
	}
}

// digitInk is the box the digits' ink shares within a cell of f, so a
// digit sits centred on its ink rather than on its cell.
func digitInk(f *gfx.Font) (left, top, w, h int) {
	minX, minY, maxX, maxY := f.W, f.H, -1, -1
	for ch := byte('0'); ch <= '9'; ch++ {
		for y, row := range f.Glyph(ch) {
			for x := 0; x < f.W; x++ {
				if row&(0x80>>uint(x)) != 0 {
					minX, maxX = min(minX, x), max(maxX, x)
					minY, maxY = min(minY, y), max(maxY, y)
				}
			}
		}
	}
	if maxX < 0 {
		return 0, 0, f.W, f.H
	}
	return minX, minY, maxX - minX + 1, maxY - minY + 1
}

// bigGlyph draws one character of f at twice its size, its cell's top
// left at (x, y).
func bigGlyph(c *gfx.Canvas, x, y int, f *gfx.Font, ch byte, col rgb) {
	for gy, row := range f.Glyph(ch) {
		for gx := 0; gx < f.W; gx++ {
			if row&(0x80>>uint(gx)) != 0 {
				c.Fill(image.Rect(x+2*gx, y+2*gy, x+2*gx+2, y+2*gy+2), col)
			}
		}
	}
}
