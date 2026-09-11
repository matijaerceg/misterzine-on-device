//go:build linux

package main

import (
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform/mister"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
)

func startupRotation(settings store.Settings, ini mister.IniSettings) gfx.Rotation {
	if settings.FollowRotation && ini.Found || settings.Rotation == "auto" {
		switch ini.OSDRotate {
		case 1:
			return gfx.RotRight
		case 2:
			return gfx.RotLeft
		default:
			return gfx.RotNone
		}
	}
	switch settings.Rotation {
	case "left":
		return gfx.RotLeft
	case "right":
		return gfx.RotRight
	default:
		return gfx.RotNone
	}
}

func iniOrientation(ini mister.IniSettings) string {
	if !ini.Found {
		return ""
	}
	switch ini.OSDRotate {
	case 0:
		return "h"
	case 1, 2:
		return "v"
	}
	return ""
}
