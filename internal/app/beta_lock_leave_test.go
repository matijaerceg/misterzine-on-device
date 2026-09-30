package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/beta"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

// The lock screen has no way to the free version of its own: X does
// nothing and the legend names no X, however the card is set up.
func TestLockScreenOffersNoSwitch(t *testing.T) {
	for _, earlier := range []bool{false, true} {
		actions := []string{}
		l := newLockedApp(t, Config{BetaEarlierBatch: earlier, Action: func(kind, arg string) { actions = append(actions, kind+" "+arg) }})
		l.typed("12")
		l.tap(platform.KeyTab)
		l.tap(platform.KeyTab)
		l.Paint()
		if l.lock.message != "" || len(actions) != 0 || l.Screen() != ScreenList || !l.Locked() || l.entry() != "12____" {
			t.Fatalf("earlier %v: X twice: message %q actions %q screen %v locked %v entry %q",
				earlier, l.lock.message, actions, l.Screen(), l.Locked(), l.entry())
		}
		if strings.Contains(l.lockHint(), "X ") {
			t.Fatalf("earlier %v: the legend names X: %q", earlier, l.lockHint())
		}
	}
}

// Only a card unlocked for an earlier batch names the way back to free,
// as text; a first install never sees it.
func TestLockScreenNamesTheWayOutOnlyToEarlierMembers(t *testing.T) {
	l := newLockedApp(t, Config{})
	if got := l.lockLeaveLines(80); got != nil {
		t.Fatalf("a first install is told the way out: %q", got)
	}
	l = newLockedApp(t, Config{BetaEarlierBatch: true})
	if got := strings.Join(l.lockLeaveLines(80), " "); got != "Leaving Beta? Run MisterZine-Switch-To-Stable from Scripts." {
		t.Fatalf("the line: %q", got)
	}
}

// In every layout the line fits the body without breaking the script's
// name, and the whole block still fits above the legend.
func TestLockLeaveLinesFit(t *testing.T) {
	ds := data.Ingest([]data.Row{{K: "red", Title: "Red Game", Base: "Arcade"}}, "test", time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC))
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		for _, inset := range []int{0, 15, 40} {
			for _, size := range [][2]int{{320, 240}, {360, 270}, {480, 270}} {
				a := New(Config{PhysW: size[0], PhysH: size[1], Rotation: rot, SafeInsetX: inset, SafeInsetY: inset,
					BetaUnlock: func(string) error { return beta.ErrLocked }, BetaEarlierBatch: true}, ds, nil)
				a.Paint()
				box := a.lay.Body.Inset(4)
				cols := a.sm.Cols(box.Dx())
				lines := a.lockLeaveLines(cols)
				whole := false
				for _, line := range lines {
					if len(line) > cols {
						t.Fatalf("rot %v inset %d %v: %q is wider than %d columns", rot, inset, size, line, cols)
					}
					switch {
					case strings.Contains(line, "Switch-To-Stable"):
						whole = true
					case strings.Contains(line, "Switch") || strings.Contains(line, "Stable") || strings.HasSuffix(line, "-"):
						t.Fatalf("rot %v inset %d %v: the script's name is broken across lines: %q", rot, inset, size, lines)
					}
				}
				if !whole || len(lines) > 3 {
					t.Fatalf("rot %v inset %d %v: %d columns give %q", rot, inset, size, cols, lines)
				}
				if cols >= len("MisterZine-Switch-To-Stable") && !strings.Contains(strings.Join(lines, " "), "MisterZine-Switch-To-Stable") {
					t.Fatalf("rot %v inset %d %v: the full name fits %d columns, yet %q", rot, inset, size, cols, lines)
				}
				if _, _, _, h := a.lockBlock(box); h > box.Dy() {
					t.Fatalf("rot %v inset %d %v: the block is %d high in a %d body", rot, inset, size, h, box.Dy())
				}
			}
		}
	}
}

// Catalogue news waits out the lock: the bar names MisterZine Arcade until
// the code is in, and the list's since-visit row has the news after.
func TestLockScreenGetsNoCatalogueNews(t *testing.T) {
	l := newLockedApp(t, Config{})
	old := l.Data()
	rows := append([]data.Row{{K: "green", Title: "Green Game", Base: "Arcade", MRA: "_Arcade/Green.mra"}}, old.Rows...)
	if news := l.CatalogueNews(old, rows); news != "" {
		t.Fatalf("news on the lock screen: %q", news)
	}
	l.typed("123456")
	l.tap(platform.KeyEnter)
	if l.Locked() {
		t.Fatal("did not unlock")
	}
	if news := l.CatalogueNews(old, rows); news == "" {
		t.Fatal("no news once unlocked")
	}
}

// An update result the app opened on stays under the lock screen: nothing
// shows an update screen while locked.
func TestLockScreenCoversAnEarlierUpdateResult(t *testing.T) {
	actions := []string{}
	l := newLockedApp(t, Config{Action: func(kind, arg string) { actions = append(actions, kind+" "+arg) }})
	l.SetUpdate(updater.State{ID: "earlier", Mode: updater.ModeAll, Status: "completed"}, true)
	l.Paint()
	l.tap(platform.KeyDown)
	if l.entry() != "9_____" {
		t.Fatalf("the lock did not take the key: entry %q", l.entry())
	}
	l.tap(platform.KeyBack) // clears the entry
	l.tap(platform.KeyBack) // quits, where the update screen would open Options
	if l.quits != 1 || l.Screen() == ScreenOptions || len(actions) != 0 {
		t.Fatalf("B under the lock: quits %d screen %v actions %q", l.quits, l.Screen(), actions)
	}
}
