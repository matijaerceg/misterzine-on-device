//go:build linux

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/buildinfo"
	"github.com/matijaerceg/misterzine-on-device/internal/fetch"
	"github.com/matijaerceg/misterzine-on-device/internal/report"
	"github.com/matijaerceg/misterzine-on-device/internal/scan"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
)

// sendReport is Troubleshooting -> Send a report. What only the UI goroutine
// may read (the app's part, settings, the last scan, the system lines) is
// gathered here; the card walk, the log, the file and the upload run off it,
// and done comes back on it. The card copy is always written, so a player
// without a connection still has a file to send.
func (h *host) sendReport(part report.AppPart, done func(report.Outcome)) {
	if !h.reportSending.CompareAndSwap(false, true) {
		done(report.Outcome{Problem: "A report is already on its way.", SaveErr: "the report is still being written"})
		return
	}
	in := report.Input{Version: buildinfo.String(), Created: time.Now(), App: part}
	sys := h.newSupportReport()
	in.System = []string{"Kernel: " + sys.Kernel, "Display: " + sys.Display, "Catalogue: " + sys.Data, "INI: " + h.iniLine}
	if b, err := json.MarshalIndent(h.settings, "", " "); err == nil {
		in.Settings = string(b)
	}
	if d := h.scanDiag; d != nil {
		in.Scan = append([]string(nil), d.lines...)
		in.Files = append([]report.File(nil), d.files...)
	} else {
		in.Scan = []string{"no card scan has finished yet"}
	}
	card, root, client := h.card, h.root, h.client
	go func() {
		if model := boardModel(); model != "" {
			in.System = append([]string{"Board: " + model}, in.System...)
		}
		in.Layout = scan.CardLayout(card, root)
		in.Log = logTail(root, report.MaxLogLines)
		body := report.Build(in)

		var o report.Outcome
		if err := store.WriteAtomic(filepath.Join(root, "report.txt"), body); err != nil {
			o.SaveErr = err.Error()
		} else {
			o.Saved = filepath.Base(root) + "/report.txt"
		}
		code, err := client.SendReport(context.Background(), body)
		if err != nil {
			o.Problem = reportProblem(err)
			h.lg.Printf("report: not sent (%v); card copy %q %s", err, o.Saved, o.SaveErr)
		} else {
			o.Code = code
			h.lg.Printf("report: sent as %s (%d bytes); card copy %q %s", report.DisplayCode(code), len(body), o.Saved, o.SaveErr)
		}
		// free before the screen can offer another send
		h.reportSending.Store(false)
		h.runOnUI(func() { done(o) })
	}()
}

// reportProblem words a failed upload for the player.
func reportProblem(err error) string {
	switch {
	case errors.Is(err, fetch.ErrOffline):
		return "No connection to the report service."
	case errors.Is(err, fetch.ErrReportBusy):
		return "Too many reports at once. Try again in a minute."
	case errors.Is(err, fetch.ErrReportOff):
		return "The report service is switched off at the moment."
	case errors.Is(err, fetch.ErrReportTooLarge):
		return "The report is too large to send."
	}
	return "The report service did not take it (" + err.Error() + ")."
}

// boardModel is the device tree's model string: which board this is.
func boardModel() string {
	b, err := os.ReadFile("/proc/device-tree/model")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(bytes.TrimRight(b, "\x00")))
}

// logTail is the last n lines of the app's log, reaching into the rotated
// file when the current one is short.
func logTail(root string, n int) []string {
	var lines []string
	for _, name := range []string{"log.1.txt", "log.txt"} {
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		lines = append(lines, strings.Split(strings.TrimRight(string(b), "\n"), "\n")...)
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}
