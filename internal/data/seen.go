package data

import (
	"time"
)

// SeenHold: a launch within this window of the last one is the same visit,
// so the baseline holds still instead of advancing on every quick return.
const SeenHold = 60 * time.Minute

// SeenRecord is the persisted state, the site's mz-seen record: t is this
// visit's clock, bt the baseline visit's clock, base the k -> updated map
// as of the baseline visit, cur the map as of the rows now on screen.
type SeenRecord struct {
	T    string            `json:"t"`
	BT   string            `json:"bt"`
	Base map[string]string `json:"base"`
	Cur  map[string]string `json:"cur"`
}

// Seen is the live last-look state for one run.
type Seen struct {
	BaseRows map[string]string // nil = first visit, no marker
	BaseTime string            // ISO clock of the baseline visit
	Horizon  string            // newest Updated stamp the baseline holds
	State    SeenRecord        // what to persist
}

// rowMap leaves local rows out: they carry no release stamp and a file the
// scan finds is not news from the catalogue.
func rowMap(rows []Row) map[string]string {
	m := make(map[string]string, len(rows))
	for i := range rows {
		if rows[i].K != "" && !rows[i].IsLocal() {
			m[rows[i].K] = rows[i].Updated
		}
	}
	return m
}

// InitSeen mirrors initSeen(rows): a stored record younger than SeenHold is
// the same visit (keep its baseline), an older one promotes its cur snapshot
// to the new baseline. An empty baseline reads as none. The returned State
// must be persisted by the caller. Pass a zero clock when the device clock is
// not trusted; an existing baseline is kept. Without one, the previous snapshot
// is still useful for comparison even though the visit age is unknown.
func InitSeen(stored *SeenRecord, rows []Row, now time.Time, clockTrusted bool) *Seen {
	s := &Seen{}
	if stored != nil {
		// an untrusted clock cannot tell a quick return from a real one, so
		// the baseline holds still (never advances wrongly)
		sameVisit := !clockTrusted
		if t, ok := ParseMetaTime(stored.T); ok && clockTrusted {
			sameVisit = now.Sub(t) < SeenHold
		}
		if sameVisit && (clockTrusted || len(stored.Base) > 0) {
			s.BaseRows, s.BaseTime = stored.Base, stored.BT
		} else {
			s.BaseRows, s.BaseTime = stored.Cur, stored.T
		}
		if len(s.BaseRows) == 0 {
			s.BaseRows = nil
		}
		for _, u := range s.BaseRows {
			if u > s.Horizon {
				s.Horizon = u
			}
		}
	}
	t := ""
	if clockTrusted {
		t = now.UTC().Format(time.RFC3339)
	} else if stored != nil {
		t = stored.T
	}
	s.State = SeenRecord{T: t, BT: s.BaseTime, Base: s.BaseRows, Cur: rowMap(rows)}
	return s
}

// Bank mirrors bankSeen(rows): rows that land mid-visit are banked as seen
// for NEXT time but kept out of this visit's baseline.
func (s *Seen) Bank(rows []Row) {
	s.State.Cur = rowMap(rows)
}

// Unseen mirrors isUnseen(d).
func (s *Seen) Unseen(r *Row) bool {
	if s == nil || s.BaseRows == nil || r.K == "" || r.IsLocal() {
		return false
	}
	base, ok := s.BaseRows[r.K]
	return !ok || base != r.Updated
}

// MarkerOn mirrors visitMarkerOn(): only under the updated-desc sort.
func (s *Seen) MarkerOn(mode SortMode) bool {
	return s != nil && s.BaseRows != nil && mode == SortUpdated
}

// SplitAt is the position in order of the LAST unseen row dated on or after
// the baseline's horizon, -1 for none or when the marker is off. The line
// after it is a timeline marker: everything above it shipped since the last
// look. Under the updated-desc sort those rows are a prefix, so the scan
// stops where the dates pass the horizon. A row with an older date that is
// nonetheless unseen (a core catalogued late, a renamed key, a corrected
// stamp) keeps its own mark but never drags the line down to it. Same-day
// rows interleave seen with unseen (the stamps are day-granular), so the
// line lands approximately on busy days; that's accepted, it's a hint and
// not an audit.
func (s *Seen) SplitAt(ds *Dataset, order []int, mode SortMode) int {
	if !s.MarkerOn(mode) {
		return -1
	}
	last := -1
	for i, idx := range order {
		r := &ds.Rows[idx]
		if r.Updated < s.Horizon {
			break
		}
		if s.Unseen(r) {
			last = i
		}
	}
	return last
}

// AnyUnseen reports whether any row of order, however deep, is unseen.
func (s *Seen) AnyUnseen(ds *Dataset, order []int) bool {
	for _, idx := range order {
		if s.Unseen(&ds.Rows[idx]) {
			return true
		}
	}
	return false
}

// VisitAgo mirrors visitAgo(iso): under a day the minute/hour ladder, from a
// day up the day/month/year ladder. Empty when the clock cannot be trusted.
func VisitAgo(now time.Time, iso string, clockTrusted bool) string {
	if !clockTrusted {
		return ""
	}
	t, ok := ParseMetaTime(iso)
	if !ok {
		return ""
	}
	if now.Sub(t) < 24*time.Hour {
		return RelUpdated(now, t)
	}
	return RelAgeAt(now, t)
}

// Label is the marker text: "your last look, 2 days ago".
func (s *Seen) Label(now time.Time, clockTrusted bool) string {
	a := VisitAgo(now, s.BaseTime, clockTrusted)
	if a == "" {
		return "your last look"
	}
	return "your last look, " + a
}
