package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

// noticeKeys drives an app the way a pad and a keyboard do, the clock
// moving past the bounce guard before each event.
type noticeKeys struct {
	a   *App
	now *time.Time
}

func (k noticeKeys) edge(key platform.Key, down bool) bool {
	*k.now = k.now.Add(50 * time.Millisecond)
	return k.a.Handle(platform.Event{Key: key, Pressed: down, At: *k.now})
}

func (k noticeKeys) tap(key platform.Key) bool {
	r := k.edge(key, true)
	return k.edge(key, false) || r
}

func (k noticeKeys) typed(s string) {
	for _, ch := range s {
		*k.now = k.now.Add(50 * time.Millisecond)
		k.a.Handle(platform.Event{Text: ch, Pressed: true, At: *k.now})
	}
}

// noticeApp is a list of twelve arcade games, two a month from January
// to June and two genres, in the build date view.
func noticeApp() (*App, noticeKeys) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	var rows []data.Row
	for i := range 12 {
		date := time.Date(2026, time.Month(1+i/2), 1+i, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		genre := "Shooter"
		if i%2 == 1 {
			genre = "Puzzle"
		}
		rows = append(rows, data.Row{K: fmt.Sprintf("r%02d", i), Title: fmt.Sprintf("Game %02d", i), Base: "Arcade", Genre: genre, Updated: date, Date: date})
	}
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return now }}, data.Ingest(rows, "test", now), nil)
	return a, noticeKeys{a, &now}
}

// Select+Y and then Y: the bar names the new view at once instead of
// keeping the layout's notice up for its two seconds.
func TestNoticeGivesWayToTheNewView(t *testing.T) {
	a, k := noticeApp()
	view := a.Sort()
	k.edge(platform.KeySelect, true)
	k.tap(platform.KeySpace)
	k.edge(platform.KeySelect, false)
	if a.notice != "Layout: split" || a.Sort() != view {
		t.Fatalf("Select+Y: notice %q, view %v", a.notice, a.Sort())
	}
	if !k.tap(platform.KeySpace) || a.Sort() == view || a.notice != "" {
		t.Fatalf("Y: view %v -> %v, notice %q", view, a.Sort(), a.notice)
	}
	if got, want := a.statusBar().view, statusViews[a.Sort()]; got != want {
		t.Fatalf("the bar reads %q, want %q", got, want)
	}
}

// Typing a search shows it at once, over any notice: the month a jump
// names, which still shows when the jump is made during a search, and
// the next letter, Backspace or B then takes it down.
func TestNoticeGivesWayToTheSearch(t *testing.T) {
	a, k := noticeApp()
	k.tap(platform.KeyPageDown)
	if a.notice != "May 2026" {
		t.Fatalf("jump: notice %q", a.notice)
	}
	k.typed("g")
	if a.query != "g" || a.notice != "" {
		t.Fatalf("typing: query %q, notice %q", a.query, a.notice)
	}
	for _, next := range []func(){
		func() { k.typed("a") },
		func() { k.tap(platform.KeyBackspace) },
		func() { k.tap(platform.KeyBack) },
	} {
		query := a.query
		k.tap(platform.KeyPageDown)
		if a.notice == "" {
			t.Fatalf("a jump during the search %q named no month", query)
		}
		next()
		if a.query == query || a.notice != "" {
			t.Fatalf("search %q -> %q, notice %q", query, a.query, a.notice)
		}
	}
	if a.query != "" || a.statusBar().total != 12 {
		t.Fatalf("B should have cleared the search: %q, bar %+v", a.query, a.statusBar())
	}
}

// A filter that changes the count shows the count at once. The list's
// count is not on Filters' bar, whose title stays under a notice while the
// filters change; X opening Filters and B going back to the list change
// the bar, and each shows the new one at once.
func TestNoticeGivesWayToAFilterChange(t *testing.T) {
	a, k := noticeApp()
	a.Notice("saved screenshot", 8*time.Second)
	k.tap(platform.KeyTab)
	if a.screen != ScreenFilter || a.notice != "" {
		t.Fatalf("X: screen %v, notice %q", a.screen, a.notice)
	}
	a.panel.sectionClosed = nil
	a.buildPanel()
	a.Notice("saved screenshot", 8*time.Second)
	for i, e := range a.panel.entries {
		if !e.header && e.kind == "genre" && e.value == "Puzzle" {
			a.panel.cursor = i
		}
	}
	k.tap(platform.KeyEnter)
	if len(a.view) != 6 || a.notice == "" {
		t.Fatalf("A on Puzzle: %d rows, notice %q; Filters' bar did not change", len(a.view), a.notice)
	}
	k.tap(platform.KeyBack)
	if a.screen != ScreenList || a.notice != "" {
		t.Fatalf("B: screen %v, notice %q", a.screen, a.notice)
	}
	if s := a.statusBar(); !s.narrow || s.shown != 6 || s.total != 12 {
		t.Fatalf("the bar reads %+v, want 6 of 12", s)
	}
}

