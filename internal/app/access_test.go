package app

import (
	"errors"
	"github.com/matijaerceg/misterzine-on-device/internal/access"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"image"
	"strings"
	"testing"
	"time"
)

func TestOptionalCodeAccess(t *testing.T) {
	var changed access.Month
	a := New(Config{PhysW: 320, PhysH: 240, UnlockCode: func(code string) (access.Month, error) {
		if code == "123456" {
			return 202610, nil
		}
		return 0, access.ErrCode
	}, AccessChanged: func(m access.Month, _ bool) { changed = m }}, data.Ingest(nil, "", time.Now()), nil)
	if a.Locked() || a.ShowBetaFeatures() {
		t.Fatal("fresh app locked or beta enabled")
	}
	a.openOptions()
	a.openCode()
	a.actLock(platform.KeyBack)
	if a.Locked() || a.screen != ScreenOptions {
		t.Fatal("cannot leave code entry")
	}
	a.openCode()
	for _, c := range "111111" {
		a.lockType(int8(c - '0'))
	}
	a.tryUnlock()
	if !a.Locked() || a.cfg.AccessMonth != 0 {
		t.Fatal("wrong code granted access")
	}
	a.lock.clear()
	for _, c := range "123456" {
		a.lockType(int8(c - '0'))
	}
	a.tryUnlock()
	if a.Locked() || changed != 202610 || a.ShowBetaFeatures() {
		t.Fatal("unlock failed or enabled beta")
	}
	a.cfg.ShowBetaFeatures = true
	if !a.featureAllowed(access.ROMReport) {
		t.Fatal("qualified beta not enabled")
	}
	a.cfg.ShowBetaFeatures = false
	if a.featureAllowed(access.ROMReport) {
		t.Fatal("beta remains active after toggle off")
	}
}

func TestStableFancyTeaser(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(nil, "", time.Now()), nil)
	f := access.Feature{Fancy: true, Since: 202610}
	if !a.featureVisible(f) || a.featureAllowed(f) {
		t.Fatal("stable Fancy visibility/access conflated")
	}
	row := a.gatedOption(panelEntry{text: "Extra", kind: "extra", help: "Describes the feature."}, f)
	if !strings.Contains(row.help, "Describes the feature.") || !strings.Contains(row.help, "Your code unlocks it forever.") || row.requiredMonth != f.Since {
		t.Fatal("locked feature lost its description or required month")
	}
	if row.kind != "enter-code" || !row.disabled {
		t.Fatal("locked extra lacks code entry")
	}
}

// Background panel rebuilds must not move selection between locked features
// or from the standalone code-entry row to the first locked feature.
func TestLockedOptionSelectionSurvivesRebuild(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240, ShowBetaFeatures: true}, data.Ingest(nil, "", time.Now()), nil)
	a.openOptions()
	expandOptionsForTest(a)
	var names []string
	for _, e := range a.panel.entries {
		if e.kind == "enter-code" {
			names = append(names, e.text)
		}
	}
	if len(names) < 1 {
		t.Fatal("need both a locked feature and standalone code entry")
	}
	for _, name := range names {
		for i, e := range a.panel.entries {
			if e.text == name {
				a.panel.cursor = i
				break
			}
		}
		a.buildPanel()
		if got := a.panel.entries[a.panel.cursor].text; got != name {
			t.Fatalf("selection jumped from %q to %q", name, got)
		}
		a.actPanel(platform.KeyEnter)
		if !a.Locked() {
			t.Fatalf("%s did not open code entry", name)
		}
		a.actLock(platform.KeyBack)
		if got := a.panel.entries[a.panel.cursor].text; got != name {
			t.Fatalf("leaving code entry jumped from %q to %q", name, got)
		}
	}
}

func TestFreeBetaDoesNotOfferCodeEntry(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240, ShowBetaFeatures: true}, data.Ingest(nil, "", time.Now()), nil)
	for _, f := range []access.Feature{access.ROMReport, access.Controls} {
		row := a.gatedOption(panelEntry{text: "Example", kind: "example", help: "Explains the feature."}, f)
		if row.kind != "example" || row.requiredMonth != 0 || row.disabled || !a.featureAllowed(f) {
			t.Fatal("free beta asks for a code")
		}
	}
	f := access.Themes
	row := a.gatedOption(panelEntry{text: "Example", kind: "example"}, f)
	if row.kind != "enter-code" || row.requiredMonth != f.Since {
		t.Fatal("supporter beta lost code requirement")
	}
}

