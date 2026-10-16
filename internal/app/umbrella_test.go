package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// umbrellaRows are a game on two cores (bub-a, bub), a game on one and a
// standin for bub-a's setname.
func umbrellaRows() []data.Row {
	cat := []data.Row{
		{K: "bub-a", G: "bub", Title: "Bubble (Core A)", Base: "Arcade", Core: "corea", SN: "buba", Updated: "2026-05-01", Date: "2025-01-01"},
		{K: "solo", Title: "Solo", Base: "Arcade", Core: "solo", SN: "solo", Updated: "2026-05-15", Date: "2025-06-01"},
		{K: "bub", G: "bub", Title: "Bubble (Core B)", Base: "Arcade", Core: "coreb", SN: "bub", Updated: "2026-06-01", Date: "2024-01-01"},
	}
	local := []data.Row{{K: data.LocalKey("buba"), Title: "Bubble (card)", Base: "Arcade", Src: data.SrcLocal, SN: "buba", Core: "corec", Standin: true}}
	return data.MergeLocal(cat, local)
}

func umbrellaApp(stored *data.SeenRecord) *App {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	return New(Config{PhysW: 320, PhysH: 240, ClockTrusted: true, Now: func() time.Time { return now }},
		data.Ingest(umbrellaRows(), "h", now), stored)
}

func viewKeys(a *App) []string {
	var out []string
	for pos := range a.view {
		out = append(out, a.keyAt(pos))
	}
	return out
}

func TestUmbrellaExpandCollapseWithA(t *testing.T) {
	a := umbrellaApp(nil)
	a.SetSort(data.SortUpdated)
	if got := strings.Join(viewKeys(a), " "); got != "g:bub solo" {
		t.Fatalf("collapsed view %q", got)
	}
	if a.Visible() != 4 {
		t.Fatalf("Visible %d counts umbrellas, not its 4 rows", a.Visible())
	}
	a.cursor = 0
	if !strings.Contains(a.listHint(), "A Expand") {
		t.Fatalf("legend %q", a.listHint())
	}
	if row, _, _ := a.current(); row != nil {
		t.Fatal("an umbrella has a current row")
	}
	a.act(platform.KeyEnter)
	if a.screen != ScreenList || a.CursorKey() != "g:bub" {
		t.Fatalf("expand left screen %v cursor %q", a.screen, a.CursorKey())
	}
	if got := strings.Join(viewKeys(a), " "); got != "g:bub bub bub-a local:buba solo" {
		t.Fatalf("open view %q", got)
	}
	if !strings.Contains(a.listHint(), "A Collapse") {
		t.Fatalf("legend %q", a.listHint())
	}
	a.act(platform.KeyDown)
	if a.CursorKey() != "bub" {
		t.Fatalf("Down from the umbrella lands on %q", a.CursorKey())
	}
	a.act(platform.KeyEnter)
	if a.screen != ScreenDetails {
		t.Fatalf("A on a member opened %v, want Details", a.screen)
	}
	a.act(platform.KeyBack)
	a.cursor = 0
	a.act(platform.KeyEnter)
	if got := strings.Join(viewKeys(a), " "); got != "g:bub solo" || a.CursorKey() != "g:bub" {
		t.Fatalf("collapsed again %q cursor %q", got, a.CursorKey())
	}
}

func TestUmbrellaKeySurvivesSortAndDataSwap(t *testing.T) {
	a := umbrellaApp(nil)
	a.SetSort(data.SortUpdated)
	a.MoveToKey("g:bub")
	a.SetSort(data.SortDebut)
	if a.CursorKey() != "g:bub" {
		t.Fatalf("after sort cursor %q", a.CursorKey())
	}
	a.SetData(data.Ingest(umbrellaRows(), "h2", time.Time{}), nil)
	if a.CursorKey() != "g:bub" {
		t.Fatalf("after data swap cursor %q", a.CursorKey())
	}
	// a member under the cursor reopens its umbrella after the swap closed it
	a.MoveToKey("bub-a")
	if a.CursorKey() != "bub-a" || !a.expanded["bub"] {
		t.Fatalf("member move: cursor %q expanded %v", a.CursorKey(), a.expanded)
	}
	a.SetData(data.Ingest(umbrellaRows(), "h3", time.Time{}), nil)
	if a.CursorKey() != "bub-a" {
		t.Fatalf("member after swap: cursor %q", a.CursorKey())
	}
}

