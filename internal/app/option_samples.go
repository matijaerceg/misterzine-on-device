package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"image"
	"time"
)

type optionSampleClock struct {
	kind             string
	start, now, next time.Time
}

func (a *App) animatedOption() string {
	if a.screen == ScreenOptions && a.panel.cursor < len(a.panel.entries) {
		k := a.panel.entries[a.panel.cursor].kind
		if k == "scroll" || k == "hold-delay" {
			return k
		}
	}
	return ""
}
func (a *App) tickOptionSamples(now time.Time) bool {
	k := a.animatedOption()
	if k == "" {
		a.optionSamples = optionSampleClock{}
		return false
	}
	s := &a.optionSamples
	if s.kind != k {
		*s = optionSampleClock{kind: k, start: now}
	}
	if !s.next.IsZero() && now.Before(s.next) {
		return false
	}
	s.now, s.next = now, now.Add(frameDur)
	a.all = true
	return true
}
func (a *App) nextOptionSampleTick() time.Time {
	if a.animatedOption() == "" {
		return time.Time{}
	}
	if a.optionSamples.kind != a.animatedOption() || a.optionSamples.next.IsZero() {
		return a.cfg.TimerNow()
	}
	return a.optionSamples.next
}

// The names stay fixed: only the selection loops or reverses, pausing at each end.
func sampleListSelection(elapsed, pace, delay time.Duration, rows int) int {
	if rows < 2 {
		return 0
	}
	if elapsed < 0 {
		elapsed = 0
	}
	if delay == 0 {
		return int(elapsed/pace) % rows
	}
	last := rows - 1
	half := delay + time.Duration(last-1)*pace
	phase := elapsed % (2 * half)
	reverse := phase >= half
	if reverse {
		phase -= half
	}
	selected := 0
	if phase >= delay {
		selected = 1 + int((phase-delay)/pace)
	}
	if reverse {
		return last - selected
	}
	return selected
}

func (a *App) paintOptionSamples(c *gfx.Canvas, area image.Rectangle, e panelEntry) {
	c.Fill(area, gen.Eva.Surface)
	c.Box(area, gen.Eva.Line)
	if e.kind != "title-font" && a.optionSamples.kind != e.kind {
		a.tickOptionSamples(a.cfg.TimerNow())
	}
	for i := 0; i < 3; i++ {
		cell := image.Rect(area.Min.X+3+i*(area.Dx()-6)/3, area.Min.Y+3, area.Min.X+3+(i+1)*(area.Dx()-6)/3-2, area.Max.Y-3)
		if e.kind != "title-font" {
			c.Fill(cell, gen.Eva.Bg)
		}
		col := gen.Eva.Fg
		if i == e.idx {
			col = gen.Eva.Accent
			c.Box(cell, col)
		}
		r := cell.Inset(3)
		if e.kind == "title-font" {
			f := []*gfx.Font{a.body, a.narrow, a.tall}[i]
			text := gfx.FitProp(f, "Game", r.Dx())
			if i == 0 {
				text = gfx.Fit("Game", f.Cols(r.Dx()))
				c.Text(r.Min.X+(r.Dx()-f.Width(text))/2, r.Min.Y+(r.Dy()-f.H)/2, f, text, col)
			} else {
				c.TextProp(r.Min.X+(r.Dx()-f.PropWidth(text))/2, r.Min.Y+(r.Dy()-f.H)/2, f, text, col)
			}
			continue
		}
		pace, delay := scrollPace(ScrollValues[i]), time.Duration(0)
		if e.kind == "hold-delay" {
			pace = scrollPace(a.ScrollSpeed())
			delay = time.Duration([]int{200, 300, 500}[i]) * time.Millisecond
		}
		r = cell.Inset(1)
		rows := 3
		if a.lay.Portrait {
			rows = 4
		}
		selected := sampleListSelection(a.optionSamples.now.Sub(a.optionSamples.start), pace, delay, rows)
		names := []string{"Galaga", "Pac-Man", "Out Run", "1942"}
		top := r.Min.Y + (r.Dy()-rows*a.sm.H)/2
		for row := 0; row < rows; row++ {
			y := top + row*a.sm.H
			textCol := gen.Eva.Fg
			if row == selected {
				c.Fill(image.Rect(r.Min.X, y, r.Max.X, y+a.sm.H), gen.Eva.Surface)
				textCol = gen.Eva.Accent
			}
			c.Text(r.Min.X+1, y, a.sm, gfx.Fit(names[row], a.sm.Cols(r.Dx()-2)), textCol)
		}
	}
}
