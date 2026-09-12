package app

import (
	"fmt"
	"image"
	"reflect"
	"strings"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

const cancelHold = 2 * time.Second

type updateView struct {
	backAt     time.Time
	now        time.Time
	cancelSent bool
	scroll     int
	lines      int
	log        []string // last painted log; held still while scrolled back
	error      string
}

func (a *App) UpdateState() updater.State { return a.update }

func (a *App) OpenUpdate() {
	a.rep = repeater{}
	a.down = map[platform.Key]bool{}
	a.updateView = updateView{now: a.cfg.TimerNow()}
	a.update = updater.State{Status: "starting", Label: "Starting Update All", Started: a.cfg.Now()}
	a.screen = ScreenUpdate
	a.all = true
	if a.cfg.Action != nil {
		a.cfg.Action("update", "")
	}
}

func (a *App) SetUpdate(s updater.State, open bool) {
	if !open && reflect.DeepEqual(a.update, s) {
		return
	}
	newRun := a.update.ID != s.ID
	pendingStart := a.screen == ScreenUpdate && a.update.ID == "" && a.update.Active() && s.Active()
	if newRun {
		held := a.updateView.backAt
		a.updateView = updateView{now: a.cfg.TimerNow()}
		if pendingStart {
			a.updateView.backAt = held
		}
	}
	a.update = s
	if !s.Active() {
		a.updateView.backAt = time.Time{}
	}
	if open {
		if a.screen != ScreenUpdate || (newRun && !pendingStart) {
			a.rep = repeater{}
			a.down = map[platform.Key]bool{}
		}
		a.screen = ScreenUpdate
	}
	if a.screen == ScreenUpdate {
		a.all = true
	}
}

func (a *App) UpdateCancelError(message string) {
	a.updateView.cancelSent = false
	a.updateView.backAt = time.Time{}
	a.updateView.error = "Cancel failed: " + message + ". Hold " + a.btn("B") + " to retry."
	a.all = true
}

func (a *App) handleUpdate(ev platform.Event) bool {
	if !ev.Pressed {
		delete(a.down, ev.Key)
		a.rep.release(ev.Key)
		if ev.Key == platform.KeyBack {
			a.updateView.backAt = time.Time{}
			a.all = true
		}
		return true
	}
	if a.down[ev.Key] {
		return false
	}
	a.down[ev.Key] = true
	switch ev.Key {
	case platform.KeyMenu:
		return a.menuButton()
	case platform.KeyBack:
		if !a.update.Active() {
			a.rep = repeater{}
			a.openPanel(ScreenOptions)
			if a.update.ResultNotice() && a.cfg.Action != nil {
				a.cfg.Action("update-dismiss", a.update.ID)
			}
		} else if !a.update.CancelRequested && !a.updateView.cancelSent {
			a.updateView.backAt = ev.At
			a.updateView.error = ""
		}
	case platform.KeyUp, platform.KeyDown, platform.KeyPageUp, platform.KeyPageDown:
		a.rep.press(ev.Key, ev.At)
		a.rep.next = ev.At.Add(time.Duration(a.HoldDelay()) * time.Millisecond)
		return a.scrollUpdate(ev.Key)
	default:
		return false // no launch, filter, options or quit action during the run
	}
	a.all = true
	return true
}

func (a *App) scrollUpdate(k platform.Key) bool {
	switch k {
	case platform.KeyUp:
		a.updateView.scroll++
	case platform.KeyDown:
		a.updateView.scroll--
	case platform.KeyPageUp:
		a.updateView.scroll += max(1, a.updateView.lines)
	case platform.KeyPageDown:
		a.updateView.scroll -= max(1, a.updateView.lines)
	default:
		return false
	}
	a.updateView.scroll = max(0, a.updateView.scroll)
	a.all = true
	return true
}

func (a *App) tickUpdate(now time.Time) bool {
	if !a.update.Active() {
		return false
	}
	v := &a.updateView
	if !v.backAt.IsZero() && !v.cancelSent && now.Sub(v.backAt) >= cancelHold && a.update.Active() && a.update.ID != "" {
		v.cancelSent = true
		if a.cfg.Action != nil {
			a.cfg.Action("update-cancel", a.update.ID)
		}
	}
	v.now = now
	a.all = true
	return true
}

func (a *App) paintUpdate(c *gfx.Canvas) {
	l := a.lay
	s := a.update
	v := &a.updateView
	c.Fill(l.Root, gen.Eva.Bg)
	c.Fill(l.Status, gen.Eva.Surface)
	c.Text(l.Status.Min.X+2, l.Status.Min.Y+2, a.sm, "Update All", gen.Eva.Accent)
	status := s.Status
	if s.Active() {
		spin := []string{"|", "/", "-", "\\"}[int((v.now.UnixMilli()/250)&3)]
		if !s.Heartbeat.IsZero() && v.now.Sub(s.Heartbeat) > 3*time.Second {
			spin = "?"
			status = "no status"
		}
		status = spin + " " + status
	}
	c.TextRight(l.Status.Max.X-2, l.Status.Min.Y+2, a.sm, status, gen.Eva.Fg)
	y := l.Body.Min.Y + 3
	if s.Active() && s.Protected {
		r := image.Rect(l.Body.Min.X, y, l.Body.Max.X, y+12)
		c.Fill(r, gen.Eva.Surface)
		c.Text(r.Min.X+3, y+2, a.sm, gfx.Fit("System write: keep power on", a.sm.Cols(r.Dx()-6)), gen.Eva.Accent)
		y += 15
	}
	if s.Reboot {
		r := image.Rect(l.Body.Min.X, y, l.Body.Max.X, y+12)
		c.Fill(r, gen.Eva.Surface)
		c.Text(r.Min.X+3, y+2, a.sm, "Restart expected", gen.Eva.Accent)
		y += 15
	}
	label := s.Label
	if label == "" {
		label = "Preparing Update All"
	}
	c.Text(l.Body.Min.X+2, y, a.body, gfx.Fit(label, a.body.Cols(l.Body.Dx()-4)), gen.Eva.Fg)
	y += a.body.H + 3
	w := (l.Body.Dx() - 4) / len(updater.Stages)
	for i := range updater.Stages {
		r := image.Rect(l.Body.Min.X+2+i*w, y, l.Body.Min.X+2+(i+1)*w-2, y+7)
		c.Box(r, gen.Eva.Line)
		if i < s.Stage {
			c.Fill(r.Inset(1), gen.Eva.Accent)
		} else if i == s.Stage && s.Active() {
			c.Fill(r.Inset(1), gen.Eva.Muted)
		}
	}
	y += 10
	for i, name := range updater.Stages {
		c.Text(l.Body.Min.X+2+i*w, y, a.sm, gfx.Fit(name, a.sm.Cols(w-2)), gen.Eva.Muted)
	}
	y += a.sm.H + 3
	age := 0
	if !s.LastOutput.IsZero() {
		age = max(0, int(v.now.Sub(s.LastOutput).Seconds()))
	}
	elapsed := fmt.Sprintf("%d:%02d elapsed", s.Elapsed/60, s.Elapsed%60)
	if v.scroll > 0 {
		elapsed += "  log paused"
	} else if s.Active() {
		elapsed += fmt.Sprintf("  output %ds ago", age)
	}
	c.Text(l.Body.Min.X+2, y, a.sm, gfx.Fit(elapsed, a.sm.Cols(l.Body.Dx()-4)), gen.Eva.Muted)
	y += a.sm.H + 3
	message := s.Summary()
	if s.Active() && s.Protected && (s.CancelRequested || v.cancelSent) {
		message = "Cancel queued until system write ends"
	}
	if s.Active() && v.cancelSent && !s.CancelRequested && !s.Protected {
		message = "Requesting cancellation"
	}
	if s.Active() && v.error != "" {
		message = v.error
	}
	if s.Active() && !v.backAt.IsZero() && !v.cancelSent {
		left := max(0, 2-int(v.now.Sub(v.backAt).Seconds()))
		message = fmt.Sprintf("Keep holding %s to cancel (%ds)", a.btn("B"), left)
	}
	for _, line := range gfx.Wrap(message, a.sm.Cols(l.Body.Dx()-4), 2) {
		c.Text(l.Body.Min.X+2, y, a.sm, line, gen.Eva.Fg)
		y += a.sm.H + 1
	}
	y += 3
	box := image.Rect(l.Body.Min.X, y, l.Body.Max.X, l.Body.Max.Y-2)
	c.Box(box, gen.Eva.Line)
	v.lines = max(1, (box.Dy()-4)/(a.sm.H+1))
	if v.scroll == 0 || v.log == nil {
		v.log = nil
		for _, line := range s.Lines {
			if strings.Trim(line, "#=-_*+|│─━═╔╗╚╝╠╣╦╩╬ ") == "" {
				continue
			}
			v.log = append(v.log, gfx.Wrap(strings.TrimSpace(line), a.sm.Cols(box.Dx()-6), 20)...)
		}
		if len(v.log) == 0 {
			v.log = []string{"Waiting for updater output..."}
		}
	}
	lines := v.log
	v.scroll = min(v.scroll, max(0, len(lines)-v.lines))
	end := len(lines) - v.scroll
	start := max(0, end-v.lines)
	y = box.Min.Y + 2
	for _, line := range lines[start:end] {
		c.Text(box.Min.X+3, y, a.sm, line, gen.Eva.Fg)
		y += a.sm.H + 1
	}
	hint := "B Options  " + gfx.ArrowUp + " " + gfx.ArrowDown + " log"
	if s.Active() {
		hint = "Hold B cancel  " + gfx.ArrowUp + " " + gfx.ArrowDown + " log"
	}
	if s.Active() && s.CancelRequested {
		hint = gfx.ArrowUp + " " + gfx.ArrowDown + " log"
	}
	a.paintHint(c, hint)
}
