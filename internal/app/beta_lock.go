package app

import (
	"errors"
	"github.com/matijaerceg/misterzine-on-device/internal/access"
	"image"
	"strings"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/beta"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
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
// The screen offers no way to Stable itself, so a member cannot
// leave by accident. On a card unlocked for an earlier batch
// (Config.BetaEarlierBatch), where a member whose membership has lapsed
// stands, one line names the way: MisterZine-Switch-To-Stable in Scripts. A
// first install never sees it.

// codeLen is the number of digits in a code.
const codeLen = 6

// betaLock is the lock screen's state; App.lock is nil once unlocked.
type betaLock struct {
	earlyAccess   bool
	featureTitle  string
	requiredMonth access.Month
	optional      bool
	digits        [codeLen]int8 // -1 while a box is empty
	box           int           // the chosen box
	message       string        // the answer to the last Unlock, "" before one
	leaving       bool          // unlocked for an earlier batch: name the way to Stable
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
	case platform.KeyBack:
		if l.optional {
			a.unlock("")
			return true
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
	if l.optional {
		if a.cfg.UnlockCode == nil {
			l.message = "Codes are available in the official app."
			return
		}
		month, err := a.cfg.UnlockCode(code)
		if errors.Is(err, access.ErrCode) {
			l.message = lockWrong
			return
		}
		if !month.Valid() {
			l.message = lockBadBuild
			return
		}
		a.cfg.AccessMonth = max(a.cfg.AccessMonth, month)
		a.accessChanged()
		if a.cfg.AccessMonth < l.requiredMonth {
			l.clear()
			l.message = "Covers " + a.cfg.AccessMonth.String() + ". Needs " + l.requiredMonth.String() + " or newer."
			return
		}
		msg := "Unlocked through " + a.cfg.AccessMonth.String()
		if err != nil {
			msg = lockUnsaved
		}
		a.unlock(msg)
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

// lockHint is the lock screen's legend, shortened where the bar is narrow.
func (a *App) lockHint() string {
	back := "B Quit"
	if a.lock.optional {
		back = "B Back"
	}
	if !a.lock.optional && (a.lock.started() || a.lock.message != "") {
		back = "B Clear"
	}
	digits := gfx.ArrowUp + gfx.ArrowDown + " Digit  " + gfx.ArrowLeft + gfx.ArrowRight + " Move  "
	hint := digits + "A Unlock  " + back
	if !a.hintFits(hint) {
		hint = digits + "A OK  " + back
	}
	return hint
}

// The code boxes: the body font's digits drawn at twice their size, in
// two groups of three.
const (
	lockBoxW, lockBoxH = 18, 24
	lockBoxGap         = 4
	lockGroupGap       = 10
	lockLeaveGap       = 4 // above the lines that name the way to Stable
)

// lockFooter is the foot of the lock screen's body: where the code comes
// from.
func (a *App) lockFooter() []string {
	return []string{"patreon.com/MisterZine"}
}

// lockLeave is the line that names the way back to Stable, from
// the longest to the shortest: where even the script's full name is wider
// than the body (tate at the widest inset), Scripts lists it as
// MisterZine-Switch-To-Stable, so the end of its name finds it.
var lockLeave = []string{
	"Leaving Beta? Run MisterZine-Switch-To-Stable from Scripts.",
	"Leaving? Run MisterZine-Switch-To-Stable in Scripts.",
	"Leaving? Run Switch-To-Stable in Scripts.",
}

// lockLeaveLines name the way back to Stable under the footer,
// wrapped to cols characters, on a card unlocked for an earlier batch; nil
// on any other. It is text, not a control: the Scripts entry does it. The
// script's full name comes first, in two lines if it can and three if it
// must; only where it is wider than the body does the short one stand in.
func (a *App) lockLeaveLines(cols int) []string {
	if !a.lock.leaving {
		return nil
	}
	for _, try := range []struct {
		s     string
		lines int
	}{{lockLeave[0], 2}, {lockLeave[1], 2}, {lockLeave[1], 3}, {lockLeave[2], 3}} {
		if len(gfx.Wrap(try.s, cols, len(try.s)+1)) <= try.lines && wordsWhole(try.s, cols) {
			return evenWrap(try.s, cols, try.lines)
		}
	}
	return evenWrap(lockLeave[len(lockLeave)-1], cols, 3)
}

// wordsWhole reports every word of s no wider than cols, so wrapping never
// cuts one.
func wordsWhole(s string, cols int) bool {
	for _, w := range strings.Fields(s) {
		if len(w) > cols {
			return false
		}
	}
	return true
}

// lockBlock lays out the lock screen's body in box: the intro's lines, the
// lines naming the way out, the footer's height with them, and the height
// of the whole block. The block holds the intro, the boxes, two lines kept
// for the answer so nothing moves when one appears, and the footer.
func (a *App) lockBlock(box image.Rectangle) (intro, leave []string, footH, h int) {
	intro = evenWrap("Enter the code from the Patreon post.", a.body.Cols(box.Dx()), 3)
	leave = a.lockLeaveLines(a.sm.Cols(box.Dx()))
	footH = len(a.lockFooter()) * (a.sm.H + 1)
	if len(leave) > 0 {
		footH += lockLeaveGap + len(leave)*(a.sm.H+1)
	}
	h = len(intro)*(a.body.H+1) + 10 + lockBoxH + 8 + 2*(a.sm.H+1) + 6 + footH
	return intro, leave, footH, h
}

// paintLock draws the lock screen: the title and the BETA mark in the
// status bar, what to do, the six boxes and the answer to the last try.
func (a *App) paintLock(c *gfx.Canvas) {
	if a.lock.optional {
		a.paintAccessCode(c)
		return
	}
	l := &a.lay
	c.Fill(l.Status, pal.Surface)
	c.HLine(l.Status.Min.X, l.Status.Max.X-1, l.Status.Max.Y-1, pal.Muted)
	right := a.paintBetaMark(c)
	y := l.Status.Min.Y + 2
	if a.notice != "" {
		c.Text(l.Status.Min.X+2, y, a.sm, gfx.Fit(a.notice, a.sm.Cols(right-l.Status.Min.X-4)), pal.Fg)
	} else {
		c.Text(l.Status.Min.X+2, y, a.sm, gfx.Fit(a.codeScreenTitle(), a.sm.Cols(right-l.Status.Min.X-4)), pal.Accent)
	}

	c.Box(l.Body, pal.Line)
	box := l.Body.Inset(4)
	msgCols := a.sm.Cols(box.Dx())
	foot := a.lockFooter()
	intro, leave, footH, h := a.lockBlock(box)
	// the block, centred on the body
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
	if len(leave) > 0 {
		y += lockLeaveGap
		for _, line := range leave {
			c.Text(box.Min.X+(box.Dx()-a.sm.Width(line))/2, y, a.sm, line, pal.Muted)
			y += a.sm.H + 1
		}
	}
	a.paintHint(c, a.lockHint())
}

// evenWrap wraps centred text into as few lines as gfx.Wrap does, but as
// narrow as they stay that few, so a last line is not left with one word;
// never so narrow that a word would be cut.
func evenWrap(s string, cols, maxLines int) []string {
	lines := gfx.Wrap(s, cols, maxLines)
	n := len(gfx.Wrap(s, cols, len(s)+1))
	if n < 2 || n > maxLines {
		return lines
	}
	for c := cols - 1; c > 0 && wordsWhole(s, c) && len(gfx.Wrap(s, c, len(s)+1)) == n; c-- {
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

func (a *App) codeScreenTitle() string {
	if a.lock.optional {
		if a.lock.featureTitle != "" {
			return a.lock.featureTitle
		}
		return "MisterZine code"
	}
	return "MisterZine Arcade"
}

// paintAccessCode explains permanent access without blocking ordinary app use.
func (a *App) paintAccessCode(c *gfx.Canvas) {
	l := &a.lay
	c.Fill(l.Status, pal.Surface)
	c.HLine(l.Status.Min.X, l.Status.Max.X-1, l.Status.Max.Y-1, pal.Muted)
	paintFeatureText(c, l.Status.Min.X+2, l.Status.Min.Y+2, a.sm,
		gfx.Fit(a.codeScreenTitle(), a.sm.Cols(l.Status.Dx()-4)), pal.Accent)
	c.Box(l.Body, pal.Line)
	box := l.Body.Inset(4)
	cols := a.sm.Cols(box.Dx())
	paragraphs := []string{
		"Patreon support funds development.",
		"Codes cover features introduced through their month, not an expiry date.",
		"No need to stay subscribed.",
	}
	if a.lock.earlyAccess {
		paragraphs = []string{
			"This feature will be free for everyone when it leaves beta.",
			"Your code keeps this early access unlocked.",
			"No need to stay subscribed.",
		}
	}
	if a.lock.requiredMonth != 0 && !a.lock.earlyAccess {
		paragraphs[0] = "A Supporter extra, including after beta."
		paragraphs[1] = "Your code unlocks it forever."
	}
	var lines []string
	for _, p := range paragraphs {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, evenWrap(p, cols, 100)...)
	}
	required := "Enter your Patreon code:"
	if a.lock.requiredMonth != 0 {
		required = "Requires " + a.lock.requiredMonth.Short() + " or newer:"
	}
	introCount := len(lines)
	lines = append(lines, "")
	lines = append(lines, gfx.Wrap("Your access: "+a.cfg.AccessMonth.Short(), cols, 100)...)
	lines = append(lines, gfx.Wrap(required, cols, 100)...)
	lineH := a.sm.H + 1
	future := evenWrap("Future features may need a newer code.", cols, 3)
	footH := len(future)*lineH + 4 + lineH
	if a.lock.earlyAccess {
		future = nil
		footH = lineH
	}
	h := 4 + lockBoxH + 4 + 2*lineH + 4 + footH
	for _, line := range lines {
		if line == "" {
			h += 6
		} else {
			h += lineH
		}
	}
	y := box.Min.Y + max(0, (box.Dy()-h)/2)
	introH := 0
	for _, line := range lines[:introCount] {
		if line == "" {
			introH += 6
		} else {
			introH += lineH
		}
	}
	requiredY := y + introH + 6
	y = box.Min.Y + max(0, (requiredY-box.Min.Y-introH)/2)
	for i, line := range lines {
		if i == introCount {
			y = requiredY
			continue
		}
		if line == "" {
			y += 6
			continue
		}
		c.Text(box.Min.X+(box.Dx()-a.sm.Width(line))/2, y, a.sm, line, pal.Fg)
		y += lineH
	}
	y += 4
	a.paintCodeBoxes(c, box, y)
	y += lockBoxH + 4
	footerY := min(box.Max.Y-footH, max(y+2*lineH+4, (y-4+box.Max.Y-footH)/2))
	for _, line := range evenWrap(a.lock.message, cols, 2) {
		c.Text(box.Min.X+(box.Dx()-a.sm.Width(line))/2, y, a.sm, line, pal.Warn)
		y += lineH
	}
	y = footerY
	for _, line := range future {
		c.Text(box.Min.X+(box.Dx()-a.sm.Width(line))/2, y, a.sm, line, pal.Fg)
		y += lineH
	}
	url := "patreon.com/MisterZine"
	c.Text(box.Min.X+(box.Dx()-a.sm.Width(url))/2, footerY+footH-lineH, a.sm, url, pal.Accent)
	a.paintHint(c, a.lockHint())
}
