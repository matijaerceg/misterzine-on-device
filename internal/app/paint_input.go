package app

import (
	"image"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// inputRec is one received event for the Input test screen.
type inputRec struct {
	ev    platform.Event
	delta time.Duration // since the previous event
	dup   bool          // ignored as a duplicate press/release
}

const inputKeep = 14

// recordInput keeps the last events for the Input test screen.
func (a *App) recordInput(ev platform.Event) {
	if a.screen != ScreenInput {
		return
	}
	dup := (ev.Pressed && a.down[ev.Key]) || (!ev.Pressed && !a.down[ev.Key])
	var delta time.Duration
	if n := len(a.inputLog); n > 0 {
		delta = ev.At.Sub(a.inputLog[n-1].ev.At)
	}
	a.inputLog = append(a.inputLog, inputRec{ev: ev, delta: delta, dup: dup})
	if len(a.inputLog) > inputKeep {
		a.inputLog = a.inputLog[len(a.inputLog)-inputKeep:]
	}
	a.all = true
}

func (a *App) paintInput(c *gfx.Canvas) {
	l := &a.lay
	a.paintStatus(c)
	body := l.Body
	c.Fill(body, gen.Eva.Bg)
	y := body.Min.Y
	c.Text(body.Min.X, y, a.body, "Input test", gen.Eva.Accent)
	y += l.Line
	c.Text(body.Min.X, y, a.sm, "press anything; hold to see repeats; B twice quickly to leave", gen.Eva.Muted)
	y += a.sm.H + 4
	cols := a.sm.Cols(body.Dx())
	for _, r := range a.inputLog {
		state := "down"
		col := gen.Eva.Fg
		if !r.ev.Pressed {
			state = "up  "
			col = gen.Eva.Muted
		}
		if r.dup {
			state = "dup "
			col = gen.Eva.Warn
		}
		ms := r.delta.Milliseconds()
		line := gfx.Fit(padLeft(itoa(int(ms)), 5)+"ms  "+padRight(r.ev.Key.String(), 10)+" "+state+"  "+r.ev.Source, cols)
		c.Text(body.Min.X, y, a.sm, line, col)
		y += a.sm.H + 1
		if y+a.sm.H > body.Max.Y {
			break
		}
	}
	held := ""
	for k := range a.down {
		held += k.String() + " "
	}
	c.Text(body.Min.X, body.Max.Y-a.sm.H-1, a.sm, gfx.Fit("held now: "+held, cols), gen.Eva.Accent)
	a.paintHint(c, "B B leave (two presses within a second)")
	_ = image.Point{}
}

func padLeft(s string, n int) string {
	for len(s) < n {
		s = " " + s
	}
	return s
}

func padRight(s string, n int) string {
	for len(s) < n {
		s += " "
	}
	return s
}

func (a *App) actInput(k platform.Key) bool {
	if k == platform.KeyBack {
		now := a.cfg.Now()
		if now.Sub(a.inputBack) < time.Second {
			a.inputLog = nil
			a.openPanel(ScreenSettings)
			return true
		}
		a.inputBack = now
	}
	a.all = true
	return true
}
