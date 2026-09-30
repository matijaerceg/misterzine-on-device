package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

// backApp is the free build on a card that had MisterZine Arcade, with the
// page up, recording the actions and saves it asks the host for.
type backApp struct {
	*App
	now     time.Time
	actions []string
	saves   int
}

func newBackApp(t *testing.T) *backApp {
	t.Helper()
	b := &backApp{now: time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)}
	b.App = New(Config{PhysW: 320, PhysH: 240, SafeInsetX: 15, SafeInsetY: 15, ArcadeBack: true,
		Now: func() time.Time { return b.now }, TimerNow: func() time.Time { return b.now },
		Action:          func(kind, arg string) { b.actions = append(b.actions, kind+" "+arg) },
		SettingsChanged: func() { b.saves++ },
	}, data.Ingest([]data.Row{{K: "red", Title: "Red Game", Base: "Arcade", MRA: "_Arcade/Red.mra"}}, "test", b.now), nil)
	return b
}

func (b *backApp) key(k platform.Key, pressed bool) {
	b.Handle(platform.Event{Key: k, Pressed: pressed, At: b.now})
}

func (b *backApp) wait(d time.Duration) {
	for end := b.now.Add(d); b.now.Before(end); {
		b.now = b.now.Add(10 * time.Millisecond)
		b.Tick(b.now)
	}
}

// B keeps the free version for good, and the list is back.
func TestArcadeBackStayFree(t *testing.T) {
	b := newBackApp(t)
	b.Paint()
	b.key(platform.KeyDown, true) // nothing reaches the list under it
	b.key(platform.KeyDown, false)
	if !b.ArcadeBackPending() || b.saves != 0 || len(b.actions) != 0 {
		t.Fatalf("Down on the page: pending %v saves %d actions %q", b.ArcadeBackPending(), b.saves, b.actions)
	}
	b.now = b.now.Add(50 * time.Millisecond)
	b.key(platform.KeyBack, true)
	if b.ArcadeBackPending() || b.saves != 1 || len(b.actions) != 0 || b.Screen() != ScreenList {
		t.Fatalf("B: pending %v saves %d actions %q screen %v", b.ArcadeBackPending(), b.saves, b.actions, b.Screen())
	}
	b.key(platform.KeyBack, false)
	if b.Screen() != ScreenList {
		t.Fatalf("the release of B left the list: %v", b.Screen())
	}
}

// A let go early starts again; A held its full time goes back through the
// member's installer, and its release does nothing on the update screen.
func TestArcadeBackHoldGoesBack(t *testing.T) {
	b := newBackApp(t)
	b.key(platform.KeyEnter, true)
	b.wait(arcadeBackHold / 2)
	if b.holdBar() == 0 || !b.ArcadeBackPending() {
		t.Fatalf("half a hold: bar %d pending %v", b.holdBar(), b.ArcadeBackPending())
	}
	b.key(platform.KeyEnter, false)
	if b.holdBar() != 0 || !b.ArcadeBackPending() || len(b.actions) != 0 || b.saves != 0 {
		t.Fatalf("let go early: bar %d pending %v actions %q saves %d", b.holdBar(), b.ArcadeBackPending(), b.actions, b.saves)
	}
	b.now = b.now.Add(50 * time.Millisecond)
	b.key(platform.KeyEnter, true)
	b.wait(arcadeBackHold)
	if b.ArcadeBackPending() || b.saves != 1 || strings.Join(b.actions, ",") != "update "+updater.ModeBeta ||
		b.Screen() != ScreenUpdate || b.UpdateState().Mode != updater.ModeBeta {
		t.Fatalf("a full hold: pending %v saves %d actions %q screen %v mode %q",
			b.ArcadeBackPending(), b.saves, b.actions, b.Screen(), b.UpdateState().Mode)
	}
	b.key(platform.KeyEnter, false)
	if len(b.actions) != 1 || b.Screen() != ScreenUpdate {
		t.Fatalf("the release acted on the update screen: actions %q screen %v", b.actions, b.Screen())
	}
}

// The page's words and legend fit every layout.
func TestArcadeBackFits(t *testing.T) {
	ds := data.Ingest([]data.Row{{K: "red", Title: "Red Game", Base: "Arcade"}}, "test", time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC))
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		for _, inset := range []int{0, 15, 40} {
			for _, labels := range []string{"mister", "playstation"} {
				a := New(Config{PhysW: 320, PhysH: 240, Rotation: rot, SafeInsetX: inset, SafeInsetY: inset, ArcadeBack: true, ButtonLabels: labels}, ds, nil)
				a.Paint()
				box := a.lay.Body.Inset(4)
				for _, p := range a.arcadeBackLines(box) {
					if last := p[len(p)-1]; strings.HasSuffix(last, gfx.Ellipsis) {
						t.Fatalf("rot %v inset %d: a paragraph is cut: %q", rot, inset, p)
					}
				}
				if h := a.arcadeBackHeight(box); h > box.Dy() {
					t.Fatalf("rot %v inset %d %s: the page is %d high in a %d body", rot, inset, labels, h, box.Dy())
				}
				if !a.hintFits(a.arcadeBackHint()) {
					t.Fatalf("rot %v inset %d %s: the legend %q does not fit", rot, inset, labels, a.arcadeBackHint())
				}
			}
		}
	}
}
