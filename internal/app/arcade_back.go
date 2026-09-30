package app

import (
	"image"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

// The free build's way back to MisterZine Arcade (Config.ArcadeBack). On a
// card that had the members' beta and now runs the free MisterZine, a page
// at start asks once whether to go back: MiSTer Companion's Install Center
// rewrites MisterZine's Downloader entry, and the next Update All then puts
// the free version in the beta's place without the member choosing it.
// Holding A runs the member's own installer (updater.ModeBeta) on the update
// screen, which offers the restart into the beta once it has finished; B
// keeps the free version. Either answer is the last one: the host records
// it once ArcadeBackPending turns false.

// arcadeBackHold is how long A is held to go back, as long as the arcade
// intro's: the page comes up at start, where a stray press is likely.
const arcadeBackHold = arcadeIntroHold

// ArcadeBackPending reports the page still waiting for its answer.
func (a *App) ArcadeBackPending() bool { return a.cfg.ArcadeBack }

// handleArcadeBack takes every key while the page is up: A held goes back,
// B stays on the free version, and nothing else reaches the list.
func (a *App) handleArcadeBack(ev platform.Event) bool {
	at := ev.At
	if at.IsZero() {
		at = a.cfg.TimerNow()
	}
	a.saver.lastInput = at
	if a.bounced(ev) {
		return false
	}
	switch ev.Key {
	case platform.KeyEnter:
		if ev.Pressed {
			if a.down[ev.Key] {
				return false
			}
			a.down[ev.Key] = true
			a.arcadeBackAt, a.arcadeBackBar = at, 0
			return false
		}
		if !a.down[ev.Key] {
			return false
		}
		// a release at the deadline completes the hold even when it arrives
		// before the tick; a shorter hold starts again
		a.tickArcadeBack(at)
		delete(a.down, ev.Key)
		a.arcadeBackAt, a.arcadeBackBar = time.Time{}, 0
		a.all = true
		return true
	case platform.KeyBack:
		if ev.Pressed {
			if a.down[ev.Key] {
				return false
			}
			a.down[ev.Key] = true
			a.answerArcadeBack()
			return true
		}
		delete(a.down, ev.Key)
	}
	return false
}

// tickArcadeBack moves the hold's progress line, and goes back once A has
// been held long enough.
func (a *App) tickArcadeBack(now time.Time) bool {
	if !a.cfg.ArcadeBack || a.arcadeBackAt.IsZero() {
		return false
	}
	if now.Sub(a.arcadeBackAt) >= arcadeBackHold {
		a.answerArcadeBack()
		a.OpenUpdate(updater.ModeBeta)
		// the held A stays down until its release, so it cannot act on the
		// update screen
		a.down[platform.KeyEnter] = true
		return true
	}
	if bar := a.holdBarWidth(a.arcadeBackAt, arcadeBackHold, now); bar != a.arcadeBackBar {
		a.arcadeBackBar = bar
		a.all = true
		return true
	}
	return false
}

// answerArcadeBack takes the page down for good.
func (a *App) answerArcadeBack() {
	a.cfg.ArcadeBack = false
	a.arcadeBackAt, a.arcadeBackBar = time.Time{}, 0
	a.settingsChanged()
	a.all = true
}

func (a *App) nextArcadeBackTick() time.Time {
	if !a.cfg.ArcadeBack || a.arcadeBackAt.IsZero() {
		return time.Time{}
	}
	return a.nextHoldPixel(a.arcadeBackAt, arcadeBackHold, a.arcadeBackBar)
}

// arcadeBackGap is the space between the page's paragraphs.
const arcadeBackGap = 5

// paintArcadeBack draws the page over the list's body, the way the arcade
// intro does, with its legend and the hold's progress line: the question
// in the accent, then its paragraphs.
func (a *App) paintArcadeBack(c *gfx.Canvas) {
	c.Fill(a.lay.Body, pal.Bg)
	box := a.lay.Body.Inset(4)
	c.Box(box, pal.Line)
	x, y := box.Min.X+5, box.Min.Y+5
	for i, p := range a.arcadeBackLines(box) {
		col := pal.Fg
		if i == 0 {
			col = pal.Accent
		}
		for _, line := range p {
			c.Text(x, y, a.sm, line, col)
			y += a.sm.H + 2
		}
		y += arcadeBackGap
	}
	a.paintHint(c, a.arcadeBackHint())
	a.paintHoldBar(c)
}

// arcadeBackLines wraps the question and the paragraphs, naming the pad's
// own buttons, to the page's box.
func (a *App) arcadeBackLines(box image.Rectangle) [][]string {
	cols := a.sm.Cols(box.Dx() - 10)
	var out [][]string
	for _, p := range []string{
		"Back to MisterZine Arcade?",
		"This card had MisterZine Arcade, the members' version. It has the free one now.",
		"Not your choice? MiSTer Companion's Install Center can do that. Hold " + a.btn("A") + " to go back; favorites and settings stay.",
		a.btn("B") + " keeps the free version. This won't ask again.",
	} {
		out = append(out, gfx.Wrap(p, cols, 8))
	}
	return out
}

// arcadeBackHeight is how tall the page's words stand in box, from its top.
func (a *App) arcadeBackHeight(box image.Rectangle) int {
	h := 5
	for _, p := range a.arcadeBackLines(box) {
		h += len(p)*(a.sm.H+2) + arcadeBackGap
	}
	return h
}

// arcadeBackHint is the page's legend, shortened where the bar is narrow.
func (a *App) arcadeBackHint() string {
	hint := "Hold A Go back  B Stay free"
	if !a.hintFits(hint) {
		hint = "Hold A Back  B Free"
	}
	return hint
}
