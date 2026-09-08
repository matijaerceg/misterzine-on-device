package updater

import (
	"strings"
	"testing"
	"time"
)

func TestStagesAndEarlyRestart(t *testing.T) {
	s := State{Status: "running"}
	now := time.Now()
	for _, tc := range []struct {
		line              string
		stage             int
		protected, reboot bool
	}{
		{"Launching Update All", 0, false, false},
		{"Running MiSTer Downloader", 1, false, false},
		{"Downloading _Arcade/example.rbf", 2, false, false},
		{"Linux will be updated from distribution_mister:", 2, true, true},
		{"Hold your breath: updating the Kernel", 2, true, true},
		{"Linux has been updated! It is recommended to reboot your system now.", 2, false, true},
		{"Running Arcade Organizer", 3, false, true},
		{"Running MiSTer Downloader", 3, false, true},
		{"Success! Log saved.", 4, false, true},
	} {
		s.output(tc.line, now)
		if s.Stage != tc.stage || s.Protected != tc.protected || s.Reboot != tc.reboot {
			t.Fatalf("%q: %+v", tc.line, s)
		}
	}
	if !s.SawSuccess {
		t.Fatal("missed final success")
	}
}

func TestLogBoundsAndSanitization(t *testing.T) {
	s := State{}
	for i := 0; i < 1000; i++ {
		s.output("\x1b[32mDownloading core\x1b[0m", time.Now())
	}
	if len(s.Lines) != TailLines || s.Lines[0] != "Downloading core" {
		t.Fatal("log tail or terminal cleanup failed")
	}
	for _, line := range []string{"Authorization: Bearer topsecret", "password=topsecret", "https://example.com/f?token=topsecret"} {
		if got := cleanLine(line); strings.Contains(got, "topsecret") {
			t.Fatalf("secret in %q", got)
		}
	}
	if len(cleanLine(strings.Repeat("x", 10000))) > 512 {
		t.Fatal("unbounded line")
	}
}
