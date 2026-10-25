package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// The Details line under Source says whether Update All brings the game's
// source to this card. Before a card scan only the site's half is known;
// with a downloader.ini read, a database the card has is fetched already,
// an update_all one the card lacks is a toggle to turn on, and any other
// needs its section written. Gated and local rows get no line.
func TestUpdateAllDetailsLine(t *testing.T) {
	rows := []data.Row{
		{K: "dist", Title: "Dist", Base: "Arcade", Src: "distribution_mister", Updated: "2026-09-01"},
		{K: "co", Title: "Coin-Op Public", Base: "Arcade", Src: "coinop", Updated: "2026-09-01"},
		{K: "meat", Title: "Meat", Base: "Arcade", Src: "meathax", Updated: "2026-09-01"},
		{K: "jt", Title: "Jotego Beta", Base: "Arcade", Src: "jtbindb", Beta: true, Gate: "jtbeta", Updated: "2026-09-01"},
		{K: "loc", Title: "Local", Base: "Arcade", Src: data.SrcLocal, Updated: "2026-09-01"},
	}
	a := New(Config{PhysW: 320, PhysH: 240, Status: func(int) data.Status { return data.StatusFoundUndated }}, data.Ingest(rows, "h", time.Now()), nil)
	line := func(k string) string {
		i := a.ds.Index(k)
		if i < 0 {
			t.Fatalf("%s not indexed", k)
		}
		for _, l := range a.detailLines(&a.ds.Rows[i], &a.ds.Der[i], i) {
			if strings.HasPrefix(l.text, "Update All:") {
				return l.text
			}
		}
		return ""
	}
	check := func(stage string, want map[string]string) {
		t.Helper()
		for k, w := range want {
			if got := line(k); got != w {
				t.Errorf("%s, %s:\n got %q\nwant %q", stage, k, got, w)
			}
		}
	}
	siteOnly := map[string]string{
		"dist": "Update All: includes MiSTer Distribution",
		"co":   "Update All: includes Coin-Op Collection",
		"meat": "Update All: does not include Meathax (MeatCores); it needs a [meathax/meatcores] section in downloader.ini",
		"jt":   "",
		"loc":  "",
	}
	check("before a scan", siteOnly)
	a.SetHiddenSources(nil, false)
	check("no downloader.ini", siteOnly)
	a.SetHiddenSources(map[string]bool{"coinop": true, "meathax": true}, true)
	check("downloader.ini read", map[string]string{
		"dist": "Update All: this card fetches MiSTer Distribution",
		"co":   "Update All: Coin-Op Collection is switched off on this card; turn it on in Update All's settings",
		"meat": "Update All: does not include Meathax (MeatCores); add a [meathax/meatcores] section to downloader.ini",
		"jt":   "",
		"loc":  "",
	})
	// the line reads the card, not the Sources option: Installed only off
	// says the same
	if a.InstalledOnly() {
		t.Fatal("Installed only is on by default")
	}
}
