package app

import (
	"errors"
	"image"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/beta"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// lockedApp is a locked beta build whose code is 123456: unlock records
// each code the lock screen hands over and answers with answer when it is
// set. quits counts Quit calls and launches the games started.
type lockedApp struct {
	*App
	now      time.Time
	tried    []string
	answer   error
	quits    int
	launches []string
	imgs     *shotImages
}

func newLockedApp(t *testing.T, cfg Config) *lockedApp {
	t.Helper()
	l := &lockedApp{now: time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC), imgs: &shotImages{have: map[string]*image.RGBA{"red/snap": flat(320, 240, shotRed)}, state: map[string]ImageState{}}}
	cfg.PhysW, cfg.PhysH = 320, 240
	cfg.SafeInsetX, cfg.SafeInsetY = 15, 15
	cfg.Now = func() time.Time { return l.now }
	cfg.Images = l.imgs
	cfg.Quit = func() { l.quits++ }
	cfg.Launch = func(p string) { l.launches = append(l.launches, p) }
	cfg.Exists = func(string) bool { return true }
	cfg.Status = func(int) data.Status { return data.StatusCurrent }
	cfg.BetaUnlock = func(code string) error {
		l.tried = append(l.tried, code)
		if l.answer != nil {
			return l.answer
		}
		if code != "123456" {
			return beta.ErrLocked
		}
		return nil
	}
	l.App = New(cfg, data.Ingest([]data.Row{
		{K: "red", Title: "Red Game", Base: "Arcade", MRA: "_Arcade/Red.mra", Img: "red", ImgSlots: []string{"snap"}},
		{K: "blue", Title: "Blue Game", Base: "Arcade", MRA: "_Arcade/Blue.mra"},
	}, "test", l.now), nil)
	return l
}

// tap presses and releases a key, spaced past the bounce guard.
func (l *lockedApp) tap(k platform.Key) {
	l.now = l.now.Add(50 * time.Millisecond)
	l.Handle(platform.Event{Key: k, Pressed: true, At: l.now})
	l.now = l.now.Add(20 * time.Millisecond)
	l.Handle(platform.Event{Key: k, At: l.now})
}

// typed sends keyboard text, a press per character as the keyboard does.
func (l *lockedApp) typed(s string) {
	for _, ch := range s {
		l.now = l.now.Add(50 * time.Millisecond)
		l.Handle(platform.Event{Key: platform.KeyOther, Text: ch, Pressed: true, At: l.now})
		l.Handle(platform.Event{Key: platform.KeyOther, At: l.now})
	}
}

func (l *lockedApp) entry() string {
	b := []byte{}
	for _, d := range l.lock.digits {
		if d < 0 {
			b = append(b, '_')
		} else {
			b = append(b, '0'+byte(d))
		}
	}
	return string(b)
}

func TestLockScreenPadEntry(t *testing.T) {
	l := newLockedApp(t, Config{})
	if !l.Locked() || l.entry() != "______" || l.lock.box != 0 {
		t.Fatalf("does not open locked and empty: %q box %d", l.entry(), l.lock.box)
	}
	// an empty box starts at 0 going up and at 9 going down, and wraps
	l.tap(platform.KeyUp)
	l.tap(platform.KeyUp)
	l.tap(platform.KeyRight)
	l.tap(platform.KeyDown)
	l.tap(platform.KeyRight)
	for i := 0; i < 10; i++ {
		l.tap(platform.KeyUp)
	}
	if l.entry() != "199___" || l.lock.box != 2 {
		t.Fatalf("entry %q box %d", l.entry(), l.lock.box)
	}
	// Left and Right stop at the ends
	for i := 0; i < 8; i++ {
		l.tap(platform.KeyRight)
	}
	if l.lock.box != 5 {
		t.Fatalf("box %d after walking right", l.lock.box)
	}
	for i := 0; i < 8; i++ {
		l.tap(platform.KeyLeft)
	}
	if l.lock.box != 0 {
		t.Fatalf("box %d after walking left", l.lock.box)
	}
	// an incomplete entry is not tried: it says so and goes to the gap
	l.tap(platform.KeyEnter)
	if len(l.tried) != 0 || l.lock.message != lockIncomplete || l.lock.box != 3 || !l.Locked() {
		t.Fatalf("incomplete entry: tried %q message %q box %d", l.tried, l.lock.message, l.lock.box)
	}
	for _, k := range []platform.Key{platform.KeyUp, platform.KeyRight, platform.KeyUp, platform.KeyRight, platform.KeyUp} {
		l.tap(k)
	}
	// a wrong code keeps the entry, to put right
	l.tap(platform.KeyStart)
	if len(l.tried) != 1 || l.tried[0] != "199000" || l.lock.message != lockWrong || l.entry() != "199000" || !l.Locked() {
		t.Fatalf("wrong code: tried %q message %q entry %q", l.tried, l.lock.message, l.entry())
	}
	// turning a digit takes the message away
	l.tap(platform.KeyDown)
	if l.lock.message != "" {
		t.Fatal("the wrong-code message outlived the next change")
	}
}