// A newer notice replaces an older one, and a press that sets a notice
// keeps it even when it also changes the bar: unstarring a game in
// Favorites takes it off the count and says so.
func TestNewerNoticeReplacesOlder(t *testing.T) {
	a, k := noticeApp()
	k.edge(platform.KeySelect, true)
	for _, want := range []string{"Layout: split", "Layout: picture"} {
		k.tap(platform.KeySpace)
		if a.notice != want {
			t.Fatalf("Select+Y: notice %q, want %q", a.notice, want)
		}
	}
	k.tap(platform.KeyTab)
	if a.notice != "Art type: title" {
		t.Fatalf("Select+X: notice %q", a.notice)
	}
	k.edge(platform.KeySelect, false)
	a.cfg.Favorites = map[string]bool{"r00": true, "r01": true}
	a.SetSort(data.SortFavorites)
	shown := len(a.view)
	k.edge(platform.KeySelect, true)
	k.tap(platform.KeyEnter)
	k.edge(platform.KeySelect, false)
	if len(a.view) != shown-1 || a.notice != "favorite removed" {
		t.Fatalf("Select+A in Favorites: %d -> %d rows, notice %q", shown, len(a.view), a.notice)
	}
	k.tap(platform.KeyDown)
	if a.notice != "favorite removed" {
		t.Fatal("a cursor move took the unstar's notice down")
	}
}

// A bar that stays as it was keeps its notice until it expires: moving
// the cursor, held or not, and changes that arrive without a press, which
// show once it has gone.
func TestUnchangedBarKeepsNoticeUntilItExpires(t *testing.T) {
	a, k := noticeApp()
	a.Notice("saved screenshot", 3*time.Second)
	deadline := a.until
	k.tap(platform.KeyDown)
	k.tap(platform.KeyDown)
	k.tap(platform.KeyUp)
	a.SetAppUpdate("v9.9.9")
	a.SetNet("offline")
	k.tap(platform.KeyDown)
	a.Paint()
	if a.notice != "saved screenshot" {
		t.Fatalf("notice %q taken down early", a.notice)
	}
	a.Tick(deadline.Add(-time.Millisecond))
	if a.notice == "" {
		t.Fatal("notice expired early")
	}
	if !a.Tick(deadline) || a.notice != "" || !a.statusBar().update {
		t.Fatalf("at its deadline: notice %q, bar %+v", a.notice, a.statusBar())
	}
}

// A jump's month stays up while the cursor moves under it, by taps and
// by a held direction's repeats.
func TestJumpNoticeSurvivesCursorMoves(t *testing.T) {
	a, k := noticeApp()
	k.tap(platform.KeyPageDown)
	label := a.notice
	if label == "" {
		t.Fatal("the jump named no month")
	}
	k.tap(platform.KeyDown)
	k.tap(platform.KeyUp)
	k.edge(platform.KeyDown, true)
	from := a.CursorKey()
	at := k.now.Add(time.Duration(a.HoldDelay()) * time.Millisecond)
	for i := range 20 {
		a.Frame(at.Add(time.Duration(i) * frameDur))
	}
	*k.now = at.Add(20 * frameDur)
	k.edge(platform.KeyDown, false)
	if a.CursorKey() == from || a.notice != label {
		t.Fatalf("held Down: cursor %q -> %q, notice %q, want %q", from, a.CursorKey(), a.notice, label)
	}
}

// A notice set while a screen without one shows, Update All's, waits
// for the screen that shows it.
func TestNoticeWaitsForAScreenThatShowsIt(t *testing.T) {
	a, k := noticeApp()
	a.SetUpdate(updater.State{ID: "run", Status: "completed", Started: *k.now, Heartbeat: *k.now}, true)
	if a.screen != ScreenUpdate {
		t.Fatalf("screen %v", a.screen)
	}
	a.Notice("Cannot restart while updater status is uncertain", 8*time.Second)
	k.tap(platform.KeyDown)
	k.tap(platform.KeyBack)
	if a.screen == ScreenUpdate || a.notice == "" {
		t.Fatalf("B: screen %v, notice %q", a.screen, a.notice)
	}
	k.tap(platform.KeyDown)
	if a.notice == "" {
		t.Fatal("a cursor move took down the notice the update screen held back")
	}
}

// The Menu hold's hint belongs to the button: it stays while the button
// is down, whatever changes under it.
func TestMenuHoldHintOutlastsTheBar(t *testing.T) {
	a, k := noticeApp()
	k.edge(platform.KeyMenu, true)
	a.Tick(k.now.Add(menuHint))
	if a.notice != menuHoldNotice {
		t.Fatalf("hold: notice %q", a.notice)
	}
	view := a.Sort()
	*k.now = k.now.Add(menuHint)
	k.tap(platform.KeySpace)
	if a.Sort() == view || a.notice != menuHoldNotice {
		t.Fatalf("Y under the hold: view %v -> %v, notice %q", view, a.Sort(), a.notice)
	}
}

// The lock screen's bar stays its name while a code is typed, so a
// notice there lasts; the unlock's notice then gives way to the list's
// first change.
func TestLockScreenNoticeLastsWhileTyping(t *testing.T) {
	l := newLockedApp(t, Config{})
	l.Notice("Remote debug enabled", 8*time.Second)
	l.typed("123")
	l.tap(platform.KeyUp)
	if l.notice != "Remote debug enabled" {
		t.Fatalf("typing on the lock screen: notice %q", l.notice)
	}
	l.typed("456")
	l.tap(platform.KeyEnter)
	if l.Locked() || l.notice != lockUnlocked {
		t.Fatalf("unlock: locked %v, notice %q", l.Locked(), l.notice)
	}
	l.tap(platform.KeyDown)
	if l.notice != lockUnlocked {
		t.Fatal("a cursor move took the unlock's notice down")
	}
	l.tap(platform.KeySpace)
	if l.notice != "" {
		t.Fatalf("Y: notice %q", l.notice)
	}
}
