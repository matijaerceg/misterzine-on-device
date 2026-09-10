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
