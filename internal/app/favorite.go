package app

import "time"

// toggleFavorite stars or unstars the row under the cursor, from the list
// (Select + A) or Details (Y), says which in a notice and saves. The view
// is rebuilt, since Favorites drops an unstarred row: the row is found
// again where it still shows, and the cursor stays in range where it has
// gone. True when the row left the view.
func (a *App) toggleFavorite() (left bool) {
	if a.cursor >= len(a.view) || a.umbrellaAt(a.cursor) != nil {
		return false // an umbrella takes no star: its members do
	}
	if a.cfg.FavoritesUnavailable {
		a.Notice(FavoritesUnavailableNotice, 8*time.Second)
		return false
	}
	if a.cfg.Favorites == nil {
		a.cfg.Favorites = map[string]bool{}
	}
	row := &a.ds.Rows[a.view[a.cursor]]
	k := row.K
	if a.familyFavorite(row) {
		delete(a.cfg.Favorites, k)
		delete(a.cfg.Favorites, row.PreferenceKey())
		for _, i := range a.ds.NeoMembers(row) {
			delete(a.cfg.Favorites, a.ds.Rows[i].K)
		}
		a.Notice("favorite removed", 2*time.Second)
	} else {
		a.cfg.Favorites[row.PreferenceKey()] = true
		a.Notice("favorite added", 2*time.Second)
	}
	if a.cfg.FavChanged != nil {
		a.cfg.FavChanged()
	}
	a.rebuild()
	a.moveToKey(k)
	left = a.CursorKey() != k
	if left {
		a.cursor = max(0, min(a.cursor, len(a.view)-1))
		a.ensureVisible()
	}
	a.all = true
	return left
}
