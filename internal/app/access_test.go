package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/access"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
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
	row := a.gatedOption(panelEntry{text: "Extra", kind: "extra"}, f)
	if row.kind != "enter-code" || !row.disabled {
		t.Fatal("locked extra lacks code entry")
	}
}
