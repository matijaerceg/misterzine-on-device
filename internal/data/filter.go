package data

// Status is a row's install state on the card, computed by the scan package.
type Status uint8

const (
	StatusUnknown      Status = iota // not scanned yet
	StatusCurrent                    // on the card at the row's shipped date or newer
	StatusOutdated                   // on the card but older than the row's shipped date
	StatusFoundUndated               // on the card, build date unknown (undated rbf)
	StatusNotFound                   // not found on the card
)

// Found reports whether the status counts as "on the card".
func (s Status) Found() bool {
	return s == StatusCurrent || s == StatusOutdated || s == StatusFoundUndated
}

// Install filter values.
const (
	InstallAll     = "all"
	InstallFound   = "found"
	InstallMissing = "missing"
)

// Filters follow the site's all-checked model: a value listed in an Off set
// is hidden, so a value the data gains later defaults to visible.
type Filters struct {
	BaseOff  map[string]bool `json:"base_off,omitempty"`
	SrcOff   map[string]bool `json:"src_off,omitempty"`
	RotOff   map[string]bool `json:"rot_off,omitempty"`   // "h", "v", ""
	PlrOff   map[string]bool `json:"plr_off,omitempty"`   // raw plr, "" = unknown
	GenreOff map[string]bool `json:"genre_off,omitempty"` // raw genre, "" = no genre
	Install  string          `json:"install,omitempty"`   // InstallAll (default), InstallFound, InstallMissing
	FavOnly  bool            `json:"fav_only,omitempty"`
}

// Active reports whether any narrowing is in effect.
func (f *Filters) Active() bool {
	if f == nil {
		return false
	}
	return len(f.BaseOff) > 0 || len(f.SrcOff) > 0 || len(f.RotOff) > 0 || len(f.PlrOff) > 0 ||
		len(f.GenreOff) > 0 || (f.Install != "" && f.Install != InstallAll) || f.FavOnly
}

// Pass reports whether one row survives the filters.
func (f *Filters) Pass(r *Row, d *Derived, st Status, fav bool) bool {
	if f == nil {
		return true
	}
	if f.BaseOff[r.Base] || f.SrcOff[r.Src] || f.RotOff[d.RotGroup] || f.PlrOff[r.Plr] || f.GenreOff[r.Genre] {
		return false
	}
	switch f.Install {
	case InstallFound:
		if !st.Found() {
			return false
		}
	case InstallMissing:
		if st != StatusNotFound {
			return false
		}
	}
	if f.FavOnly && !fav {
		return false
	}
	return true
}

// Apply narrows an order to the rows that pass. status and fav may be nil.
func Apply(ds *Dataset, order []int, f *Filters, status func(i int) Status, fav func(k string) bool) []int {
	if !f.Active() {
		out := make([]int, len(order))
		copy(out, order)
		return out
	}
	out := make([]int, 0, len(order))
	for _, i := range order {
		st := StatusUnknown
		if status != nil {
			st = status(i)
		}
		isFav := false
		if fav != nil {
			isFav = fav(ds.Rows[i].K)
		}
		if f.Pass(&ds.Rows[i], &ds.Der[i], st, isFav) {
			out = append(out, i)
		}
	}
	return out
}
