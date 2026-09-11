package app

import "github.com/matijaerceg/misterzine-on-device/internal/data"

// rememberedPick is the index, among the current row's launch entries, of
// the version chosen last time: 0 (the main version) when nothing is
// remembered or the remembered file has since left the card.
func (a *App) rememberedPick() int {
	row, _, i := a.current()
	if row == nil {
		return 0
	}
	want := a.cfg.Versions[row.K]
	if want == "" {
		return 0
	}
	for n, e := range a.launchEntries(row, i) {
		if e.path == want {
			return n
		}
	}
	return 0
}

// rememberPick records entry pick of the row as the version to open and
// launch next time. The main version is the default, so choosing it clears
// the record instead.
func (a *App) rememberPick(row *data.Row, entries []launchEntry, pick int) {
	if row == nil || len(entries) == 0 {
		return
	}
	p := ""
	if pick > 0 && pick < len(entries) {
		p = entries[pick].path
	}
	if a.cfg.Versions[row.K] == p {
		return
	}
	if p == "" {
		delete(a.cfg.Versions, row.K)
	} else {
		a.cfg.Versions[row.K] = p
	}
	if a.cfg.VersionChanged != nil {
		a.cfg.VersionChanged()
	}
}
