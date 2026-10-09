package scan

import (
	"testing"
	"time"
)

// A closed gate holds a walk between two steps; opening it lets the walk go
// on, and a nil gate or an open one never waits.
func TestGateHoldsAndReleases(t *testing.T) {
	var g Gate
	g.Wait() // open: returns at once
	(*Gate)(nil).Wait()
	(*Gate)(nil).SetPaused(true)
	g.SetPaused(true)
	g.SetPaused(true) // twice is once
	done := make(chan struct{})
	go func() {
		g.Wait()
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("waited through a closed gate")
	case <-time.After(50 * time.Millisecond):
	}
	g.SetPaused(false)
	g.SetPaused(false)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("still held after the gate opened")
	}
}

// The alternatives walk waits at the gate between game folders, so a held
// key can stop it mid-card and it finishes once the key is released.
func TestScanWaitsAtTheMotionGate(t *testing.T) {
	card := fakeCard(t)
	Motion.SetPaused(true)
	defer Motion.SetPaused(false)
	done := make(chan int)
	go func() {
		alts, _, _ := ScanAlternativesWithError(card, "")
		done <- len(alts)
	}()
	select {
	case <-done:
		t.Fatal("the walk ran through a closed gate")
	case <-time.After(50 * time.Millisecond):
	}
	Motion.SetPaused(false)
	select {
	case n := <-done:
		if n != 2 {
			t.Fatalf("alternatives after release: %d", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the walk never finished after the gate opened")
	}
}
