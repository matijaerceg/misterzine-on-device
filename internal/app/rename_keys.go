package app

import "github.com/matijaerceg/misterzine-on-device/internal/data"

// AdoptStrandedLocalKeys moves the favorites, remembered versions and launch
// records still filed under a local row this run never listed, because the
// catalogue already names its game (data.StrandedLocalKeys), and returns the
// moves. The host calls it once a complete card scan has settled the local
// rows, so a local or standin row the card still has keeps its key.
func (a *App) AdoptStrandedLocalKeys() map[string]string {
	if a.ds == nil {
		return nil
	}
	var keys []string
	for k := range a.cfg.Favorites {
		keys = append(keys, k)
	}
	for k := range a.cfg.Versions {
		keys = append(keys, k)
	}
	for _, r := range a.cfg.RecentLaunches {
		keys = append(keys, r.K)
	}
	moves := data.StrandedLocalKeys(keys, a.ds.Rows, a.ds.Catalogue())
	a.RenameKeys(moves)
	return moves
}

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