func TestLockScreenKeyboardEntryUnlocksIntoTheList(t *testing.T) {
	l := newLockedApp(t, Config{})
	l.typed("12")
	l.tap(platform.KeyBackspace) // the box after the typed ones is empty: erases the 2
	if l.entry() != "1_____" || l.lock.box != 1 {
		t.Fatalf("after Backspace: %q box %d", l.entry(), l.lock.box)
	}
	l.typed("234567") // the last box takes the last digit typed
	if l.entry() != "123457" || l.lock.box != 5 {
		t.Fatalf("typed: %q box %d", l.entry(), l.lock.box)
	}
	l.tap(platform.KeyBackspace) // the chosen box is filled: erases it in place
	l.typed("6")
	l.tap(platform.KeyEnter)
	if l.Locked() {
		t.Fatalf("the right code did not unlock: tried %q message %q", l.tried, l.lock.message)
	}
	if l.Screen() != ScreenList || l.notice != lockUnlocked {
		t.Fatalf("unlocked onto %v with notice %q", l.Screen(), l.notice)
	}
	// the list works at once: the cursor moves and Details opens
	first := l.CursorKey()
	l.tap(platform.KeyDown)
	if l.CursorKey() == first {
		t.Fatalf("cursor stayed on %q after Down", first)
	}
	l.tap(platform.KeyEnter)
	if l.Screen() != ScreenDetails {
		t.Fatalf("A opened %v", l.Screen())
	}
}

// Nothing but the lock screen is reachable: no Options, Filters, search,
// sort, Details or launch, and a Start hold launches nothing.
func TestLockScreenReachesNothingElse(t *testing.T) {
	l := newLockedApp(t, Config{})
	sort := l.Sort()
	l.typed("ab c")
	for _, k := range []platform.Key{platform.KeyTab, platform.KeySpace, platform.KeyPageDown, platform.KeyHome, platform.KeyEnd, platform.KeySelect, platform.KeyOther} {
		l.tap(k)
	}
	l.now = l.now.Add(50 * time.Millisecond)
	l.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: l.now})
	for i := 0; i < 300; i++ {
		l.now = l.now.Add(10 * time.Millisecond)
		l.Tick(l.now)
	}
	l.Handle(platform.Event{Key: platform.KeyStart, At: l.now})
	if !l.Locked() || l.Screen() != ScreenList || l.query != "" || l.Sort() != sort || len(l.launches) != 0 || l.quits != 0 {
		t.Fatalf("got past the lock: screen %v query %q sort %v launches %q quits %d", l.Screen(), l.query, l.Sort(), l.launches, l.quits)
	}
	// the page under the lock is not painted and asks for no pictures
	l.Paint()
	if len(l.imgs.wants) != 0 {
		t.Fatalf("the locked app asked for pictures: %+v", l.imgs.wants)
	}
	// what the host may do meanwhile does not open the app either
	l.MoveToKey("blue")
	l.Paint()
	if !l.Locked() {
		t.Fatal("moving the cursor unlocked")
	}
}

func TestLockScreenBackClearsThenQuits(t *testing.T) {
	for _, k := range []platform.Key{platform.KeyBack, platform.KeyMenu} {
		l := newLockedApp(t, Config{})
		l.typed("12")
		l.tap(k)
		if k == platform.KeyMenu {
			// the Menu button always leaves for the MiSTer menu
			if l.quits != 1 {
				t.Fatalf("Menu with an entry: %d quits", l.quits)
			}
			continue
		}
		if l.quits != 0 || l.entry() != "______" || l.lock.box != 0 {
			t.Fatalf("B on an entry: %d quits, entry %q box %d", l.quits, l.entry(), l.lock.box)
		}
		if got := l.lockHint(); got[len(got)-6:] != "B Quit" {
			t.Fatalf("the legend after clearing: %q", got)
		}
		l.tap(k)
		if l.quits != 1 || !l.Locked() {
			t.Fatalf("B on an empty entry: %d quits", l.quits)
		}
	}
}