func TestUmbrellaNotFoldedUnderCoreFavoritesRecents(t *testing.T) {
	a := umbrellaApp(nil)
	a.cfg.Favorites = map[string]bool{"bub": true, "bub-a": true}
	a.cfg.RecentLaunches = []data.Recent{{K: "bub", At: "2026-10-01T10:00:00Z"}, {K: "bub-a", At: "2026-09-01T10:00:00Z"}}
	for _, m := range []data.SortMode{data.SortCore, data.SortFavorites, data.SortRecents} {
		a.mode = m
		a.rebuild()
		if len(a.groups) != 0 {
			t.Errorf("%v folds: %v", m, viewKeys(a))
		}
	}
}

func TestSearchPromotesLoneMember(t *testing.T) {
	a := umbrellaApp(nil)
	a.SetSort(data.SortAlphabetical)
	a.setSearch("core b")
	if got := strings.Join(viewKeys(a), " "); got != "bub" || a.shown != 1 {
		t.Fatalf("search view %q shown %d", got, a.shown)
	}
	// widening the search folds the group again, opened for the member the
	// cursor stays on
	a.setSearch("bubble")
	if got := strings.Join(viewKeys(a), " "); got != "g:bub bub-a local:buba bub" || a.CursorKey() != "bub" {
		t.Fatalf("search view %q cursor %q", got, a.CursorKey())
	}
}

func TestUmbrellaRollsUpTheSinceVisitDot(t *testing.T) {
	seenAll := map[string]string{"bub-a": "2026-05-01", "bub": "2026-06-01", "solo": "2026-05-15"}
	old := &data.SeenRecord{T: "2026-01-01T00:00Z", Cur: seenAll}
	a := umbrellaApp(old)
	a.SetSort(data.SortUpdated)
	if u := a.umbrellaAt(0); u == nil || a.umbrellaUnseen(u) {
		t.Fatal("an umbrella of seen members shows the dot")
	}
	rebuilt := map[string]string{"bub-a": "2026-04-01", "bub": "2026-06-01", "solo": "2026-05-15"}
	a = umbrellaApp(&data.SeenRecord{T: "2026-01-01T00:00Z", Cur: rebuilt})
	a.SetSort(data.SortUpdated)
	u := a.umbrellaAt(0)
	if u == nil || !a.umbrellaUnseen(u) {
		t.Fatal("a rebuilt member's dot does not roll up")
	}
	a.act(platform.KeyEnter)
	if a.umbrellaUnseen(a.umbrellaAt(0)) {
		t.Fatal("an open umbrella keeps the dot its members show")
	}
}

func TestUmbrellaTakesNoStar(t *testing.T) {
	a := umbrellaApp(nil)
	a.SetSort(data.SortUpdated)
	a.MoveToKey("g:bub")
	a.toggleFavorite()
	if len(a.cfg.Favorites) != 0 {
		t.Fatalf("favorites %v", a.cfg.Favorites)
	}
	a.all = true
	a.Paint() // the row and pane draw the umbrella from its members
}

func TestMemberLinesSitUnderTheirOpenUmbrella(t *testing.T) {
	a := umbrellaApp(nil)
	a.SetSort(data.SortUpdated)
	for pos := range a.view {
		if a.memberAt(pos) {
			t.Fatalf("closed umbrella: line %d (%s) counts as a member", pos, a.keyAt(pos))
		}
	}
	a.cursor = 0
	a.act(platform.KeyEnter)
	want := map[string]bool{"bub": true, "bub-a": true, "local:buba": true}
	for pos := range a.view {
		if got := a.memberAt(pos); got != want[a.keyAt(pos)] {
			t.Fatalf("open umbrella: line %d (%s) member=%v", pos, a.keyAt(pos), got)
		}
	}
}
