//go:build linux

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/buildinfo"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/platform/mister"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

type supportHost struct {
	loaded, launching bool
	report            support.Report
	capture           *support.Capture
	probe             *mister.InputProbe
	until             time.Time
}

func (h *host) supportHooks() *app.SupportHooks {
	return &app.SupportHooks{Start: h.startSupport, Finish: func() support.Report { return h.finishSupport(false) },
		Load: h.loadSupport, Launch: h.testSupportLaunch}
}

func (h *host) loadSupport() support.Report {
	s := &h.troubleshooting
	if !s.loaded {
		s.loaded = true
		f, err := os.Open(filepath.Join(h.root, "troubleshooting.json"))
		if err == nil {
			defer f.Close()
			var r support.Report
			err = json.NewDecoder(io.LimitReader(f, 128<<10)).Decode(&r)
			if err == nil && r.Schema == 1 && len(r.Devices) <= support.MaxDevices && len(r.Signals) <= support.MaxSignals {
				s.report = r
			} else {
				s.report.Note = "Saved result unreadable. Run a new test."
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			s.report.Note = "Could not read saved result: " + err.Error()
		}
	}
	return s.report
}

func (h *host) newSupportReport() support.Report {
	r := support.Report{Schema: 1, Version: buildinfo.String(), Created: time.Now().UTC().Format(time.RFC3339)}
	var u syscall.Utsname
	if syscall.Uname(&u) == nil {
		var b []byte
		for _, ch := range u.Release {
			if ch == 0 {
				break
			}
			b = append(b, byte(ch))
		}
		r.Kernel = string(b)
	}
	if h.a != nil {
		x, y := h.a.Inset()
		r.Display = fmt.Sprintf("Rotation %s; inset %d,%d", h.a.Rotation(), x, y)
		r.Data = h.a.Data().Updated.UTC().Format("2006-01-02 15:04")
	}
	if h.fb != nil {
		r.Display += "; " + h.fb.Geometry().String()
	}
	return r
}

func (h *host) startSupport(from, until time.Time) {
	if h.troubleshooting.capture != nil {
		h.finishSupport(true)
	}
	s := &h.troubleshooting
	s.loaded, s.launching = true, false
	s.capture = support.NewCapture(h.newSupportReport(), from, until)
	s.until = until
	s.probe = mister.OpenInputProbe(s.capture, until)
}

func (h *host) finishSupport(interrupted bool) support.Report {
	s := &h.troubleshooting
	if s.capture == nil {
		return h.loadSupport()
	}
	if s.probe != nil {
		s.probe.Close()
		s.probe = nil
	}
	s.report = s.capture.Snapshot()
	s.report.Interrupted = interrupted && time.Now().Before(s.until)
	s.capture = nil
	h.saveSupport()
	return s.report
}

func (h *host) recordSupportInput(ev platform.Event) {
	if c := h.troubleshooting.capture; c != nil && ev.Key == platform.KeyStart && ev.Source != "debug" && ev.Source != "script" {
		c.RecordStart(ev.Pressed, time.Now())
	}
}

func (h *host) saveSupport() {
	if err := store.Save(filepath.Join(h.root, "troubleshooting.json"), h.troubleshooting.report); err != nil {
		h.troubleshooting.report.Note = "Result not saved; take a photo before closing. " + err.Error()
		h.lg.Printf("troubleshooting save: %v", err)
	}
}

func (h *host) testSupportLaunch(game, target string) support.Report {
	r := h.loadSupport()
	if !r.HasResult() || r.Schema != 1 {
		r = h.newSupportReport()
	}
	r.Launch = support.Launch{Game: game, Target: target, Result: "Checking launch target", Version: buildinfo.String()}
	h.troubleshooting.report = r
	h.troubleshooting.launching = true
	h.requestLaunch(target)
	return h.troubleshooting.report
}

// Only explicit support launches write a diagnostic result. Normal launches
// keep their existing behavior and do not create a support report.
func (h *host) supportLaunchResult(result string, err error) {
	if !h.troubleshooting.launching {
		return
	}
	r := &h.troubleshooting.report
	r.Launch.Result = result
	r.Launch.Detail = ""
	if err != nil {
		r.Launch.Detail = err.Error()
		h.troubleshooting.launching = false
	}
	h.saveSupport()
}
