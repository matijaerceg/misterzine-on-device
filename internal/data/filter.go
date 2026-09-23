package data

import "strings"

// Status is a row's install state on the card, computed by the scan package.
type Status uint8

const (
	StatusUnknown      Status = iota // not scanned yet
	StatusCurrent                    // on the card at the row's shipped date or newer
	StatusOutdated                   // on the card but older than the row's shipped date
	StatusFoundUndated               // on the card, build date unknown (undated rbf)
	StatusNotFound                   // not found on the card
	// StatusLikelyOutdated is an undated rbf whose md5 differs from the
	// shipped build's: the same evidence update_all acts on, so "older" is
	// the working assumption even though the direction is unprovable.
	// Appended after StatusNotFound so the harness's status cycle is stable.
	StatusLikelyOutdated
)

// Found reports whether the status counts as "on the card".
func (s Status) Found() bool {
	return s == StatusCurrent || s == StatusOutdated || s == StatusFoundUndated || s == StatusLikelyOutdated
}

// Older reports whether the status means the card is behind the shipped build.
func (s Status) Older() bool {
	return s == StatusOutdated || s == StatusLikelyOutdated
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

// ROMState is what the ROM check knows about one row: nothing yet, all its
// versions clean, a problem with the version Start would launch, or one
// with another version only.
type ROMState uint8

const (
	ROMUnknown ROMState = iota
	ROMClean
	ROMLaunchIssue
	ROMOtherIssue
)

// ROM filter values.
const (
	ROMAll    = "all"
	ROMLaunch = "launch" // the version Start launches has a problem
	ROMAny    = "any"    // any version has one
	ROMNone   = "none"   // every version checked and clean
)

// Filters follow the site's all-checked model: a value listed in an Off set
// is hidden, so a value the data gains later defaults to visible.
type Filters struct {
	ArcadeOnly     bool            `json:"-"`                 // runtime catalogue preference
	HideDeprecated bool            `json:"-"`                 // runtime catalogue preference
	MatchRotation  string          `json:"-"`                 // runtime INI rule; includes system/unknown rows
	SrcHidden      map[string]bool `json:"-"`                 // runtime rule: sources whose Downloader database the card lacks
	ResOff         map[string]bool `json:"res_off,omitempty"` // raw resolution, "" = unknown
	BaseOff        map[string]bool `json:"base_off,omitempty"`
	BetaOff        map[string]bool `json:"beta_off,omitempty"` // arcade sub-type: "stable", "beta"
	SrcOff         map[string]bool `json:"src_off,omitempty"`
	RotOff         map[string]bool `json:"rot_off,omitempty"`   // "h", "v", ""
	PlrOff         map[string]bool `json:"plr_off,omitempty"`   // raw plr, "" = unknown
	GenreOff       map[string]bool `json:"genre_off,omitempty"` // raw genre, "" = no genre
	DirectionsOff  map[string]bool `json:"directions_off,omitempty"`
	ButtonsOff     map[string]bool `json:"buttons_off,omitempty"`
	YearOff        map[string]bool `json:"year_off,omitempty"` // original arcade release year; "" = unknown
	Install        string          `json:"install,omitempty"`  // InstallAll (default) or one of the Install* values
	FavOnly        bool            `json:"fav_only,omitempty"`
	Since          bool            `json:"since,omitempty"` // only rows changed since the last look
	ROM            string          `json:"rom,omitempty"`   // ROMAll (default) or one of the ROM* values
}

// Active reports whether any narrowing is in effect.
func (f *Filters) Active() bool {
	if f == nil {
		return false
	}
	return f.ArcadeOnly || f.HideDeprecated || f.MatchRotation != "" || len(f.SrcHidden) > 0 || len(f.YearOff) > 0 || len(f.BaseOff) > 0 || len(f.BetaOff) > 0 || len(f.SrcOff) > 0 || len(f.RotOff) > 0 || len(f.PlrOff) > 0 ||
		len(f.ResOff) > 0 || len(f.GenreOff) > 0 || len(f.DirectionsOff) > 0 || len(f.ButtonsOff) > 0 || (f.Install != "" && f.Install != InstallAll) || f.FavOnly || f.Since || f.ROMActive()
}

// ROMActive reports whether the ROM check narrows the list.
func (f *Filters) ROMActive() bool {
	return f != nil && f.ROM != "" && f.ROM != ROMAll
}

// Pass reports whether one row survives the filters.
func (f *Filters) Pass(r *Row, d *Derived, st Status, fav, unseen bool, rom ROMState) bool {
	return f.Why(r, d, st, fav, unseen, rom) == ""
}

// Why names the first rule that hides a row, "" when the row survives the
// filters; Pass is Why == "", so the diagnostic report can never disagree
// with the list.
func (f *Filters) Why(r *Row, d *Derived, st Status, fav, unseen bool, rom ROMState) string {
	if f == nil {
		return ""
	}
	if f.ArcadeOnly && !r.IsArcade() {
		return "not an arcade game (Options: arcade only)"
	}
	if f.MatchRotation != "" && d.RotGroup != f.MatchRotation {
		return "Filter by rotation (the screen shows " + rotWord(f.MatchRotation) + " games; this one is " + rotWord(d.RotGroup) + ")"
	}
	if f.HideDeprecated && r.Deprecated {
		return "a deprecated core (Options: deprecated hidden)"
	}
	if f.BaseOff[r.Base] {
		return "Filters -> Type (" + r.Base + " off)"
	}
	if f.SrcOff[r.Src] {
		return "Filters -> Source (" + r.Src + " off)"
	}
	if f.SrcHidden[r.Src] {
		return "Options -> Sources: installed only (" + r.Src + " is not in downloader.ini)"
	}
	if r.IsArcade() {
		switch {
		case f.BetaOff[BetaKind(r)]:
			return "Filters -> Type (" + BetaKind(r) + " off)"
		case f.YearOff[d.Year]:
			return "Filters -> Year (" + orUnknown(d.Year) + " off)"
		case f.ResOff[r.Res]:
			return "Filters -> Resolution (" + orUnknown(r.Res) + " off)"
		case f.RotOff[d.RotGroup]:
			return "Filters -> Rotation (" + orUnknown(map[string]string{"h": "horizontal", "v": "vertical"}[d.RotGroup]) + " off)"
		case f.PlrOff[r.Plr]:
			return "Filters -> Players (" + orUnknown(r.Plr) + " off)"
		case f.GenreOff[r.Genre]:
			return "Filters -> Genre (" + orUnknown(r.Genre) + " off)"
		case f.DirectionsOff[d.Directions]:
			return "Filters -> Controls (" + orUnknown(d.Directions) + " off)"
		case f.ButtonsOff[d.Buttons]:
			return "Filters -> Buttons (" + orUnknown(d.Buttons) + " off)"
		}
	}
	switch f.Install {
	case InstallFound:
		if !st.Found() {
			return "Filters: only games on the card"
		}
	case InstallCurrent:
		if st != StatusCurrent {
			return "Filters: only current builds"
		}
	case InstallOlder:
		if !st.Older() {
			return "Filters: only older builds"
		}
	case InstallUndated:
		if st != StatusFoundUndated {
			return "Filters: only builds of unknown date"
		}
	case InstallMissing:
		if st != StatusNotFound {
			return "Filters: only games not on the card"
		}
	}
	if f.FavOnly && !fav {
		return "Filters: favorites only"
	}
	if f.Since && !unseen {
		return "Filters: only rows changed since the last look"
	}
	switch f.ROM {
	case ROMLaunch:
		if rom != ROMLaunchIssue {
			return "Filters: only games whose version to launch has a ROM problem"
		}
	case ROMAny:
		if rom != ROMLaunchIssue && rom != ROMOtherIssue {
			return "Filters: only games with a ROM problem in any version"
		}
	case ROMNone:
		if rom != ROMClean {
			return "Filters: only games whose ROMs all checked clean"
		}
	}
	return ""
}

func rotWord(g string) string {
	switch g {
	case "h":
		return "horizontal"
	case "v":
		return "vertical"
	}
	return "of unknown rotation"
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

// BetaKind is an arcade row's sub-type for the Type filter: "beta" for a
// Patreon-gated core (alphas included), "stable" otherwise.
func BetaKind(r *Row) string {
	if r.Beta {
		return "beta"
	}
	return "stable"
}

// GateStage is a gated row's stage word for its chip: "alpha" when the
// source's filter term ends in alpha, "beta" otherwise.
func GateStage(r *Row) string {
	if strings.HasSuffix(strings.ToLower(r.Gate), "alpha") {
		return "alpha"
	}
	return "beta"
}

// GateLine explains what a gated core needs, per source. A row flagged beta
// without a gate term comes from an older export and was Jotego's; an
// unknown term gets the generic wording.
func GateLine(r *Row) string {
	stage := GateStage(r)
	switch g := strings.ToLower(r.Gate); {
	case g == "" || g == "jtbeta":
		return "Patreon beta: needs Jotego's jtbeta.zip"
	case strings.HasPrefix(g, "coinop-collection-"):
		return "Patreon " + stage + ": needs a Coin-Op licence key for this MiSTer, and their filter overridden in downloader.ini"
	}
	return "Patreon " + stage + ": early access, needs a key from the author"
}

// Apply narrows an order to the rows that pass. status, fav, unseen and rom may be nil.
func Apply(ds *Dataset, order []int, f *Filters, status func(i int) Status, fav func(k string) bool, unseen func(i int) bool, rom func(i int) ROMState) []int {
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
		rs := ROMUnknown
		if rom != nil && f.ROMActive() {
			rs = rom(i) // asked only when it decides something: it may read the card's answers
		}
		if f.Pass(&ds.Rows[i], &ds.Der[i], st, isFav, uns, rs) {
			out = append(out, i)
		}
	}
	return out
}
