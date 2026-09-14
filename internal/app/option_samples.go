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

// Twelve rows in each direction, with the chosen initial-repeat delay at each end.
func sampleListOffset(elapsed, pace, delay time.Duration) int {
	if elapsed < 0 {
		elapsed = 0
	}
	if delay == 0 {
		return int(elapsed / pace)
	}
	travel := 12 * pace
	phase := elapsed % (2 * (delay + travel))
	if phase < delay {
		return 0
	}
	phase -= delay
	if phase < travel {
		return int(phase / pace)
	}
	phase -= travel
	if phase < delay {
		return 12
	}
	return 12 - int((phase-delay)/pace)
}

func (a *App) paintOptionSamples(c *gfx.Canvas, area image.Rectangle, e panelEntry) {
	c.Fill(area, gen.Eva.Surface)
	c.Box(area, gen.Eva.Line)
	if e.kind != "title-font" && a.optionSamples.kind != e.kind {
		a.tickOptionSamples(a.cfg.TimerNow())
	}
	for i := 0; i < 3; i++ {
		cell := image.Rect(area.Min.X+3+i*(area.Dx()-6)/3, area.Min.Y+3, area.Min.X+3+(i+1)*(area.Dx()-6)/3-2, area.Max.Y-3)
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
		offset := sampleListOffset(a.optionSamples.now.Sub(a.optionSamples.start), pace, delay)
		// Unequal strokes make each moving row recognizable without tiny text.
		for row := 0; row < 4; row++ {
			y := r.Min.Y + row*(r.Dy()-1)/3
			width := r.Dx() * (5 + (offset+row)%7) / 12
			c.HLine(r.Min.X, r.Min.X+width-1, y, col)
		}
	}
}
