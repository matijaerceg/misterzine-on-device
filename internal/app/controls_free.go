//go:build !arcade

package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/controls"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"time"
)

func (a *App) discoverControls(time.Time) bool { return false }
func (a *App) nextControlDiscovery() time.Time { return time.Time{} }
func (a *App) OpenControlLabels() {}

type controlsView struct{}
type controlsInput struct{}

func (a *App) controlEvent(ev platform.Event) (platform.Event, bool, bool) {
	if ev.ObservationOnly { return ev, true, false }; return a.padEvent(ev), false, false
}
func (a *App) controlsChanged()                            {}
func (a *App) OpenControls()                               {}
func (a *App) tickControls(time.Time) bool                 { return false }
func (a *App) nextControlsTick() time.Time                 { return time.Time{} }
func (a *App) paintControls(*gfx.Canvas)                   {}
func (a *App) controlProfile(string) controls.Profile      { return controls.Profile{} }
func (a *App) controlLegend(s string) (string, bool)       { return s, false }
func (a *App) controlsOptions(e []panelEntry) []panelEntry { return e }
