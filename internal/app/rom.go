package app

import (
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
)

// launchPath is the file Start launches for row i without asking the
// card: the remembered version when the card still offers it, else the
// main MRA. "" for a row without one (a core, or a game not on the card).
func (a *App) launchPath(row *data.Row, i int) string {
	if row.MRA == "" {
		return ""
	}
	if a.cfg.Status != nil && a.status(i) == data.StatusNotFound {
		return ""
	}
	if want := a.cfg.Versions[row.K]; want != "" && want != row.MRA && a.cfg.Alternatives != nil {
		for _, alt := range a.cfg.Alternatives(row) {
			if alt == want {
				return want
			}
		}
	}
	return row.MRA
}

// romState is what the background ROM check knows about row i, memoised
// until ROMChanged: unknown until the launch version has an answer, a
// launch issue when that answer names a problem, an other issue when a
// version the card also offers has one, clean when every version answered
// and none did. A row the check cannot mark (no file, or no ROMKnown hook)
// stays unknown.
func (a *App) romState(i int) data.ROMState {
	if a.cfg.ROMKnown == nil || i < 0 || i >= len(a.ds.Rows) {
		return data.ROMUnknown
	}
	if s, ok := a.romMemo[i]; ok {
		return s
	}
	row := &a.ds.Rows[i]
	s := data.ROMUnknown
	if p := a.launchPath(row, i); p != "" {
		if text, _, known := a.cfg.ROMKnown(p); known {
			switch {
			case text != "":
				s = data.ROMLaunchIssue
			default:
				s = data.ROMClean
				if a.cfg.Alternatives != nil {
					for _, alt := range a.cfg.Alternatives(row) {
						if alt == p {
							continue
						}
						t, _, k := a.cfg.ROMKnown(alt)
						if !k {
							s = data.ROMUnknown
							break
						}
						if t != "" {
							s = data.ROMOtherIssue
							break
						}
					}
				}
			}
		}
	}
	if a.romMemo == nil {
		a.romMemo = map[int]data.ROMState{}
	}
	a.romMemo[i] = s
	return s
}

// romVersions counts row i's versions with a known ROM problem and the
// versions the card offers, for Details.
func (a *App) romVersions(row *data.Row, entries []launchEntry) (bad, total int) {
	if a.cfg.ROMKnown == nil {
		return 0, 0
	}
	for _, e := range entries {
		if !e.ok || strings.HasPrefix(e.path, "core:") {
			continue
		}
		total++
		if text, _, known := a.cfg.ROMKnown(e.path); known && text != "" {
			bad++
		}
	}
	return bad, total
}

// romMark is the list's mark for a row whose version to launch has a ROM
// problem: "!" in the warning colour when MiSTer could not load it, muted
// when it is a warning the game may run with. "" otherwise.
func (a *App) romMark(i int) (string, rgb, bool) {
	if a.romState(i) != data.ROMLaunchIssue {
		return "", rgb{}, false
	}
	if _, block, _ := a.cfg.ROMKnown(a.launchPath(&a.ds.Rows[i], i)); block {
		return "!", gen.Eva.Warn, true
	}
	return "!", gen.Eva.Muted, true
}

// ROMChanged tells the app the background ROM check has new answers, or
// the versions it judges by changed: the memo goes, the list repaints,
// and a ROM filter in force re-applies.
func (a *App) ROMChanged() {
	a.romMemo = nil
	if a.filters.ROMActive() {
		a.Refilter()
		return
	}
	if a.screen == ScreenFilter {
		a.buildPanel() // the section's counts and progress line
	}
	a.Invalidate()
}
