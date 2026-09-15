package app

import (
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

func (a *App) handleArcadeIntro(ev platform.Event) bool {
	a.saver.lastInput = ev.At
	if ev.Key != platform.KeyEnter && ev.Key != platform.KeyBack {
		return false
	}
	if ev.Pressed {
		a.down[ev.Key] = true
		return false
	}
	if !a.down[ev.Key] {
		return false
	}
	a.down = map[platform.Key]bool{}
	a.cfg.ArcadeIntro = false
	a.all = true
	if a.cfg.SettingsChanged != nil {
		a.cfg.SettingsChanged()
	}
	return true
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
	a.paintHint(c, "A Continue  B Dismiss")
}
