package data

import (
	"fmt"
	"strings"
)

// DiffNews mirrors diffNews(d): what a swap just brought in, phrased for the
// notice, or "" when nothing a visitor would care about changed. fresh = keys
// absent before, built = keys whose updated stamp moved.
func DiffNews(old *Dataset, rows []Row) string {
	var fresh, built []string
	for i := range rows {
		r := &rows[i]
		if r.K == "" {
			continue
		}
		j := old.Index(r.K)
		if j < 0 {
			fresh = append(fresh, r.Title)
		} else if old.Rows[j].Updated != r.Updated {
			built = append(built, r.Title)
		}
	}
	if len(fresh) == 0 && len(built) == 0 {
		return ""
	}
	if len(fresh) > 0 && len(built) > 0 {
		return fmt.Sprintf("%d new, %d updated", len(fresh), len(built))
	}
	names, noun := fresh, " new: "
	if len(fresh) == 0 {
		names, noun = built, " updated: "
	}
	show := names
	if len(show) > 3 {
		show = show[:3]
	}
	rest := len(names) - len(show)
	s := fmt.Sprintf("%d%s%s", len(names), noun, strings.Join(show, ", "))
	if rest > 0 {
		s += fmt.Sprintf(" +%d more", rest)
	}
	return s
}
