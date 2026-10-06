//go:build linux

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFrontendShortcutMigration(t *testing.T) {
	p := filepath.Join(t.TempDir(), "MisterZine Arcade.mgl")
	for _, core := range []string{"", zaparooMenu, zaparooMenu, "degauss/menu.rbf", ""} {
		body := menuEntryBody(core)
		if err := ensureMGLBodyAt(p, body); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(p)
		if err != nil || string(b) != body || !strings.Contains(string(b), "<setname>misterzine</setname>") {
			t.Fatalf("entry: %s %v", b, err)
		}
		before, _ := os.Stat(p)
		if err := ensureMGLBodyAt(p, body); err != nil {
			t.Fatal(err)
		}
		after, _ := os.Stat(p)
		if !before.ModTime().Equal(after.ModTime()) {
			t.Fatal("unchanged entry rewritten")
		}
	}
}

func TestFrontendMenuRequiresConfiguredMainAndCore(t *testing.T) {
	for _, c := range []struct {
		main   string
		exists bool
		want   string
	}{
		{"", true, ""}, {"MiSTer", true, ""},
		{"zaparoo/MiSTer_Zaparoo", true, zaparooMenu},
		{"zaparoo/MiSTer_Zaparoo", false, ""},
		{"degauss/MiSTer_Degauss", true, "degauss/menu.rbf"},
		{"degauss/MiSTer_Degauss", false, ""},
		{"/media/fat/degauss/MiSTer_Degauss", true, "degauss/menu.rbf"},
		{"ConsoleMode/MiSTer_ConsoleMode", true, ""},
	} {
		got := frontendMenuCore(c.main, func(path string) bool {
			if c.want != "" && path != "/media/fat/"+c.want {
				t.Fatalf("wrong core checked: %s", path)
			}
			return c.exists
		})
		if got != c.want {
			t.Fatalf("%+v: %q", c, got)
		}
		body := menuEntryBody(got)
		want := strings.TrimSuffix(c.want, ".rbf")
		if want == "" {
			want = "menu"
		}
		if !strings.Contains(body, "<rbf>"+want+"</rbf>") {
			t.Fatalf("wrong shortcut: %s", body)
		}
	}
}

func TestConsoleLeaseAcknowledgementAndRelease(t *testing.T) {
	s := consoleState{123, "ready", "-"}
	var sent []string
	l := consoleLease{123, "test-1", func() consoleState { return s }, func(cmd string) error {
		sent = append(sent, cmd)
		if strings.Contains(cmd, "acquire") {
			s = consoleState{123, "acquired", "test-1"}
		} else {
			s.status = "released"
		}
		return nil
	}, func(time.Duration) {}}
	if err := l.acquire(); err != nil {
		t.Fatal(err)
	}
	if err := l.release(); err != nil {
		t.Fatal(err)
	}
	if len(sent) != 2 || sent[0] != "zaparoo_console acquire test-1 2" || sent[1] != "zaparoo_console release test-1" {
		t.Fatal(sent)
	}
}

func TestConsoleLeaseRejectsBusyStaleAndTimeout(t *testing.T) {
	for _, c := range []struct {
		name  string
		state consoleState
		sends int
	}{
		{"busy", consoleState{123, "busy", "ours"}, 1},
		{"failed", consoleState{123, "failed", "ours"}, 1},
		{"restarted Main", consoleState{456, "acquired", "ours"}, 1},
		{"other nonce", consoleState{123, "acquired", "theirs"}, 2},
		{"no acknowledgement", consoleState{123, "ready", "-"}, 2},
	} {
		t.Run(c.name, func(t *testing.T) {
			sends := 0
			l := consoleLease{123, "ours", func() consoleState { return c.state }, func(string) error { sends++; return nil }, func(time.Duration) {}}
			if err := l.acquire(); err == nil {
				t.Fatal("accepted invalid acknowledgement")
			}
			if sends != c.sends {
				t.Fatalf("sent %d commands", sends)
			}
		})
	}
}

func TestConsoleReleaseDoesNotTouchNewMainOrReleasedLease(t *testing.T) {
	for _, s := range []consoleState{{456, "acquired", "ours"}, {123, "released", "ours"}} {
		l := consoleLease{123, "ours", func() consoleState { return s }, func(string) error { t.Fatal("touched another session"); return nil }, func(time.Duration) {}}
		if err := l.release(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestConsoleReleaseAfterAnotherClientWasRejected(t *testing.T) {
	s := consoleState{123, "busy", "theirs"}
	l := consoleLease{123, "ours", func() consoleState { return s }, func(cmd string) error {
		if cmd != "zaparoo_console release ours" {
			t.Fatal(cmd)
		}
		s = consoleState{123, "released", "ours"}
		return nil
	}, func(time.Duration) {}}
	if err := l.release(); err != nil {
		t.Fatal(err)
	}
	if s.status != "released" {
		t.Fatal("left our console leased after a competing request")
	}
}

func TestParseConsoleState(t *testing.T) {
	for _, b := range []string{"", "2 123 ready -", "1 x ready -", "1 -3 ready -", "1 123 ready - extra"} {
		if got := parseConsoleState([]byte(b)); got.pid != 0 {
			t.Fatalf("accepted %q", b)
		}
	}
	if got := parseConsoleState([]byte("1 123 acquired ours\n")); got != (consoleState{123, "acquired", "ours"}) {
		t.Fatal(got)
	}
}
