package app

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

// cardApp is an app whose card statuses come from st, by row key; a key
// missing from st reads as not scanned yet.
func cardApp(rows []data.Row, st map[string]data.Status) *App {
	ds := data.Ingest(rows, "", time.Now())
	var a *App
	a = New(Config{PhysW: 320, PhysH: 240, Status: func(i int) data.Status {
		if a != nil { // the rows on show, after a SetData too
			return st[a.ds.Rows[i].K]
		}
		return st[ds.Rows[i].K]
	}}, ds, nil)
	return a
}

func arcadeRows(keys ...string) []data.Row {
	rows := make([]data.Row, len(keys))
	for i, k := range keys {
		rows[i] = data.Row{K: k, Title: "Game " + k, Base: "Arcade", Src: "distribution_mister"}
	}
	return rows
}

func hasChangeLine(lines []string) bool {
	for _, l := range lines {
		if strings.Contains(l, "up to date:") && !strings.HasPrefix(l, "Up to date:") {
			return true
		}
	}
	return false
}

// Rescan card says how many releases became up to date, and how: an older
// or undated copy replaced counts as updated, one the card did not have as
// installed. Releases no longer up to date get a line of their own; a
// status unknown on either side, and releases outside the enabled
// catalogue, count nowhere.
func TestRescanCountsReleasesNowUpToDate(t *testing.T) {
	rows := arcadeRows("older", "likely", "undated", "missing", "same", "gone", "behind", "partly", "unknown", "lost")
	rows = append(rows, data.Row{K: "console", Title: "Console game", Base: "Console", Src: "distribution_mister"})
	st := map[string]data.Status{
		"older": data.StatusOutdated, "likely": data.StatusLikelyOutdated, "undated": data.StatusFoundUndated,
		"missing": data.StatusNotFound, "same": data.StatusCurrent, "gone": data.StatusCurrent,
		"behind": data.StatusCurrent, "partly": data.StatusNotFound, "lost": data.StatusCurrent,
		"console": data.StatusNotFound,
	}
	a := cardApp(rows, st)
	a.OpenScan()
	for k, s := range map[string]data.Status{
		"older": data.StatusCurrent, "likely": data.StatusCurrent, "undated": data.StatusCurrent,
		"missing": data.StatusCurrent, "gone": data.StatusNotFound, "behind": data.StatusOutdated,
		"partly": data.StatusOutdated, "unknown": data.StatusCurrent, "console": data.StatusCurrent,
	} {
		st[k] = s
	}
	delete(st, "lost") // its status unknown now: says nothing about the card
	a.FinishScan("", true)
	lines := a.scanLines(54, 20)
	if a.scanWatch == nil || a.scanWatch.change != (cardChange{updated: 3, installed: 1, lost: 2}) {
		t.Fatalf("change %+v", a.scanWatch)
	}
	want := []string{"Card scan complete", "Newly up to date: 4 (3 updated, 1 installed)", "No longer up to date: 2", "", "Up to date: 6"}
	if !reflect.DeepEqual(lines[:5], want) {
		t.Fatalf("lines %q\nwant %q", lines[:5], want)
	}
	a.Paint()
}

