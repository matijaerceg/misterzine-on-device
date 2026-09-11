//go:build linux

package main

import (
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform/mister"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
	"os"
	"path/filepath"
	"testing"
)

func TestRotationRereadsIniAndRespectsToggle(t *testing.T) {
	p := filepath.Join(t.TempDir(), "MiSTer.ini")
	settings := store.DefaultSettings()
	settings.Rotation = "right"
	for _, tc := range []struct {
		content string
		want    gfx.Rotation
	}{
		{"osd_rotate=1\n[Menu]\nosd_rotate=2\n", gfx.RotLeft},
		{"osd_rotate=0\n", gfx.RotNone},
		{"[Menu]\nosd_rotate=1\n", gfx.RotRight},
	} {
		if err := os.WriteFile(p, []byte(tc.content), 0600); err != nil {
			t.Fatal(err)
		}
		if got := startupRotation(settings, mister.ReadIni(p)); got != tc.want {
			t.Fatalf("got %v want %v", got, tc.want)
		}
	}
	settings.FollowRotation = false
	settings.Rotation = "left"
	if startupRotation(settings, mister.ReadIni(p)) != gfx.RotLeft {
		t.Fatal("off ignored manual rotation")
	}
	settings.FollowRotation = true
	if startupRotation(settings, mister.ReadIni(p+".absent")) != gfx.RotLeft {
		t.Fatal("missing INI discarded manual fallback")
	}
}
