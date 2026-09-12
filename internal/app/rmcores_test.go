package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// An rmCores row carries the site's name and short chip and explains the rm
// build under Source in Details; other sources get no such line.
func TestRmCoresNamesAndDetailsLine(t *testing.T) {
	rows := []data.Row{
		{K: "rm-night-slashers", Title: "Night Slashers (rmCores)", Base: "Arcade", Src: "rmcores", Core: "rmNightSlashers", MRA: "_Arcade/_rmCores/rm Night Slashers.mra"},
		{K: "night-slashers", Title: "Night Slashers", Base: "Arcade", Src: "distribution_mister", Core: "NightSlashers", MRA: "_Arcade/Night Slashers.mra"},
	}
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(rows, "", time.Now()), nil)
	if data.SrcFull("rmcores") != "rmCores (rmonic79)" || data.SrcShort("rmcores") != "rmCores" {
		t.Fatalf("names: %q %q", data.SrcFull("rmcores"), data.SrcShort("rmcores"))
	}
	if a.ds.Der[0].SrcShort != "rmCores" {
		t.Fatalf("row chip: %q", a.ds.Der[0].SrcShort)
	}
	count := func(i int) (int, bool) {
		n, src := 0, false
		for _, line := range a.detailLines(&a.ds.Rows[i], &a.ds.Der[i], i) {
			if strings.HasPrefix(line.text, "rm build:") {
				n++
			}
			if strings.HasPrefix(line.text, "Source:") && strings.Contains(line.text, "rmCores (rmonic79)") {
				src = true
			}
		}
		return n, src
	}
	if n, src := count(0); n != 1 || !src {
		t.Fatalf("rm row: %d rm lines, source named %v", n, src)
	}
	if n, src := count(1); n != 0 || src {
		t.Fatalf("Distribution row: %d rm lines, rm source %v", n, src)
	}
}
