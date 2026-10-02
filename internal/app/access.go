package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/access"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

func (a *App) ShowBetaFeatures() bool { return a.cfg.ShowBetaFeatures }
func (a *App) featureAllowed(f access.Feature) bool {
	return f.Allowed(a.cfg.AccessMonth, a.cfg.ShowBetaFeatures)
}
func (a *App) featureVisible(f access.Feature) bool { return f.Visible(a.cfg.ShowBetaFeatures) }
func featureMarks(f access.Feature) string {
	s := ""
	if f.Fancy {
		s += " " + gfx.Star
	}
	if f.Beta {
		s += " " + gfx.Beta
	}
	return s
}

func (a *App) gatedOption(row panelEntry, f access.Feature) panelEntry {
	row.text += featureMarks(f)
	if !a.featureAllowed(f) {
		row.kind = "enter-code"
		row.vals = []string{"locked"}
		row.idx = 0
		row.opensPage = true
		row.disabled = true
		row.help = "Needs " + f.Since.String() + " code or newer. patreon.com/MisterZine. " + a.btn("A") + " enters code."
	}
	return row
}

func (a *App) accessChanged() {
	selected := ""
	if a.panel.cursor < len(a.panel.entries) {
		selected = a.panel.entries[a.panel.cursor].text
	}
	a.membersStart()
	if a.cfg.AccessChanged != nil {
		a.cfg.AccessChanged(a.cfg.AccessMonth, a.cfg.ShowBetaFeatures)
	}
	a.buildPanel()
	for i, e := range a.panel.entries {
		if e.text == selected {
			a.panel.cursor = i
			break
		}
	}
	a.all = true
}

func (a *App) openCode() {
	a.lock = newBetaLock()
	a.lock.optional = true
	a.all = true
}

func (a *App) SetROMList(path string) { a.cfg.ROMList = path; a.all = true }
