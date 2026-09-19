package data

import (
	"strconv"
	"strings"
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

// Since counts, over the whole catalogue rather than the filtered view, the
// rows added since the baseline visit and the rows whose build moved. include
// is the catalogue rule (hidden sources, non-arcade rows); nil takes every
// row. Nothing without a baseline.
func (s *Seen) Since(ds *Dataset, include func(*Row) bool) (added, updated int) {
	if s == nil || s.BaseRows == nil {
		return 0, 0
	}
	for i := range ds.Rows {
		r := &ds.Rows[i]
		if !s.Unseen(r) || include != nil && !include(r) {
			continue
		}
		if _, ok := s.BaseRows[r.K]; ok {
			updated++
		} else {
			added++
		}
	}
	return added, updated
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

// Status mirrors the site's status row, the first line of the list under
// every sort. The counts run over the whole catalogue, never the filtered
// view, and the copy never says "new": that word is for a MiSTer debut. The
// candidates run from the full sentence down to what fits a narrow line,
// keeping the visit's age as long as possible; the caller takes the first
// that fits.
func (s *Seen) Status(now time.Time, clockTrusted bool, added, updated int) []string {
	if s == nil || s.BaseRows == nil {
		return []string{"First visit. Next visit, changed rows get a green date.", "First visit. Changes show next time.", "First visit"}
	}
	var parts []string
	if added > 0 {
		parts = append(parts, strconv.Itoa(added)+" added")
	}
	if updated > 0 {
		parts = append(parts, strconv.Itoa(updated)+" updated")
	}
	var forms []string
	if len(parts) == 0 {
		forms = []string{"nothing added or updated since your last visit", "nothing since your last visit", "nothing added or updated", "nothing since visit"}
	} else {
		counts := strings.Join(parts, ", ")
		forms = []string{counts + " since your last visit", counts + " since visit", counts}
	}
	var out []string
	if ago := VisitAgo(now, s.BaseTime, clockTrusted); ago != "" {
		for _, f := range forms {
			out = append(out, f+", "+ago)
		}
	}
	return append(out, forms...)
}
