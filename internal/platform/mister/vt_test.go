package mister

import (
	"testing"
	"time"
)

func TestAwaitConsoleReturnsOnceTheSwitchLands(t *testing.T) {
	seen := []string{"tty2", "tty2", "tty3"}
	i := 0
	active := func() string {
		if i < len(seen)-1 {
			i++
			return seen[i-1]
		}
		return seen[len(seen)-1]
	}
	var slept time.Duration
	if !awaitConsole("tty3", active, func(d time.Duration) { slept += d }, 2*time.Second) {
		t.Fatal("switch landed but awaitConsole said no")
	}
	if slept != 100*time.Millisecond {
		t.Fatalf("slept %v, want two polls", slept)
	}
}

// Main stuck after F9, as /proc/<pid>/syscall showed it on the DE10, and
// the lines that must not read as a stuck wait.
func TestVTWaitTarget(t *testing.T) {
	for _, c := range []struct {
		line string
		n    int
		ok   bool
	}{
		{"54 0xc 0x5607 0x1 0x0 0x264478 0xc 0xb37fdea4 0xb6ebc188\n", 1, true},
		{"54 0xc 0x5607 0x2 0x0 0x0 0x0 0xbe8 0xb6e", 2, true},
		{"168 0x16add4 0x21 0x19 0x0 0x19 0x21 0xb38feb10 0xb6ea5766", 0, false}, // poll: Main at work
		{"54 0xc 0x5606 0x1 0x0 0x0 0x0 0x0 0x0", 0, false},                      // VT_ACTIVATE
		{"54 0xc 0x5607 0x0 0x0 0x0 0x0 0x0 0x0", 0, false},
		{"running", 0, false},
		{"", 0, false},
	} {
		if n, ok := VTWaitTarget(c.line); n != c.n || ok != c.ok {
			t.Errorf("VTWaitTarget(%q) = %d, %v; want %d, %v", c.line, n, ok, c.n, c.ok)
		}
	}
}

func TestAwaitConsoleGivesUpWhenTheKernelDropsTheSwitch(t *testing.T) {
	var slept time.Duration
	if awaitConsole("tty3", func() string { return "tty2" }, func(d time.Duration) { slept += d }, 2*time.Second) {
		t.Fatal("console never switched but awaitConsole said yes")
	}
	if slept < 2*time.Second || slept > 2*time.Second+50*time.Millisecond {
		t.Fatalf("slept %v, want about the 2s timeout", slept)
	}
}
