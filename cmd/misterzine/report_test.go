//go:build linux

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
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
	os.WriteFile(filepath.Join(h.root, "watch.log"), []byte("11:58:00 watch: started, pid 812\n11:59:00 watch: could not open the console (F9 pressed 20 times, active tty3)\n"), 0644)
	os.WriteFile(filepath.Join(h.card, "MisterZine Arcade.mgl"), []byte(mglBody), 0644)
	h.iniLine = "MiSTer.ini alt=0 found=true"
	h.iniMain = "ConsoleMode/MiSTer_ConsoleMode"
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
		"_Arcade/Odd.mra | no setname", "Gigandes_baz.rbf", "scan: 1 local rows", "meta.json?...",
		"== MENU ENTRY\nEntry: MisterZine Arcade.mgl present\n", "Main: ConsoleMode/MiSTer_ConsoleMode (main= in MiSTer.ini)",
		"== WATCH LOG (last 2 lines)\n11:58:00 watch: started, pid 812\n11:59:00 watch: could not open the console"} {
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

// The report's MENU ENTRY section: a card that never ran Setup, then one
// with the entry under both names, a startup script saved with Windows line
// endings, a helper that left its pid behind and a replacement Main, and
// last a helper that is running.
func TestMenuEntryStatus(t *testing.T) {
	card, root := t.TempDir(), t.TempDir()
	got := strings.Join(menuEntryStatus(card, root, "", ""), "\n")
	want := "Entry: MisterZine Arcade.mgl missing (MisterZine-Setup writes it)\nStartup line: no linux/user-startup.sh\n" +
		"Helper: not running\nMain: the stock MiSTer\nCore name now: none"
	if got != want {
		t.Fatalf("fresh card:\n%s", got)
	}

	os.WriteFile(filepath.Join(card, "MisterZine Arcade.mgl"), []byte(mglBody), 0644)
	os.WriteFile(filepath.Join(card, "MisterZine.mgl"), []byte(mglBody), 0644)
	os.MkdirAll(filepath.Join(card, "linux"), 0755)
	os.WriteFile(filepath.Join(card, "linux", "user-startup.sh"), []byte("#!/bin/sh\r\n\r\n"+startupMark+"\r\n"+startupLine+"\r\n"), 0755)
	os.WriteFile(filepath.Join(root, "watch.pid"), []byte("999999\n"), 0644)
	got = strings.Join(menuEntryStatus(card, root, "misterzine", "ConsoleMode/MiSTer_ConsoleMode"), "\n")
	want = "Entry: MisterZine Arcade.mgl present\nOld entry: MisterZine.mgl still present\n" +
		"Startup line: in linux/user-startup.sh (the file has Windows line endings)\n" +
		"Helper: not running (watch.pid names pid 999999)\nMain: ConsoleMode/MiSTer_ConsoleMode (main= in MiSTer.ini)\nCore name now: misterzine"
	if got != want {
		t.Fatalf("set-up card:\n%s", got)
	}

	// the helper is known by "watch" on its command line; the trailing
	// true keeps sh from exec'ing sleep in its place
	helper := exec.Command("sh", "-c", "sleep 30; true", "watch")
	if err := helper.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { helper.Process.Kill(); helper.Wait() }()
	// Start returns once exec is under way, a moment before the kernel sets
	// the new command line: /proc reads it empty until then
	cmdline := fmt.Sprintf("/proc/%d/cmdline", helper.Process.Pid)
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if b, _ := os.ReadFile(cmdline); strings.Contains(string(b), "watch") {
			break
		}
	}
	os.WriteFile(filepath.Join(root, "watch.pid"), []byte(strconv.Itoa(helper.Process.Pid)), 0644)
	if lines := menuEntryStatus(card, root, "MENU", ""); !slices.Contains(lines, fmt.Sprintf("Helper: running, pid %d", helper.Process.Pid)) {
		t.Fatalf("running helper:\n%s", strings.Join(lines, "\n"))
	}
}

// The watch log tail reads the rotated file first and skips empty files.
func TestWatchLogTail(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "watch.1.log"), []byte("a\nb\n"), 0644)
	os.WriteFile(filepath.Join(root, "watch.log"), []byte("c\nd\n"), 0644)
	if got := strings.Join(watchLogTail(root, 3), ","); got != "b,c,d" {
		t.Fatalf("tail %q", got)
	}
	os.WriteFile(filepath.Join(root, "watch.log"), nil, 0644)
	if got := strings.Join(watchLogTail(root, 3), ","); got != "a,b" {
		t.Fatalf("empty current file: %q", got)
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
