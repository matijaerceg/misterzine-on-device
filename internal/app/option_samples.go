package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"image"
	"time"
)

type optionSampleClock struct {
	kind    string
	elapsed time.Duration // advances once per displayed frame, never from wall time
}

func (a *App) animatedOption() string {
	if a.screen == ScreenOptions && !a.saver.active && a.panel.cursor < len(a.panel.entries) {
		k := a.panel.entries[a.panel.cursor].kind
		if k == "scroll" || k == "smooth-scroll" || k == "hold-delay" {
			return k
		}
	}
	return ""
}

// OptionSamplesRunning asks the host for the same vertical-blank cadence as scrolling.
func (a *App) OptionSamplesRunning() bool { return a.animatedOption() != "" }

func (a *App) tickOptionSamples() bool {
	k := a.animatedOption()
	if a.optionSamples.kind == k {
		return false
	}
	a.optionSamples = optionSampleClock{kind: k}
	if k == "" {
		return false
	}
	a.all = true
	return true
}

// OptionSampleFrame advances by one refresh, even if background work delayed it.
// A late frame must never skip over one or more selected rows.
func (a *App) OptionSampleFrame() bool {
	a.tickOptionSamples()
	if !a.OptionSamplesRunning() || a.PageTransitionRunning() {
		return false
	}
	a.optionSamples.elapsed += frameDur
	a.all = true
	return true
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
	if e.kind == "smooth-scroll" {
		a.paintSmoothScrollSamples(c, area, e)
		return
	}
	c.Fill(area, gen.Eva.Surface)
	c.Box(area, gen.Eva.Line)
	if e.kind != "title-font" && a.optionSamples.kind != e.kind {
		a.tickOptionSamples()
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
			delay = ((delay + frameDur/2) / frameDur) * frameDur
		}
		r = cell.Inset(1)
		rows := 3
		if a.lay.Portrait {
			rows = 4
		}
		selected := sampleListSelection(a.optionSamples.elapsed, pace, delay, rows)
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

// Both samples share a clock and row speed; only the intermediate frames differ.
func smoothSampleTravel(elapsed, pace time.Duration, smooth bool, line int) int {
	frames := int(elapsed / frameDur)
	every := max(1, int(pace/frameDur))
	if smooth {
		return frames * line / every
	}
	return frames / every * line
}

func (a *App) paintSmoothScrollSamples(c *gfx.Canvas, area image.Rectangle, e panelEntry) {
	c.Fill(area, gen.Eva.Surface)
	c.Box(area, gen.Eva.Line)
	if a.optionSamples.kind != e.kind {
		a.tickOptionSamples()
	}
	names := []string{"Galaga", "Pac-Man", "Out Run", "1942", "R-Type", "Gradius"}
	for i, label := range []string{"off", "on"} {
		cell := image.Rect(area.Min.X+3+i*(area.Dx()-6)/2, area.Min.Y+3, area.Min.X+3+(i+1)*(area.Dx()-6)/2-2, area.Max.Y-3)
		c.Fill(cell, gen.Eva.Bg)
		col := gen.Eva.Fg
		if i == e.idx {
			col = gen.Eva.Accent
			c.Box(cell, col)
		}
		c.Text(cell.Min.X+(cell.Dx()-a.sm.Width(label))/2, cell.Min.Y+1, a.sm, label, col)
		viewport := cell.Inset(2)
		viewport.Min.Y += a.sm.H + 1
		clipped := &gfx.Canvas{RGBA: c.Sub(viewport)}
		line := a.lay.Line
		travel := smoothSampleTravel(a.optionSamples.elapsed, scrollPace(a.ScrollSpeed()), i == 1, line)
		first, offset := travel/line, travel%line
		selectedY := viewport.Min.Y + max(0, (viewport.Dy()/line-1)/2)*line
		clipped.Fill(image.Rect(viewport.Min.X, selectedY, viewport.Max.X, selectedY+line), gen.Eva.Surface)
		for row, y := first, viewport.Min.Y-offset; y < viewport.Max.Y; row, y = row+1, y+line {
			clipped.Text(viewport.Min.X+1, y+2, a.sm, gfx.Fit(names[row%len(names)], a.sm.Cols(viewport.Dx()-2)), gen.Eva.Fg)
		}
		c.Dirty(viewport)
	}
}
