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
	InstallFound   = "found"   // current, older or date unknown
	InstallCurrent = "current" // on the card at the shipped date or newer
	InstallOlder   = "older"   // on the card but behind the shipped date
	InstallUndated = "undated" // on the card, build date unknown
	InstallMissing = "missing" // not found on the card
)

// Filters follow the site's all-checked model: a value listed in an Off set
// is hidden, so a value the data gains later defaults to visible.
type Filters struct {
	ResOff        map[string]bool `json:"res_off,omitempty"` // raw resolution, "" = unknown
	BaseOff       map[string]bool `json:"base_off,omitempty"`
	SrcOff        map[string]bool `json:"src_off,omitempty"`
	RotOff        map[string]bool `json:"rot_off,omitempty"`   // "h", "v", ""
	PlrOff        map[string]bool `json:"plr_off,omitempty"`   // raw plr, "" = unknown
	GenreOff      map[string]bool `json:"genre_off,omitempty"` // raw genre, "" = no genre
	DirectionsOff map[string]bool `json:"directions_off,omitempty"`
	ButtonsOff    map[string]bool `json:"buttons_off,omitempty"`
	Install       string          `json:"install,omitempty"` // InstallAll (default) or one of the Install* values
	FavOnly       bool            `json:"fav_only,omitempty"`
	Since         bool            `json:"since,omitempty"` // only rows changed since the last look
}

// Active reports whether any narrowing is in effect.
func (f *Filters) Active() bool {
	if f == nil {
		return false
	}
	return len(f.BaseOff) > 0 || len(f.SrcOff) > 0 || len(f.RotOff) > 0 || len(f.PlrOff) > 0 ||
		len(f.ResOff) > 0 || len(f.GenreOff) > 0 || len(f.DirectionsOff) > 0 || len(f.ButtonsOff) > 0 || (f.Install != "" && f.Install != InstallAll) || f.FavOnly || f.Since
}

// Pass reports whether one row survives the filters.
func (f *Filters) Pass(r *Row, d *Derived, st Status, fav, unseen bool) bool {
	if f == nil {
		return true
	}
	if f.ResOff[r.Res] || f.BaseOff[r.Base] || f.SrcOff[r.Src] || f.RotOff[d.RotGroup] || f.PlrOff[r.Plr] || f.GenreOff[r.Genre] || f.DirectionsOff[d.Directions] || f.ButtonsOff[d.Buttons] {
		return false
	}
	switch f.Install {
	case InstallFound:
		if !st.Found() {
			return false
		}
	case InstallCurrent:
		if st != StatusCurrent {
			return false
		}
	case InstallOlder:
		if st != StatusOutdated {
			return false
		}
	case InstallUndated:
		if st != StatusFoundUndated {
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
	if f.Since && !unseen {
		return false
	}
	return true
}

// Apply narrows an order to the rows that pass. status, fav and unseen may be nil.
func Apply(ds *Dataset, order []int, f *Filters, status func(i int) Status, fav func(k string) bool, unseen func(i int) bool) []int {
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
		uns := false
		if unseen != nil {
			uns = unseen(i)
		}
		if f.Pass(&ds.Rows[i], &ds.Der[i], st, isFav, uns) {
			out = append(out, i)
		}
	}
	return out
}