func TestLockScreenAnswers(t *testing.T) {
	// a build that cannot be unlocked says so and stays locked
	l := newLockedApp(t, Config{})
	l.answer = beta.ErrBuild
	l.typed("123456")
	l.tap(platform.KeyEnter)
	if !l.Locked() || l.lock.message != lockBadBuild {
		t.Fatalf("build error: locked %v message %q", l.Locked(), l.lock.message)
	}
	// a right code the card could not save opens the app for this session
	l = newLockedApp(t, Config{})
	l.answer = errors.New("read-only file system")
	l.typed("123456")
	l.tap(platform.KeyEnter)
	if l.Locked() || l.notice != lockUnsaved {
		t.Fatalf("unsaved unlock: locked %v notice %q", l.Locked(), l.notice)
	}
}

func TestLockScreenHeldDigitRepeats(t *testing.T) {
	l := newLockedApp(t, Config{HoldDelay: 300})
	l.Handle(platform.Event{Key: platform.KeyUp, Pressed: true, At: l.now})
	// the press, then a step every repeatStep once the hold delay is over
	end := l.now.Add(300*time.Millisecond + 2*repeatStep)
	for l.now.Before(end) {
		l.now = l.now.Add(10 * time.Millisecond)
		l.Tick(l.now)
	}
	l.Handle(platform.Event{Key: platform.KeyUp, At: l.now})
	if l.entry() != "3_____" {
		t.Fatalf("held Up: %q", l.entry())
	}
	// Enter never repeats: one try however long it is held
	l.typed("311111")
	l.Handle(platform.Event{Key: platform.KeyEnter, Pressed: true, At: l.now})
	for i := 0; i < 200; i++ {
		l.now = l.now.Add(10 * time.Millisecond)
		l.Tick(l.now)
	}
	if len(l.tried) != 1 {
		t.Fatalf("held Enter tried %d times", len(l.tried))
	}
}

// The screensaver runs over the lock screen as over any other, with the
// lettering in place of the screenshots, whose Start hold plays a game.
func TestLockScreenSaver(t *testing.T) {
	l := newLockedApp(t, Config{SaverStyle: "shots", Screensaver: "1"})
	l.typed("12")
	l.now = l.now.Add(time.Minute + time.Second)
	l.Tick(l.now)
	l.Paint()
	if !l.ScreensaverActive() || l.saver.style != "word" || l.saver.shots != nil {
		t.Fatalf("saver active %v style %q", l.ScreensaverActive(), l.saver.style)
	}
	// a key wakes it and does nothing else
	l.tap(platform.KeyStart)
	if l.ScreensaverActive() || len(l.tried) != 0 || l.entry() != "12____" {
		t.Fatalf("the wake acted: tried %q entry %q", l.tried, l.entry())
	}
	// once unlocked, the saver's own style is back
	l.typed("3456")
	l.tap(platform.KeyEnter)
	l.startSaver(l.now)
	if l.saver.style != "shots" {
		t.Fatalf("unlocked saver style %q", l.saver.style)
	}
}

// statusHasMark reports whether the status bar's right end holds the
// mark's amber.
func statusHasMark(a *App) bool {
	st := a.lay.Status
	for y := st.Min.Y; y < st.Max.Y; y++ {
		for x := st.Max.X - 40; x < st.Max.X; x++ {
			if a.logical.RGBAAt(x, y) == gen.Eva.Warn {
				return true
			}
		}
	}
	return false
}

func TestBetaMarkOnlyInTheBeta(t *testing.T) {
	rows := data.Ingest([]data.Row{{K: "red", Title: "Red Game", Base: "Arcade"}}, "test", time.Now())
	for _, layout := range listLayouts {
		for _, on := range []bool{false, true} {
			restore := beta.Set(on)
			a := New(Config{PhysW: 320, PhysH: 240, SafeInsetX: 15, SafeInsetY: 15, ListLayout: layout}, rows, nil)
			a.Paint()
			if got := statusHasMark(a); got != on {
				t.Errorf("%s, beta %v: mark %v", layout, on, got)
			}
			// Options is not the main screen: no mark
			a.openOptions()
			a.Paint()
			if statusHasMark(a) {
				t.Errorf("%s, beta %v: mark on Options", layout, on)
			}
			restore()
		}
	}
}

// A free build is never locked, whatever it is handed.
func TestFreeBuildHasNoLock(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(nil, "test", time.Now()), nil)
	if a.Locked() {
		t.Fatal("a build without BetaUnlock is locked")
	}
}

// With page transitions on, the list arrives the way a forward page does.
func TestUnlockWipesForwardIntoTheList(t *testing.T) {
	l := newLockedApp(t, Config{})
	l.EnablePageTransitions()
	l.Paint()
	l.typed("123456")
	l.tap(platform.KeyEnter)
	l.Paint()
	if !l.PageTransitionRunning() || l.transition.back {
		t.Fatalf("unlock transition running %v back %v", l.PageTransitionRunning(), l.transition.back)
	}
}
