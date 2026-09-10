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
)

func (m SortMode) String() string {
	switch m {
	case SortDebut:
		return "Debut"
	case SortAlphabetical:
		return "Alphabetical"
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
	if mode == SortAlphabetical {
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
