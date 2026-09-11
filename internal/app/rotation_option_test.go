package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"testing"
	"time"
)

func TestDisablingFollowKeepsCurrentOrientation(t *testing.T) {
	saved := ""
	changes := 0
	a := New(Config{PhysW: 320, PhysH: 240, Rotation: gfx.RotLeft, FollowRotation: true,
		Action: func(kind, arg string) {
			if kind == "rotation" {
				saved = arg
			}
		},
		SettingsChanged: func() { changes++ },
	}, data.Ingest(nil, "", time.Now()), nil)
	a.openPanel(ScreenOptions)
	for i, e := range a.panel.entries {
		if e.kind == "follow-rotation" {
			a.panel.cursor = i
			break
		}
	}
	a.stepValue(-1)
	if a.FollowRotation() || saved != "left" || a.rot != gfx.RotLeft || changes != 1 {
		t.Fatal("turning off did not retain current rotation")
	}
	a.stepValue(1)
	if !a.FollowRotation() || a.rot != gfx.RotLeft || changes != 2 {
		t.Fatal("turning on must take effect next launch")
	}
}

func TestRotationRowDisabledWhileFollowing(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240, Rotation: gfx.RotNone, FollowRotation: true}, data.Ingest(nil, "", time.Now()), nil)
	a.openPanel(ScreenOptions)
	idx := map[string]int{}
	for i, e := range a.panel.entries {
		idx[e.kind] = i
	}
	// Rotation sits right under Follow INI rotation in Display; Filter by
	// rotation lives in the List group above them.
	if idx["follow-rotation"]+1 != idx["rotation"] || idx["filter-rotation"] > idx["follow-rotation"] {
		t.Fatal("order must be filter (List), then follow, rotation (Display)", idx)
	}
	if !a.panel.entries[idx["rotation"]].disabled {
		t.Fatal("rotation must be disabled while following")
	}
	a.panel.cursor = idx["rotation"]
	if a.stepValue(1) || a.rot != gfx.RotNone {
		t.Fatal("disabled rotation row must ignore Left/Right")
	}
	a.panel.cursor = idx["follow-rotation"]
	a.stepValue(-1)
	if a.panel.entries[idx["rotation"]].disabled {
		t.Fatal("rotation must be enabled once following is off")
	}
	a.panel.cursor = idx["rotation"]
	if !a.stepValue(1) || a.rot != gfx.RotLeft {
		t.Fatal("manual rotation must work when following is off")
	}
}
