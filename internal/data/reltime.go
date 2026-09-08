package data

import (
	"fmt"
	"math"
	"time"
)

// ParseISODate parses "YYYY-MM-DD" as UTC midnight, like JS Date.parse.
func ParseISODate(iso string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// ParseMetaTime parses meta.json's "2026-09-08T00:59Z".
func ParseMetaTime(s string) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02T15:04Z", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func plural(n int, unit string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s ago", n, unit)
	}
	return fmt.Sprintf("%d %ss ago", n, unit)
}

// RelAge mirrors the site's relAge(iso) for whole-day facts: today,
// yesterday, "N days ago" up to 30, then months (rounded, days/30.44) below
// 12, then years (rounded, days/365.25). Rounding, not flooring, so an exact
// anniversary reads as the whole unit.
func RelAge(now time.Time, iso string) string {
	t, ok := ParseISODate(iso)
	if !ok {
		return ""
	}
	return RelAgeAt(now, t)
}

// RelAgeAt is RelAge for a full timestamp (the site passes ISO clock strings
// through the same ladder; whole days are floored from the exact moment).
func RelAgeAt(now, t time.Time) string {
	days := int(math.Floor(now.Sub(t).Hours() / 24))
	if days < 0 {
		days = 0
	}
	switch {
	case days == 0:
		return "today"
	case days == 1:
		return "yesterday"
	case days <= 30:
		return fmt.Sprintf("%d days ago", days)
	}
	mon := int(math.Round(float64(days) / 30.44))
	if mon < 12 {
		return plural(mon, "month")
	}
	yr := int(math.Round(float64(days) / 365.25))
	return plural(yr, "year")
}

// YearAge mirrors yearAge(y): "45 years old", "1 year old", "" under a year.
func YearAge(now time.Time, year string) string {
	var y int
	if _, err := fmt.Sscanf(year, "%d", &y); err != nil || y <= 0 {
		return ""
	}
	n := now.Year() - y
	if n < 1 {
		return ""
	}
	if n == 1 {
		return "1 year old"
	}
	return fmt.Sprintf("%d years old", n)
}

// RelUpdated mirrors relUpdated(dt) for the clock-grained data stamp: minutes
// and hours, then a short absolute date past a day.
func RelUpdated(now, t time.Time) string {
	min := int(math.Round(now.Sub(t).Minutes()))
	switch {
	case min < 1:
		return "just now"
	case min < 60:
		return plural(min, "minute")
	}
	hr := int(math.Round(float64(min) / 60))
	if hr < 24 {
		return plural(hr, "hour")
	}
	return t.Format("Jan 2, 2006")
}