func TestFeatureCodeRequiresCoveredMonth(t *testing.T) {
	var saved access.Month
	a := New(Config{PhysW: 320, PhysH: 240, ShowBetaFeatures: true,
		UnlockCode: func(code string) (access.Month, error) {
			if code == "123456" {
				return 202609, nil
			}
			if code == "654321" {
				return 202709, nil
			}
			return 0, access.ErrCode
		},
		AccessChanged: func(m access.Month, _ bool) { saved = m },
	}, data.Ingest(nil, "", time.Now()), nil)
	a.openOptions()
	a.openCode()
	a.lock.requiredMonth = 202709
	a.lock.featureTitle = "Example - Supporter"
	for _, ch := range "123456" {
		a.lockType(int8(ch - '0'))
	}
	a.tryUnlock()
	if !a.Locked() || a.cfg.AccessMonth != 202609 || saved != 202609 {
		t.Fatal("older code must retain its access while keeping this feature's entry open")
	}
	if a.lock.message != "Covers September 2026. Needs September 2027 or newer." {
		t.Fatalf("unclear insufficient coverage: %q", a.lock.message)
	}
	if a.lock.started() || a.lock.box != 0 || a.lock.requiredMonth != 202709 {
		t.Fatal("retry did not reset input or preserve feature context")
	}
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		layoutApp := New(Config{PhysW: 320, PhysH: 240, Rotation: rot, SafeInsetX: 40, SafeInsetY: 40}, data.Ingest(nil, "", time.Now()), nil)
		if len(gfx.Wrap(a.lock.message, layoutApp.sm.Cols(layoutApp.lay.Body.Inset(4).Dx()), 100)) > 2 {
			t.Fatal("coverage feedback exceeds reserved message space")
		}
	}
	for _, ch := range "654321" {
		a.lockType(int8(ch - '0'))
	}
	a.tryUnlock()
	if a.Locked() || a.cfg.AccessMonth != 202709 || saved != 202709 || !a.ShowBetaFeatures() {
		t.Fatal("sufficient newer code did not complete retry")
	}
	// Entering an older code cannot reduce already held access.
	a.openCode()
	a.lock.requiredMonth = 202709
	for _, ch := range "123456" {
		a.lockType(int8(ch - '0'))
	}
	a.tryUnlock()
	if a.Locked() || a.cfg.AccessMonth != 202709 {
		t.Fatal("older code reduced existing coverage")
	}
}

func TestAccessOverviewAndReturn(t *testing.T) {
	for _, month := range []access.Month{0, 202609, 202610, 202611} {
		for _, beta := range []bool{false, true} {
			a := New(Config{PhysW: 320, PhysH: 240, AccessMonth: month, ShowBetaFeatures: beta}, data.Ingest(nil, "", time.Now()), nil)
			a.openOptions()
			expandOptionsForTest(a)
			for i, e := range a.panel.entries {
				if e.kind == "your-access" {
					a.panel.cursor = i
					break
				}
			}
			a.actPanel(platform.KeyEnter)
			if a.screen != ScreenAccess {
				t.Fatal("overview unavailable")
			}
			for _, f := range access.Catalog() {
				found := -1
				for i, e := range a.panel.entries {
					if e.value == f.ID {
						found = i
						break
					}
				}
				if !f.Fancy && !beta {
					if found >= 0 {
						t.Fatal("free beta shown with toggle off")
					}
					continue
				}
				if found < 0 {
					t.Fatal("missing visible feature", f.ID)
				}
				a.panel.cursor = found
				a.buildPanel()
				row := a.panel.entries[found]
				if row.value != f.ID {
					t.Fatal("rebuild moved selection")
				}
				want := "Available"
				if f.Fancy {
					want = "Needs Oct 2026+"
					if f.Covered(month) {
						want = "Unlocked"
						if !beta {
							want = "Beta off"
						}
					}
				}
				if row.accessStatus != want {
					t.Fatalf("%s: got %q want %q", f.ID, row.accessStatus, want)
				}
				if !f.Fancy && !strings.Contains(row.help, "No code needed") {
					t.Fatal("free beta explanation missing")
				}
				if f.Fancy && f.Beta && !beta && !strings.Contains(row.help, "Enable Beta") {
					t.Fatal("supporter beta requirement missing")
				}
			}
			a.panel.cursor = 1
			a.actPanel(platform.KeyEnter)
			a.actLock(platform.KeyBack)
			if a.screen != ScreenAccess || a.panel.cursor != 1 {
				t.Fatal("code back lost overview")
			}
			a.actPanel(platform.KeyBack)
			if a.screen != ScreenOptions || a.panel.entries[a.panel.cursor].kind != "your-access" {
				t.Fatal("overview back lost options row")
			}
		}
	}
}

