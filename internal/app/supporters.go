package app

import (
	_ "embed"
	"encoding/json"
	"strings"
)

// Supporter is one Patreon member as the site's supporters.json lists
// them: a display name and the months the support ran (Until is empty
// while it is current).
type Supporter struct {
	Name  string `json:"name"`
	Since string `json:"since"`
	Until string `json:"until,omitempty"`
}

// Supporters is https://misterzine.fyi/supporters.json: who supports
// MisterZine on Patreon now and who has in the past, and the early
// adopters who tested it before it was ready. The site's update workflow
// keeps the Patreon lists in step and passes the adopters through from
// its hand-kept source; the app fetches the file alongside the catalogue
// check and caches it, and ships this snapshot for a device that has
// never been online. A name added on the site reaches every online
// device on its next check, without a release.
type Supporters struct {
	Updated       string      `json:"updated"`
	Current       []Supporter `json:"current"`
	Past          []Supporter `json:"past"`
	EarlyAdopters []string    `json:"early_adopters"`
}

// supportersSnapshot is the site's file as of the build. Refresh it at
// release prep: curl -s https://misterzine.fyi/supporters.json.
//
//go:embed supporters.json
var supportersSnapshot []byte

// DecodeSupporters parses supporters.json. Names are trimmed and blank
// ones dropped, so a damaged file cannot put an empty row on the page.
func DecodeSupporters(b []byte) (Supporters, error) {
	var s Supporters
	if err := json.Unmarshal(b, &s); err != nil {
		return Supporters{}, err
	}
	clean := func(in []Supporter) []Supporter {
		out := in[:0]
		for _, p := range in {
			p.Name = strings.TrimSpace(p.Name)
			if p.Name != "" {
				out = append(out, p)
			}
		}
		return out
	}
	s.Current = clean(s.Current)
	s.Past = clean(s.Past)
	names := s.EarlyAdopters[:0]
	for _, n := range s.EarlyAdopters {
		if n = strings.TrimSpace(n); n != "" {
			names = append(names, n)
		}
	}
	s.EarlyAdopters = names
	return s, nil
}

// defaultSupporters is the embedded snapshot, parsed once.
func defaultSupporters() Supporters {
	s, _ := DecodeSupporters(supportersSnapshot)
	return s
}

// SetSupporters installs a newer supporters list (from the cache at
// startup, or freshly fetched). A file without the early adopters (an
// older site, or a cache from before the app read them) keeps the
// snapshot's, so the group never empties. The Credits page rebuilds if
// it is open.
func (a *App) SetSupporters(s Supporters) {
	if len(s.EarlyAdopters) == 0 {
		s.EarlyAdopters = a.supporters.EarlyAdopters
	}
	a.supporters = s
	if a.screen == ScreenCredits {
		cur := a.panel.cursor
		a.buildPanel()
		if cur < len(a.panel.entries) {
			a.panel.cursor = cur
		}
	}
	a.all = true
}