// A scan that finds the card as it was says so plainly.
func TestRescanWithNoChangeSaysSo(t *testing.T) {
	st := map[string]data.Status{"a": data.StatusCurrent, "b": data.StatusOutdated, "c": data.StatusNotFound}
	a := cardApp(arcadeRows("a", "b", "c"), st)
	a.OpenScan()
	a.FinishScan("", true)
	got := a.scanLines(54, 20)[:3]
	want := []string{"Card scan complete", "Newly up to date: 0", ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
}

// The first scan has nothing to compare with: every release it finds on
// the card was unknown before, and none of them reads as just installed,
// on the Rescan card screen or after an update run.
func TestRescanFirstScanClaimsNothing(t *testing.T) {
	st := map[string]data.Status{}
	a := cardApp(arcadeRows("a", "b", "c"), st)
	a.OpenScan()
	for _, k := range []string{"a", "b", "c"} {
		st[k] = data.StatusCurrent
	}
	a.FinishScan("", true)
	if a.scanWatch != nil || hasChangeLine(a.scanLines(54, 20)) {
		t.Fatalf("a first scan reported changes: %q", a.scanLines(54, 20))
	}
	b := cardApp(arcadeRows("a", "b", "c"), map[string]data.Status{})
	b.SetUpdate(updater.State{ID: "run", Status: "running"}, true)
	b.SetUpdate(updater.State{ID: "run", Status: "completed"}, true)
	b.RescanAfterUpdate()
	b.CardScanned(true)
	if b.updateView.card != nil {
		t.Fatal("an update's rescan with nothing known before reported changes")
	}
}

// Rows the rescan adds or takes away count nowhere: a release new to the
// list was never known off the card, and one that left the list is not a
// card change. The rows both sides know still count.
func TestRescanRowsComingAndGoing(t *testing.T) {
	st := map[string]data.Status{"left": data.StatusCurrent, "b": data.StatusOutdated, "c": data.StatusNotFound}
	a := cardApp(arcadeRows("left", "b", "c"), st)
	a.OpenScan()
	st["b"], st["c"], st["arrived"], st["also"] = data.StatusCurrent, data.StatusCurrent, data.StatusCurrent, data.StatusNotFound
	delete(st, "left")
	a.SetData(data.Ingest(arcadeRows("b", "c", "arrived", "also"), "", time.Now()), nil)
	a.FinishScan("", true)
	if a.scanWatch == nil || a.scanWatch.change != (cardChange{updated: 1, installed: 1}) {
		t.Fatalf("change %+v", a.scanWatch)
	}
}

// A scan that failed before any status arrived shows its problem alone.
func TestRescanFailureReportsNoChange(t *testing.T) {
	a := cardApp(arcadeRows("a"), map[string]data.Status{"a": data.StatusOutdated})
	a.OpenScan()
	a.FinishScan("Card scan failed", false)
	if a.scanWatch != nil || hasChangeLine(a.scanLines(54, 20)) {
		t.Fatalf("a failed scan reported changes: %q", a.scanLines(54, 20))
	}
}

// On the widest safe zone the change lines take only the rows the result
// leaves: a scan problem and its pointer to the log stay on screen, and the
// number goes first, then its breakdown, then what stopped being current.
func TestRescanReportLeavesProblemsOnScreen(t *testing.T) {
	keys := []string{"n", "c"}
	st := map[string]data.Status{"n": data.StatusNotFound, "c": data.StatusCurrent}
	for i := range 10 {
		k := "o" + itoa(i)
		keys, st[k] = append(keys, k), data.StatusOutdated
	}
	a := cardApp(arcadeRows(keys...), st)
	a.OpenScan()
	for k, s := range st {
		st[k] = data.StatusCurrent
		if s == data.StatusCurrent {
			st[k] = data.StatusNotFound
		}
	}
	a.FinishScan("Card scan incomplete", true)
	// 44 columns and 12 rows is 320x240 at a 40 px safe zone, where the
	// result and its problem fill all 12; 13 and 14 leave one and two spare
	full := a.scanLines(44, 20)
	if full[1] != "Newly up to date: 11" || full[2] != "(10 updated, 1 installed)" || full[3] != "No longer up to date: 1" {
		t.Fatalf("with room: %q", full)
	}
	for fit, want := range map[int][]string{
		12: nil,
		13: {"Newly up to date: 11"},
		14: {"Newly up to date: 11", "(10 updated, 1 installed)"},
	} {
		lines := a.scanLines(44, fit)
		if textRows(lines, 44) > fit || lines[len(lines)-1] != "See the device log for details." {
			t.Fatalf("fit %d pushed the problem off: %q", fit, lines)
		}
		got := lines[1 : 1+len(want)]
		if len(want) == 0 {
			got, want = lines[1:2], []string{""}
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("fit %d: %q want %q", fit, got, want)
		}
	}
}

// The words: the number first, the breakdown in brackets when it fits the
// line, on a line of its own when it does not, "Checking card..." while
// the scan runs.
func TestRescanReportWording(t *testing.T) {
	for _, tc := range []struct {
		change cardChange
		cols   int
		want   []string
	}{
		{cardChange{updated: 9, installed: 3}, 54, []string{"Newly up to date: 12 (9 updated, 3 installed)"}},
		{cardChange{updated: 9, installed: 3}, 28, []string{"Newly up to date: 12", "(9 updated, 3 installed)"}},
		{cardChange{updated: 1}, 38, []string{"Newly up to date: 1 (updated)"}},
		{cardChange{installed: 2}, 38, []string{"Newly up to date: 2 (installed)"}},
		{cardChange{installed: 2}, 28, []string{"Newly up to date: 2", "(installed)"}},
		{cardChange{lost: 3}, 28, []string{"Newly up to date: 0", "No longer up to date: 3"}},
		{cardChange{}, 28, []string{"Newly up to date: 0"}},
	} {
		w := &cardWatch{done: true, change: tc.change}
		if got := w.lines(tc.cols, 5); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%+v at %d: %q want %q", tc.change, tc.cols, got, tc.want)
		}
	}
	if got := (&cardWatch{}).lines(28, 5); !reflect.DeepEqual(got, []string{"Checking card..."}) {
		t.Fatalf("running: %q", got)
	}
	if (*cardWatch)(nil).lines(28, 5) != nil || (&cardWatch{done: true}).lines(28, 0) != nil {
		t.Fatal("lines without a watch or room")
	}
}

