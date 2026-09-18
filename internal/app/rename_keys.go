package app

// RenameKeys moves favorites, remembered versions and launch records from
// old row keys to new ones (data.LocalTakeovers), when a game the card scan
// listed on its own joins the catalogue. An entry the new key already has
// is kept as it is; the old key is dropped either way.
func (a *App) RenameKeys(moves map[string]string) {
	if len(moves) == 0 {
		return
	}
	favs, versions, recents := false, false, false
	for old, k := range moves {
		if a.cfg.Favorites[old] {
			delete(a.cfg.Favorites, old)
			a.cfg.Favorites[k] = true
			favs = true
		}
		if v, ok := a.cfg.Versions[old]; ok {
			delete(a.cfg.Versions, old)
			if _, taken := a.cfg.Versions[k]; !taken {
				a.cfg.Versions[k] = v
			}
			versions = true
		}
	}
	seen := map[string]bool{}
	kept := a.cfg.RecentLaunches[:0]
	for _, r := range a.cfg.RecentLaunches {
		if k, ok := moves[r.K]; ok {
			r.K = k
			recents = true
		}
		if seen[r.K] {
			recents = true
			continue // the newer record (earlier in the list) wins
		}
		seen[r.K] = true
		kept = append(kept, r)
	}
	a.cfg.RecentLaunches = kept
	if favs && a.cfg.FavChanged != nil {
		a.cfg.FavChanged()
	}
	if versions && a.cfg.VersionChanged != nil {
		a.cfg.VersionChanged()
	}
	if recents && a.cfg.RecentsChanged != nil {
		a.cfg.RecentsChanged()
	}
	if favs || recents {
		a.rebuild()
		a.all = true
	}
}
