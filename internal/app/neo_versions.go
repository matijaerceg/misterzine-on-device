package app

import (
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// Collapse recognised families after filtering. Keep their first position
// (especially in Recents), putting the preferred visible target there.
func (a *App) collapseNeoVersions(view []int) []int {
	positions := map[string]int{}
	out := view[:0]
	for _, i := range view {
		r := &a.ds.Rows[i]
		if len(a.ds.NeoMembers(r)) < 2 {
			out = append(out, i)
			continue
		}
		pos, seen := positions[r.SN]
		if !seen {
			positions[r.SN] = len(out)
			out = append(out, i)
		} else if a.mode != data.SortRecents {
			j := out[pos]
			missing, prevMissing := a.status(i) == data.StatusNotFound, a.status(j) == data.StatusNotFound
			if !missing && prevMissing || missing == prevMissing && a.ds.Der[i].NeoRank < a.ds.Der[j].NeoRank {
				out[pos] = i
			}
		}
	}
	return out
}

func neoMatches(r *data.Row, q string) bool {
	if !r.IsNeoGeoShortcut() {
		return false
	}
	if strings.Contains(searchText(r.NeoSet), q) || strings.Contains(searchText(data.ASCII(r.NeoVariant)), q) {
		return true
	}
	for _, name := range r.NeoAliases {
		if strings.Contains(searchText(data.ASCII(name)), q) {
			return true
		}
	}
	return false
}

func (a *App) neoVersionSearch(row *data.Row) bool {
	if !row.IsNeoGeoShortcut() || a.query == "" {
		return false
	}
	q := searchText(a.query)
	return q != "" && neoMatches(row, q) && !strings.Contains(searchText(data.ASCII(row.Title)), q)
}

func (a *App) familyFavorite(row *data.Row) bool {
	if a.cfg.Favorites[row.K] || a.cfg.Favorites[row.PreferenceKey()] {
		return true
	}
	for _, j := range a.ds.NeoMembers(row) {
		if a.cfg.Favorites[a.ds.Rows[j].K] {
			return true
		}
	}
	return false
}

func (a *App) neoVersionCount(row *data.Row) int {
	members := a.ds.NeoMembers(row)
	n := len(members)
	if a.cfg.Alternatives != nil {
		for _, j := range members {
			n += len(a.cfg.Alternatives(&a.ds.Rows[j]))
		}
	}
	return n
}
