package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/access"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
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

// Only beta-only features promise free access after graduation.
func TestBetaOnlyEarlyAccessCopy(t *testing.T) {
	for _, supporter := range []bool{false, true} {
		a := New(Config{PhysW: 320, PhysH: 240, ShowBetaFeatures: true}, data.Ingest(nil, "", time.Now()), nil)
		f := access.Feature{Fancy: supporter, Beta: true, Since: 202610}
		row := a.gatedOption(panelEntry{text: "Example", kind: "example", help: "Explains the feature."}, f)
		if row.earlyAccess == supporter {
			t.Fatal("early access status conflated with Supporter")
		}
		if strings.Contains(row.help, "Free for everyone after beta.") == supporter {
			t.Fatal("wrong free-after-beta promise")
		}
		a.openOptions()
		a.panel.entries = []panelEntry{row}
		a.panel.cursor = 0
		a.actPanel(platform.KeyEnter)
		if a.lock == nil || a.lock.earlyAccess == supporter {
			t.Fatal("code screen lost feature status")
		}
		if strings.Contains(a.codeScreenTitle(), "Early access") == supporter {
			t.Fatal("wrong code screen title")
		}
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
	a.lock.featureTitle = "Example - Early access"
	a.lock.earlyAccess = true
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
	if a.lock.started() || a.lock.box != 0 || !a.lock.earlyAccess || a.lock.requiredMonth != 202709 {
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
			rows := a.panel.entries
			if len(rows) != 2+2*len(access.Catalog()) {
				t.Fatal("missing features")
			}
			for i, f := range access.Catalog() {
				a.panel.cursor = 2 + i*2
				a.buildPanel()
				if a.panel.entries[a.panel.cursor].value != f.ID {
					t.Fatal("rebuild moved feature selection")
				}
				state := rows[3+i*2].text
				if strings.HasPrefix(state, "Unlocked") != f.Covered(month) {
					t.Fatalf("wrong coverage: %v %s", month, state)
				}
				if strings.Contains(state, "beta off") != (f.Covered(month) && f.Beta && !beta) {
					t.Fatal("beta confused with access")
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
