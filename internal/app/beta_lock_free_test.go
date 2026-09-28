package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

// freeLockedApp is a locked beta on a card that can switch back to the
// free version (can), recording the actions it asks the host for.
func freeLockedApp(t *testing.T, can bool) (*lockedApp, *[]string) {
	t.Helper()
	actions := &[]string{}
	l := newLockedApp(t, Config{
		CanSwitchToFree: func() bool { return can },
		Action:          func(kind, arg string) { *actions = append(*actions, kind+" "+arg) },
	})
	return l, actions
}

func TestLockScreenOffersTheFreeVersionOnlyWhereTheCardCanSwitch(t *testing.T) {
	l, actions := freeLockedApp(t, false)
	l.tap(platform.KeyTab)
	l.Paint()
	if l.lock.message != "" || len(*actions) != 0 || l.Screen() != ScreenList {
		t.Fatalf("X on a card that cannot switch: message %q actions %q screen %v", l.lock.message, *actions, l.Screen())
	}
	if strings.Contains(l.lockHint(), "X ") || l.lockSwitchLine(80) != "" {
		t.Fatalf("offered anyway: legend %q line %q", l.lockHint(), l.lockSwitchLine(80))
	}

	l, _ = freeLockedApp(t, true)
	if !strings.HasSuffix(l.lockHint(), "X Free version") || l.lockSwitchLine(80) != "X Switch to the free version" {
		t.Fatalf("not offered: legend %q line %q", l.lockHint(), l.lockSwitchLine(80))
	}
	// a narrow body takes the short line
	if got := l.lockSwitchLine(20); got != "X Free version" {
		t.Fatalf("narrow line %q", got)
	}
}

// The first X asks; B or a changed digit drops the question, and the
// entry stays. The second X starts the switch, still locked.
func TestLockScreenSwitchAsksFirst(t *testing.T) {
	l, actions := freeLockedApp(t, true)
	l.typed("12")
	l.tap(platform.KeyTab)
	if l.lock.message != l.lockSwitchAsk() || len(*actions) != 0 {
		t.Fatalf("first X: message %q actions %q", l.lock.message, *actions)
	}
	if !strings.Contains(l.lockHint(), "B Clear") {
		t.Fatalf("the legend while asking: %q", l.lockHint())
	}
	l.tap(platform.KeyBack)
	if l.lock.message != "" || l.entry() != "12____" || l.quits != 0 {
		t.Fatalf("B on the question: message %q entry %q quits %d", l.lock.message, l.entry(), l.quits)
	}
	l.tap(platform.KeyTab)
	l.tap(platform.KeyDown)
	l.tap(platform.KeyTab)
	if l.lock.message != l.lockSwitchAsk() || len(*actions) != 0 {
		t.Fatalf("a digit turned between the presses did not ask again: message %q actions %q", l.lock.message, *actions)
	}
	l.tap(platform.KeyTab)
	if got := strings.Join(*actions, ","); got != "update "+updater.ModeFree {
		t.Fatalf("second X asked the host for %q", got)
	}
	if !l.Locked() || !l.lock.switching || l.Screen() != ScreenUpdate || l.UpdateState().Mode != updater.ModeFree {
		t.Fatalf("the switch: locked %v switching %v screen %v mode %q", l.Locked(), l.lock.switching, l.Screen(), l.UpdateState().Mode)
	}
}