// After an Update All run the update screen says what the rescan the run
// ends with changed: "Checking card..." until it has finished, then the
// counts, which later scans leave alone. A MisterZine-only run leaves the
// games alone and says nothing; neither does a rescan that failed, and a
// new run starts clean.
func TestUpdateRescanReport(t *testing.T) {
	st := map[string]data.Status{"a": data.StatusOutdated, "b": data.StatusCurrent}
	a := cardApp(arcadeRows("a", "b"), st)
	run := updater.State{ID: "run", Mode: updater.ModeAll, Status: "running"}
	a.SetUpdate(run, true)
	done := run
	done.Status = "completed"
	a.SetUpdate(done, true)
	a.RescanAfterUpdate()
	cols := a.sm.Cols(a.lay.Body.Dx() - 4)
	if got := a.updateView.card.lines(cols, 3); !reflect.DeepEqual(got, []string{"Checking card..."}) {
		t.Fatalf("before the scan finished: %q", got)
	}
	a.Paint()
	st["a"] = data.StatusCurrent
	a.CardScanned(true)
	if got := a.updateView.card.lines(cols, 3); !reflect.DeepEqual(got, []string{"Newly up to date: 1 (updated)"}) {
		t.Fatalf("after the scan: %q", got)
	}
	st["b"] = data.StatusNotFound
	a.CardScanned(true)
	a.SetUpdate(done, true) // Options -> Last update result
	if a.updateView.card.change != (cardChange{updated: 1}) {
		t.Fatalf("a later scan rewrote the run's report: %+v", a.updateView.card.change)
	}
	a.Paint()
	a.OpenUpdate(updater.ModeAll)
	if a.updateView.card != nil {
		t.Fatal("a new run kept the last run's report")
	}

	app := updater.State{ID: "app", Mode: updater.ModeApp, Status: "running"}
	a.SetUpdate(app, true)
	app.Status = "completed"
	a.SetUpdate(app, true)
	a.RescanAfterUpdate()
	if a.updateView.card != nil {
		t.Fatal("a MisterZine-only run watched the card")
	}
	free := updater.State{ID: "free", Mode: updater.ModeFree, Status: "running"}
	a.SetUpdate(free, true)
	free.Status = "completed"
	a.SetUpdate(free, true)
	a.RescanAfterUpdate()
	if a.updateView.card != nil {
		t.Fatal("the switch to the free version watched the card")
	}

	failed := updater.State{ID: "failed", Status: "running"}
	a.SetUpdate(failed, true)
	failed.Status = "failed"
	a.SetUpdate(failed, true)
	a.RescanAfterUpdate()
	a.CardScanned(false)
	if a.updateView.card != nil {
		t.Fatal("a rescan that delivered no statuses left a report")
	}
}
