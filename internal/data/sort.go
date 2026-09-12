package data

import "sort"

// SortMode selects the device's list order.
type SortMode int

const (
	// SortUpdated is the site's default: newest shipped build first, equal
	// dates ordered by arrival batch, then core label and title.
	SortUpdated SortMode = iota
	// SortDebut is newest MiSTer debut first, then title.
	SortDebut
	// SortAlphabetical orders titles naturally, ignoring case and accents.
	SortAlphabetical
	// SortFavorites is an alphabetical favorites-only browsing mode.
	SortFavorites
	// SortYear is newest original release year first (the catalogue has
	// years only, so titles order within a year); unknown years last. It
	// follows SortFavorites in value because the value is saved, but sits
	// after Debut in the browsing cycle.
	SortYear
	// SortRecents lists the games launched from the app, latest launch first.
	// It joins the browsing cycle after Favorites only when Options ->
	// Recents view is on; the app orders it from its launch history.
	SortRecents
)

// Recent is one launch from the app: the row key and the UTC time (RFC 3339).
type Recent struct {
	K  string `json:"k"`
	At string `json:"at"`
}

// OrderRecents returns the row indexes of the launched games in history
// order (the history is newest first); keys no longer in the catalogue
// are skipped and a key counts once.
func (ds *Dataset) OrderRecents(recents []Recent) []int {
	byKey := make(map[string]int, len(ds.Rows))
	for i := range ds.Rows {
		byKey[ds.Rows[i].K] = i
	}
	out := make([]int, 0, len(recents))
	seen := map[string]bool{}
	for _, r := range recents {
		if i, ok := byKey[r.K]; ok && !seen[r.K] {
			seen[r.K] = true
			out = append(out, i)
		}
	}
	return out
}

// SortCycle is the order Y walks the modes in.
var SortCycle = []SortMode{SortUpdated, SortDebut, SortYear, SortAlphabetical, SortFavorites}

// NextSort is the mode after m in the cycle.
func NextSort(m SortMode) SortMode {
	for i, mode := range SortCycle {
		if mode == m {
			return SortCycle[(i+1)%len(SortCycle)]
		}
	}
	return SortUpdated
}

func (m SortMode) String() string {
	switch m {
	case SortDebut:
		return "Debut"
	case SortAlphabetical:
		return "Alphabetical"
	case SortFavorites:
		return "Favorites"
	case SortYear:
		return "Year"
	case SortRecents:
		return "Recents"
	}
	return "Updated"
}

// Order returns row indexes in the site's order for the mode. It mirrors the
// apply() comparator: blank dates sink to the bottom whatever the direction,
// the date compares descending. Only the updated sort breaks equal dates by
// arrival batch descending, then core ascending; title breaks the rest.
// These tiebreaks are never direction-flipped.
// The sort is stable over data.json order, like Array.prototype.sort.
func (ds *Dataset) Order(mode SortMode) []int {
	idx := make([]int, len(ds.Rows))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(x, y int) bool {
		return ds.less(mode, idx[x], idx[y])
	})
	return idx
}

func (ds *Dataset) less(mode SortMode, a, b int) bool {
	ra, rb := &ds.Rows[a], &ds.Rows[b]
	da, db := &ds.Der[a], &ds.Der[b]
	if mode == SortAlphabetical || mode == SortFavorites || mode == SortRecents {
		return CompareKeys(da.titleKey, db.titleKey) < 0
	}
	if mode == SortYear {
		ay, by := ReleaseYear(ra.Year), ReleaseYear(rb.Year)
		if (ay == "") != (by == "") {
			return ay != "" // a known year comes first
		}
		if ay != by {
			return ay > by // newest first
		}
		return CompareKeys(da.titleKey, db.titleKey) < 0
	}
	var av, bv string
	var ak, bk []elem
	if mode == SortDebut {
		av, bv, ak, bk = ra.Date, rb.Date, da.dateKey, db.dateKey
	} else {
		av, bv, ak, bk = ra.Updated, rb.Updated, da.updatedKey, db.updatedKey
	}
	ae, be := av == "", bv == ""
	if ae != be {
		return !ae // the non-blank one comes first
	}
	if c := CompareKeys(ak, bk); c != 0 {
		return c > 0 // descending
	}
	if mode == SortUpdated {
		// Later same-day arrivals stay above earlier refreshes, matching the
		// site. Old feeds omit b and tie at zero, preserving their prior order.
		if ra.B != rb.B {
			return ra.B > rb.B
		}
		if c := CompareKeys(da.coreKey, db.coreKey); c != 0 {
			return c < 0
		}
	}
	return CompareKeys(da.titleKey, db.titleKey) < 0
}
