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

func TestAwaitConsoleGivesUpWhenTheKernelDropsTheSwitch(t *testing.T) {
	var slept time.Duration
	if awaitConsole("tty3", func() string { return "tty2" }, func(d time.Duration) { slept += d }, 2*time.Second) {
		t.Fatal("console never switched but awaitConsole said yes")
	}
	if slept < 2*time.Second || slept > 2*time.Second+50*time.Millisecond {
		t.Fatalf("slept %v, want about the 2s timeout", slept)
	}
}
