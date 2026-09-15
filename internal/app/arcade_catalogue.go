package app

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// catalogueIncludes ignores temporary filters and search, but respects Options.
func (a *App) catalogueIncludes(r *data.Row) bool {
	return (a.cfg.ShowNonArcade || r.IsArcade()) && (a.cfg.ShowDeprecated || !r.Deprecated) && !(a.cfg.InstalledOnly && a.hiddenSrc[r.Src])
}

// CatalogueNews describes changes within the enabled catalogue only.
func (a *App) CatalogueNews(old *data.Dataset, rows []data.Row) string {
	visible := make([]data.Row, 0, len(rows))
	for _, r := range rows {
		if a.catalogueIncludes(&r) {
			visible = append(visible, r)
		}
	}
	return data.DiffNews(old, visible)
}

const arcadeIntroHold = 2 * time.Second

func (a *App) handleArcadeIntro(ev platform.Event) bool {
	at := ev.At
	if at.IsZero() {
		at = a.cfg.TimerNow()
	}
	a.saver.lastInput = at
	if ev.Key != platform.KeyEnter {
		return false
	}
	if ev.Pressed {
		if a.down[ev.Key] {
			return false
		}
		a.down[ev.Key] = true
		a.arcadeIntroAt, a.arcadeIntroBar = at, 0
		return false
	}
	if !a.down[ev.Key] {
		return false
	}
	// A release at the deadline also completes a hold if the event arrived
	// before the scheduled tick. Short holds reset without dismissing.
	a.tickArcadeIntro(at)
	delete(a.down, ev.Key)
	a.arcadeIntroAt, a.arcadeIntroBar = time.Time{}, 0
	a.all = true
	return true
}

func (a *App) tickArcadeIntro(now time.Time) bool {
	if !a.cfg.ArcadeIntro || a.arcadeIntroAt.IsZero() {
		return false
	}
	if now.Sub(a.arcadeIntroAt) >= arcadeIntroHold {
		a.cfg.ArcadeIntro = false
		a.arcadeIntroAt, a.arcadeIntroBar = time.Time{}, 0
		// Keep the held key down until its release so it cannot open Details.
		a.settingsChanged()
		a.all = true
		return true
	}
	if bar := a.holdBarWidth(a.arcadeIntroAt, arcadeIntroHold, now); bar != a.arcadeIntroBar {
		a.arcadeIntroBar = bar
		a.all = true
		return true
	}
	return false
}

func (a *App) nextArcadeIntroTick() time.Time {
	if !a.cfg.ArcadeIntro || a.arcadeIntroAt.IsZero() {
		return time.Time{}
	}
	return a.nextHoldPixel(a.arcadeIntroAt, arcadeIntroHold, a.arcadeIntroBar)
}

func (a *App) paintArcadeIntro(c *gfx.Canvas) {
	c.Fill(a.lay.Body, gen.Eva.Bg)
	box := a.lay.Body.Inset(4)
	c.Fill(box, gen.Eva.Bg)
	c.Box(box, gen.Eva.Line)
	x, y := box.Min.X+5, box.Min.Y+5
	for _, line := range gfx.Wrap("MisterZine now shows arcade games by default. Enable other cores in Options > List > Show non-arcade cores. Your favorites are kept.", a.sm.Cols(box.Dx()-10), 30) {
		c.Text(x, y, a.sm, line, gen.Eva.Fg)
		y += a.sm.H + 2
	}
	a.paintHint(c, "Hold A 2 s to continue")
}