// Dates must not crowd out the feature explanation on a narrow tate screen.
func TestLockedDescriptionsFit(t *testing.T) {
	for _, month := range []access.Month{0, 202609} {
		a := New(Config{PhysW: 320, PhysH: 240, Rotation: gfx.RotLeft, SafeInsetX: 40, SafeInsetY: 40, ShowBetaFeatures: true, AccessMonth: month}, data.Ingest(nil, "", time.Now()), nil)
		for _, row := range a.optionsEntries() {
			if row.requiredMonth != 0 && len(gfx.Wrap(row.help, a.sm.Cols(a.lay.Body.Dx()-6), 100)) > 4 {
				t.Fatalf("%s explanation is truncated: %s", row.text, row.help)
			}
		}
	}
}

func TestForgetCodeRequiresFreshContinuousHold(t *testing.T) {
	now := time.Unix(1800000000, 0)
	calls := 0
	fail := false
	a := New(Config{PhysW: 320, PhysH: 240, AccessMonth: 202610, ShowBetaFeatures: true, TimerNow: func() time.Time { return now }, ForgetCode: func() (access.Month, error) {
		calls++
		if fail {
			return 202610, errors.New("disk error")
		}
		return 0, nil
	}}, data.Ingest(nil, "", now), nil)
	a.openAccess()
	a.panel.cursor = len(a.panel.entries) - 1
	key := func(k platform.Key, pressed bool) { a.Handle(platform.Event{Key: k, Pressed: pressed, At: now}) }
	key(platform.KeyEnter, true)
	now = now.Add(3 * time.Second)
	a.Tick(now)
	if calls != 0 || a.forget == nil {
		t.Fatal("opening press confirmed deletion")
	}
	key(platform.KeyEnter, false)
	key(platform.KeyEnter, true)
	now = now.Add(time.Second)
	a.Tick(now)
	key(platform.KeyEnter, false)
	now = now.Add(3 * time.Second)
	a.Tick(now)
	if calls != 0 {
		t.Fatal("short hold deleted code")
	}
	key(platform.KeyBack, true)
	key(platform.KeyBack, false)
	if a.forget != nil || a.cfg.AccessMonth != 202610 {
		t.Fatal("cancel changed access")
	}
	a.openForgetCode()
	fail = true
	key(platform.KeyEnter, true)
	now = now.Add(2 * time.Second)
	a.Tick(now)
	if a.forget == nil || a.cfg.AccessMonth != 202610 || a.forget.message == "" {
		t.Fatal("failure claimed success")
	}
	key(platform.KeyEnter, false)
	fail = false
	key(platform.KeyEnter, true)
	now = now.Add(2 * time.Second)
	key(platform.KeyEnter, false)
	if a.forget != nil || a.cfg.AccessMonth != 0 || !a.cfg.ShowBetaFeatures || a.screen != ScreenAccess {
		t.Fatal("forget did not apply free access in place")
	}
	for _, e := range a.panel.entries {
		if e.kind == "forget-code" {
			t.Fatal("forget row retained without access")
		}
	}
}

func TestAccessColumnsFallbackAndROMChild(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240, AccessMonth: 202610, ShowBetaFeatures: true}, data.Ingest(nil, "", time.Now()), nil)
	rows := a.accessEntries()
	if rows[3].accessStatus != "Unlocked" {
		t.Fatal("wide view not in columns")
	}
	a.lay.Body = image.Rect(0, 0, 100, 200)
	rows = a.accessEntries()
	if rows[3].accessStatus != "" || !rows[4].info || rows[4].text != "Unlocked" {
		t.Fatal("narrow view truncated instead of stacking")
	}
	for _, month := range []access.Month{0, 202610} {
		a.cfg.AccessMonth = month
		rows = a.optionsEntries()
		for i, e := range rows {
			if strings.HasPrefix(e.text, "ROM report") {
				if !e.child || i == 0 || rows[i-1].kind != "rescan" {
					t.Fatal("ROM report not under Rescan")
				}
			}
		}
	}
}
