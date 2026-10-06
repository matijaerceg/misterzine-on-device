package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"time"
)

type forgetCodeState struct {
	at      time.Time
	bar     int
	message string
}

func (a *App) openForgetCode() {
	if !a.cfg.AccessMonth.Valid() {
		return
	}
	a.forget = &forgetCodeState{}
	a.rep = repeater{}
	a.all = true
}

func (a *App) handleForgetCode(ev platform.Event) bool {
	if !ev.Pressed {
		delete(a.down, ev.Key)
		if ev.Key == platform.KeyEnter {
			at := ev.At
			if at.IsZero() {
				at = a.cfg.TimerNow()
			}
			a.tickForgetCode(at)
			if a.forget == nil {
				return true
			}
			a.forget.at = time.Time{}
			a.forget.bar = 0
			a.all = true
			return true
		}
		return false
	}
	if a.down[ev.Key] {
		return false
	}
	a.down[ev.Key] = true
	switch ev.Key {
	case platform.KeyBack, platform.KeyMenu:
		a.forget = nil
		a.all = true
		return true
	case platform.KeyEnter:
		a.forget.at = ev.At
		if a.forget.at.IsZero() {
			a.forget.at = a.cfg.TimerNow()
		}
		a.forget.message = ""
		a.all = true
		return true
	}
	return false
}

func (a *App) tickForgetCode(now time.Time) bool {
	f := a.forget
	if f == nil || f.at.IsZero() {
		return false
	}
	if now.Sub(f.at) >= 2*time.Second {
		f.at = time.Time{}
		f.bar = 0
		if a.cfg.ForgetCode == nil {
			f.message = "Could not forget code. Try again."
		} else {
			month, err := a.cfg.ForgetCode()
			a.cfg.AccessMonth = month
			a.accessChanged()
			if err != nil || month.Valid() {
				f.message = "Could not remove all unlocks. Try again."
			} else {
				a.forget = nil
				a.panel.cursor = 0
				a.Notice("Code forgotten", 3*time.Second)
			}
		}
		a.all = true
		return true
	}
	bar := a.holdBarWidth(f.at, 2*time.Second, now)
	if bar == f.bar {
		return false
	}
	f.bar = bar
	a.all = true
	return true
}

func (a *App) paintForgetCode(c *gfx.Canvas) {
	l := &a.lay
	c.Fill(l.Status, pal.Surface)
	c.Text(l.Status.Min.X+2, l.Status.Min.Y+2, a.sm, "Forget code?", pal.Accent)
	c.Fill(l.Body, pal.Bg)
	c.Box(l.Body, pal.Line)
	box := l.Body.Inset(5)
	paragraphs := []string{"Remove your saved unlocks from this device?", "Your settings stay saved. Enter your code again to restore access.", "Beta stays as you set it. Turn it off for the default free experience."}
	var lines []string
	for _, p := range paragraphs {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, gfx.Wrap(p, a.sm.Cols(box.Dx()), 100)...)
	}
	if a.forget.message != "" {
		lines = append(lines, "")
		lines = append(lines, gfx.Wrap(a.forget.message, a.sm.Cols(box.Dx()), 100)...)
	}
	y := box.Min.Y + max(0, (box.Dy()-len(lines)*(a.sm.H+2))/2)
	for _, line := range lines {
		c.Text(box.Min.X, y, a.sm, line, pal.Fg)
		y += a.sm.H + 2
	}
	a.paintHint(c, "Hold A Forget  B Keep")
	if a.forget.bar > 0 {
		c.HLine(l.Hint.Min.X, l.Hint.Min.X+a.forget.bar-1, l.Hint.Min.Y, pal.Ok)
	}
}