// While the switch runs and after it, the update screen's keys work, but
// nothing leads past the lock: B after the run comes back to the lock
// screen, not Options.
func TestLockSwitchScreenKeepsTheLock(t *testing.T) {
	l, actions := freeLockedApp(t, true)
	l.typed("12")
	l.tap(platform.KeyTab)
	l.tap(platform.KeyTab)
	run := updater.State{ID: "switch", Mode: updater.ModeFree, Status: "running", Lines: []string{"Running MiSTer Downloader"}}
	l.SetUpdate(run, true)
	*actions = nil
	for _, k := range []platform.Key{platform.KeySpace, platform.KeyStart, platform.KeySelect, platform.KeyEnter, platform.KeyHome, platform.KeyMenu} {
		l.tap(k)
	}
	l.typed("3")
	l.Paint()
	if !l.Locked() || !l.lock.switching || l.Screen() != ScreenUpdate || l.entry() != "12____" || l.quits != 0 || len(l.launches) != 0 || len(*actions) != 0 {
		t.Fatalf("keys during the run: locked %v switching %v screen %v entry %q quits %d launches %q actions %q",
			l.Locked(), l.lock.switching, l.Screen(), l.entry(), l.quits, l.launches, *actions)
	}
	if len(l.imgs.wants) != 0 {
		t.Fatalf("the switch screen asked for pictures: %+v", l.imgs.wants)
	}
	// a tap of B does not leave a running switch; holding it cancels
	l.tap(platform.KeyBack)
	if !l.lock.switching || l.Screen() != ScreenUpdate {
		t.Fatal("B left the running switch")
	}
	l.now = l.now.Add(50 * time.Millisecond)
	l.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: l.now})
	for end := l.now.Add(cancelHold + 100*time.Millisecond); l.now.Before(end); {
		l.now = l.now.Add(50 * time.Millisecond)
		l.Tick(l.now)
	}
	l.Handle(platform.Event{Key: platform.KeyBack, At: l.now})
	if got := strings.Join(*actions, ","); got != "update-cancel switch" {
		t.Fatalf("holding B asked for %q", got)
	}
	*actions = nil

	// finished, with the free version on the card: A restarts into it
	run.Status = "completed"
	l.SetUpdate(run, true)
	l.SetUpdateRestart(run.ID, true)
	l.tap(platform.KeyEnter)
	if got := strings.Join(*actions, ","); got != "update-restart switch" {
		t.Fatalf("A after the switch asked for %q", got)
	}
	*actions = nil
	// B comes back to the lock screen with the entry as it was
	l.tap(platform.KeyBack)
	l.Paint()
	if got := strings.Join(*actions, ","); got != "update-dismiss switch" {
		t.Fatalf("B after the switch asked for %q", got)
	}
	if !l.Locked() || l.lock.switching || l.Screen() != ScreenList || l.entry() != "12____" || l.quits != 0 {
		t.Fatalf("B after the switch: locked %v switching %v screen %v entry %q quits %d", l.Locked(), l.lock.switching, l.Screen(), l.entry(), l.quits)
	}
	// the lock screen works as before, and a later snapshot of the same
	// run does not bring its screen back
	l.SetUpdate(run, false)
	l.typed("3456")
	l.tap(platform.KeyEnter)
	if l.Locked() || l.Screen() != ScreenList {
		t.Fatalf("the code after the switch: locked %v screen %v", l.Locked(), l.Screen())
	}
}

// Menu after the run returns to the MiSTer menu, where the update screen
// would open Options.
func TestLockSwitchMenuLeaves(t *testing.T) {
	for _, button := range []string{"options", "leave"} {
		l, _ := freeLockedApp(t, true)
		l.cfg.MenuButton = button
		l.tap(platform.KeyTab)
		l.tap(platform.KeyTab)
		l.SetUpdate(updater.State{ID: "switch", Mode: updater.ModeFree, Status: "failed", Message: "Downloader is not installed on this card"}, true)
		l.tap(platform.KeyMenu)
		if l.quits != 1 || l.Screen() == ScreenOptions || !l.Locked() {
			t.Fatalf("%s: Menu after the switch: quits %d screen %v locked %v", button, l.quits, l.Screen(), l.Locked())
		}
	}
}

// An update result the app opened on stays under the lock screen: only the
// lock's own switch shows an update screen while locked.
func TestLockScreenCoversAnEarlierUpdateResult(t *testing.T) {
	l, actions := freeLockedApp(t, true)
	l.SetUpdate(updater.State{ID: "earlier", Mode: updater.ModeAll, Status: "completed"}, true)
	l.Paint()
	l.tap(platform.KeyDown)
	if l.entry() != "9_____" || l.lock.switching {
		t.Fatalf("the lock did not take the key: entry %q switching %v", l.entry(), l.lock.switching)
	}
	l.tap(platform.KeyBack) // clears the entry
	l.tap(platform.KeyBack) // quits, where the update screen would open Options
	if l.quits != 1 || l.Screen() == ScreenOptions || len(*actions) != 0 {
		t.Fatalf("B under the lock: quits %d screen %v actions %q", l.quits, l.Screen(), *actions)
	}
}

// With page transitions on, the switch's screen arrives as a forward page
// and the lock screen comes back as a backward one.
func TestLockSwitchWipes(t *testing.T) {
	l, _ := freeLockedApp(t, true)
	l.EnablePageTransitions()
	l.Paint()
	l.tap(platform.KeyTab)
	l.Paint()
	l.tap(platform.KeyTab)
	l.Paint()
	if !l.PageTransitionRunning() || l.transition.back {
		t.Fatalf("into the switch: running %v back %v", l.PageTransitionRunning(), l.transition.back)
	}
	l.SetUpdate(updater.State{ID: "switch", Mode: updater.ModeFree, Status: "failed"}, true)
	for i := 0; i < 30; i++ {
		l.now = l.now.Add(20 * time.Millisecond)
		l.Tick(l.now)
		l.Paint()
	}
	l.tap(platform.KeyBack)
	l.Paint()
	if !l.PageTransitionRunning() || !l.transition.back {
		t.Fatalf("back to the lock: running %v back %v", l.PageTransitionRunning(), l.transition.back)
	}
}
