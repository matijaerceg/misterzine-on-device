//go:build linux

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/fetch"
	"github.com/matijaerceg/misterzine-on-device/internal/report"
	"github.com/matijaerceg/misterzine-on-device/internal/scan"
)

// Send a report writes the card copy whatever happens, uploads it, and
// hands the outcome back on the UI goroutine: the code when the service took
// it, why not and where the copy is when it did not.
func TestSendReportSavesAndUploads(t *testing.T) {
	h := backgroundHost(t)
	os.MkdirAll(filepath.Join(h.card, "_Arcade", "cores"), 0755)
	os.WriteFile(filepath.Join(h.card, "_Arcade", "cores", "Gigandes_baz.rbf"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(h.root, "log.txt"), []byte("12:00:00 fetch https://misterzine.fyi/meta.json?t=9 ok\n12:00:01 scan: 1 local rows\n"), 0644)
	h.iniLine = "MiSTer.ini alt=0 found=true"
	h.scanDiag = &scanDiag{lines: []string{"cores: 1"}, files: []report.File{{Path: "_Arcade/Odd.mra", Reason: "no setname"}}}

	var got []byte
	status := http.StatusCreated
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		w.WriteHeader(status)
		if status == http.StatusCreated {
			io.WriteString(w, `{"code":"K7M4"}`)
		}
	}))
	defer srv.Close()
	old := fetch.ReportService
	fetch.ReportService = srv.URL
	defer func() { fetch.ReportService = old }()

	send := func() report.Outcome {
		t.Helper()
		out := make(chan report.Outcome, 1)
		h.sendReport(report.AppPart{View: "updated", Local: []report.LocalGame{{K: "local:gigandes", Title: "Gigandes (World bazset)"}}},
			func(o report.Outcome) { out <- o })
		deadline := time.After(10 * time.Second)
		for {
			select {
			case f := <-h.uiRun:
				f()
			case o := <-out:
				return o
			case <-deadline:
				t.Fatal("no outcome")
			}
		}
	}

	o := send()
	if o.Code != "K7M4" || o.Saved != filepath.Base(h.root)+"/report.txt" || o.Problem != "" {
		t.Fatalf("outcome %+v", o)
	}
	saved, err := os.ReadFile(filepath.Join(h.root, "report.txt"))
	if err != nil || string(saved) != string(got) {
		t.Fatalf("the card copy is not what was sent: %v", err)
	}
	text := string(got)
	for _, want := range []string{report.Magic, "INI: MiSTer.ini alt=0 found=true", "local:gigandes | Gigandes (World bazset)",
		"_Arcade/Odd.mra | no setname", "Gigandes_baz.rbf", "scan: 1 local rows", "meta.json?..."} {
		if !strings.Contains(text, want) {
			t.Errorf("report lacks %q", want)
		}
	}
	if strings.Contains(text, "t=9") {
		t.Error("a query string left the card")
	}

	status = http.StatusTooManyRequests
	os.Remove(filepath.Join(h.root, "report.txt"))
	o = send()
	if o.Code != "" || !strings.Contains(o.Problem, "Try again in a minute") || o.Saved == "" {
		t.Fatalf("refused outcome %+v", o)
	}
	if _, err := os.Stat(filepath.Join(h.root, "report.txt")); err != nil {
		t.Fatal("no card copy when the service refused")
	}

	srv.Close()
	o = send()
	if o.Code != "" || o.Problem != "No connection to the report service." || o.Saved == "" {
		t.Fatalf("offline outcome %+v", o)
	}
}

// The report's card scan section names what the scan found listed but gone,
// folders with a slash, other skips left out, and a long run cut short.
func TestGoneLines(t *testing.T) {
	files := []scan.Skipped{
		{Path: "_Arcade/_alternatives/_Ibara/Ibara.mra", Reason: scan.GoneReason, Gone: true},
		{Path: "_Arcade/_alternatives/_Game/empty.mra", Reason: "no XML content"},
	}
	dirs := []scan.Skipped{
		{Path: "_Arcade/_Extra/deeper", Reason: scan.GoneReason, Gone: true},
		{Path: "_Arcade/_Organized", Reason: "an organiser's folder"},
	}
	got := goneLines(files, dirs)
	want := []string{"listed but gone: _Arcade/_alternatives/_Ibara/Ibara.mra", "listed but gone: _Arcade/_Extra/deeper/"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("lines %q", got)
	}
	if got := goneLines(nil, nil); len(got) != 0 {
		t.Fatalf("nothing gone: %q", got)
	}
	var many []scan.Skipped
	for i := 0; i < maxGoneLines+5; i++ {
		many = append(many, scan.Skipped{Path: fmt.Sprintf("_Arcade/g%02d.mra", i), Gone: true})
	}
	got = goneLines(many, nil)
	if len(got) != maxGoneLines+1 || got[maxGoneLines] != "... and 5 more listed but gone" {
		t.Fatalf("long run: %d lines, last %q", len(got), got[len(got)-1])
	}
}
