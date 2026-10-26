package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"testing"
	"time"
)

// Existing sorting tests take the next enabled view through the public actions.
func chooseNextView(a *App) {
	a.act(platform.KeySpace)
	a.act(platform.KeyDown)
	if a.viewChooser.cursor == len(a.viewChooser.entries) {
		a.act(platform.KeyHome)
	}
	a.act(platform.KeyEnter)
}

func TestViewChooserCancelAndCommit(t *testing.T) {
	a, tap := viewsApp("recents", "debut")
	a.SetSort(data.SortAlphabetical)
	a.MoveToKey("b")
	a.setSearch("Beta")
	cursor, top := a.cursor, a.top
	tap(platform.KeySpace)
	if a.screen != ScreenViewChooser || a.viewChooser.entries[a.viewChooser.cursor].value != "alphabetical" {
		t.Fatal("chooser did not open on current view")
	}
	for _, e := range a.viewChooser.entries {
		if e.value == "recents" || e.value == "debut" {
			t.Fatal("disabled view offered")
		}
	}
	tap(platform.KeyDown)
	a.Handle(platform.Event{Text: 'x', Pressed: true, At: time.Now()})
	tap(platform.KeyStart)
	if a.mode != data.SortAlphabetical || a.Search() != "Beta" || a.cursor != cursor || a.top != top {
		t.Fatal("browsing changed the list")
	}
	tap(platform.KeyBack)
	if a.screen != ScreenList || a.mode != data.SortAlphabetical || a.CursorKey() != "b" {
		t.Fatal("cancel changed selection")
	}
	tap(platform.KeySpace)
	tap(platform.KeyDown)
	tap(platform.KeyEnter)
	if a.screen != ScreenList || a.mode != data.SortMaker || a.Search() != "Beta" || a.CursorKey() != "b" {
		t.Fatal("selection lost context")
	}
}

func TestViewChooserEditReturnAndLastView(t *testing.T) {
	a, tap := viewsApp("recents")
	tap(platform.KeySpace)
	tap(platform.KeyUp)
	if a.viewChooser.cursor != len(a.viewChooser.entries) {
		t.Fatal("Up did not reach fixed Edit row")
	}
	tap(platform.KeyEnter)
	if a.screen != ScreenViews || a.viewsPage != "chooser" {
		t.Fatal("editor not opened from chooser")
	}
	tap(platform.KeyEnter) // disable current Build date
	a.panel.cursor = 10
	tap(platform.KeyEnter) // enable Shoot 'em ups
	tap(platform.KeyBack)
	if a.screen != ScreenViewChooser || a.mode != data.SortDebut || a.viewChooser.entries[a.viewChooser.cursor].value != "debut" {
		t.Fatal("editor did not return to updated chooser")
	}
	found := false
	for _, e := range a.viewChooser.entries {
		if e.value == "shmups" {
			found = true
		}
	}
	if !found {
		t.Fatal("new genre absent")
	}
	for _, m := range data.ViewOrder {
		if m != data.SortShmups {
			a.setViewOn(m, false)
		}
	}
	a.openViewChooser()
	tap(platform.KeyEnd)
	tap(platform.KeyEnter)
	a.panel.cursor = 10
	tap(platform.KeyEnter)
	if !a.viewOn(data.SortShmups) || a.viewsOnCount() != 1 {
		t.Fatal("last view disabled through popup")
	}
	tap(platform.KeyBack)
	tap(platform.KeyHome)
	tap(platform.KeyEnter)
	if a.screen != ScreenList || a.mode != data.SortShmups {
		t.Fatal("single view cannot be selected")
	}
}

func TestViewChooserScrollAndMenuReturn(t *testing.T) {
	a, tap := viewsApp()
	for _, m := range data.ViewOrder {
		a.setViewOn(m, true)
	}
	a.SetSort(data.SortOther)
	tap(platform.KeySpace)
	a.Paint()
	if a.viewChooser.top == 0 || a.viewChooser.cursor < a.viewChooser.top || a.viewChooser.cursor >= a.viewChooser.top+a.viewChooser.lines {
		t.Fatal("current view is outside scroll window")
	}
	a.menuButton()
	if a.screen != ScreenOptions {
		t.Fatal("Menu did not open Options")
	}
	tap(platform.KeyBack)
	if a.screen != ScreenViewChooser {
		t.Fatal("Options did not return to chooser")
	}
	tap(platform.KeyEnd)
	a.rep.count = 1
	a.act(platform.KeyDown)
	if a.viewChooser.cursor != len(a.viewChooser.entries) {
		t.Fatal("held Down wrapped")
	}
}

func TestViewChooserOnlySelectionStartsTransition(t *testing.T) {
	a, tap := viewsApp()
	a.EnablePageTransitions()
	a.Paint()
	tap(platform.KeySpace)
	a.Paint()
	if a.PageTransitionRunning() {
		t.Fatal("opening popup started a page transition")
	}
	tap(platform.KeyDown)
	a.Paint()
	if a.PageTransitionRunning() {
		t.Fatal("browsing popup started a page transition")
	}
	tap(platform.KeyBack)
	a.Paint()
	if a.PageTransitionRunning() {
		t.Fatal("cancelling popup started a page transition")
	}
	tap(platform.KeySpace)
	tap(platform.KeyDown)
	tap(platform.KeyEnter)
	a.Paint()
	if !a.PageTransitionRunning() {
		t.Fatal("selecting a different view did not animate")
	}
}
