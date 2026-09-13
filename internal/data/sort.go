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
	// SortMaker groups the games by manufacturer (Derived.Maker: the first
	// company of the credit, corporate suffixes dropped), makers A-Z with
	// titles A-Z inside and the unknown maker last. Last in value because
	// the value is saved; it sits after A-Z in the browsing cycle.
	SortMaker
)

// Valid reports whether m is a sort mode the app knows.
func (m SortMode) Valid() bool { return m >= SortUpdated && m <= SortMaker }

// ViewOrder is every view in the order Y walks them, Recents included.
var ViewOrder = []SortMode{SortUpdated, SortDebut, SortYear, SortAlphabetical, SortMaker, SortFavorites, SortRecents}

var sortNames = map[SortMode]string{SortUpdated: "updated", SortDebut: "debut", SortYear: "year", SortAlphabetical: "alphabetical", SortMaker: "maker", SortFavorites: "favorites", SortRecents: "recents"}

// Name is the mode's settings name (Options -> Views keeps the ones left
// out of the cycle by name).
func (m SortMode) Name() string { return sortNames[m] }

// ParseSort is the inverse of Name.
func ParseSort(name string) (SortMode, bool) {
	for m, n := range sortNames {
		if n == name {
			return m, true
		}
	}
	return SortUpdated, false
}

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

// SortCycle is the order Y walks the modes in (Recents joins after
// Favorites when it is on; the app skips the views turned off).
var SortCycle = []SortMode{SortUpdated, SortDebut, SortYear, SortAlphabetical, SortMaker, SortFavorites}

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
	case SortMaker:
		return "Manufacturer"
	}
	return "Updated"
}

// Order returns row indexes in the site's order for the mode. It mirrors the
// apply() comparator: blank dates sink to the bottom whatever the direction,
// the date compares descending. Only the updated sort breaks equal dates by
// arrival batch descending, then core ascending; title breaks the rest.
// These tiebreaks are never direction-flipped.
// The sort is stable over data.json order, like Array.prototype.sort.
//
// The rows never change after Ingest, so each mode is sorted once per
// Dataset and the same slice is returned after that: callers read it and
// never write to it (Apply copies). A full sort of the catalogue costs
// 6-20 ms on the MiSTer's ARM core, and the list is rebuilt on every Y
// press, search keystroke and status arrival.
func (ds *Dataset) Order(mode SortMode) []int {
	ds.orderMu.Lock()
	defer ds.orderMu.Unlock()
	if idx, ok := ds.orders[mode]; ok {
		return idx
	}
	idx := make([]int, len(ds.Rows))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(x, y int) bool {
		return ds.less(mode, idx[x], idx[y])
	})
	if ds.orders == nil {
		ds.orders = map[SortMode][]int{}
	}
	ds.orders[mode] = idx
	return idx
}

func (ds *Dataset) less(mode SortMode, a, b int) bool {
	ra, rb := &ds.Rows[a], &ds.Rows[b]
	da, db := &ds.Der[a], &ds.Der[b]
	if mode == SortAlphabetical || mode == SortFavorites || mode == SortRecents {
		return CompareKeys(da.titleKey, db.titleKey) < 0
	}
	if mode == SortMaker {
		if (da.Maker == "") != (db.Maker == "") {
			return da.Maker != "" // the unknown maker comes last
		}
		if c := CompareKeys(da.makerKey, db.makerKey); c != 0 {
			return c < 0
		}
		if da.Maker != db.Maker {
			return da.Maker < db.Maker // labels that collate alike stay separate groups
		}
		return CompareKeys(da.titleKey, db.titleKey) < 0
	}
	if mode == SortYear {
		ay, by := da.Year, db.Year // ReleaseYear, derived once at Ingest
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
