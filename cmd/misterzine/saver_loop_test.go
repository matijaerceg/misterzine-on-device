//go:build linux

package main

import (
	"io"
	"log"
	"os"
	"syscall"
	"testing"
)

// A signal that arrives while the saver loop paces the frames ends the
// loop and stays in the channel for the main loop's own handling; the
// quit channel ends it too. Nothing pending lets the loop go on.
func TestSaverLoopPumpHandsSignalsToTheMainLoop(t *testing.T) {
	h := &host{quit: make(chan struct{}), sig: make(chan os.Signal, 2), lg: log.New(io.Discard, "", 0)}
	if !h.pump() {
		t.Fatal("an idle pump stopped the loop")
	}
	h.sig <- syscall.SIGTERM
	if h.pump() {
		t.Fatal("a signal did not end the saver loop")
	}
	select {
	case s := <-h.sig:
		if s != syscall.SIGTERM {
			t.Fatalf("signal %v handed back, want SIGTERM", s)
		}
	default:
		t.Fatal("the signal was swallowed instead of handed to the main loop")
	}
	close(h.quit)
	if h.pump() {
		t.Fatal("a closed quit channel did not end the saver loop")
	}
}
