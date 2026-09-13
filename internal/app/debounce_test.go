package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// A press that follows the same key's release by less than the debounce
// window is contact bounce and is dropped; a press the window later, or
// one stamped at the very instant of the release (scripted input), counts.
func TestBouncePressAfterReleaseIsDropped(t *testing.T) {
	now := time.Now()
	rows := []data.Row{{K: "one", Title: "One"}, {K: "two", Title: "Two"}, {K: "three", Title: "Three"}, {K: "four", Title: "Four"}}
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return now }}, data.Ingest(rows, "test", now), nil)
	press := func(at time.Duration) bool {
		return a.Handle(platform.Event{Key: platform.KeyDown, Pressed: true, At: now.Add(at)})
	}
	release := func(at time.Duration) { a.Handle(platform.Event{Key: platform.KeyDown, At: now.Add(at)}) }

	press(0)
	release(30 * time.Millisecond)
	if a.cursor != 1 {
		t.Fatalf("first tap: cursor %d", a.cursor)
	}
	if press(35*time.Millisecond) || a.cursor != 1 || a.down[platform.KeyDown] || a.Repeating() {
		t.Fatalf("a press 5 ms after the release was taken for a tap: cursor %d", a.cursor)
	}
	release(40 * time.Millisecond) // the bounce's own release changes nothing
	press(70 * time.Millisecond)   // 30 ms after the last release: a real tap
	if a.cursor != 2 {
		t.Fatalf("a press past the window was dropped: cursor %d", a.cursor)
	}
	release(100 * time.Millisecond)
	press(100 * time.Millisecond) // same instant as the release: scripted, not a switch
	if a.cursor != 3 {
		t.Fatalf("a same-instant press was dropped: cursor %d", a.cursor)
	}
	release(130 * time.Millisecond)
	// another key right after this one's release is not a bounce
	a.Handle(platform.Event{Key: platform.KeyUp, Pressed: true, At: now.Add(135 * time.Millisecond)})
	if a.cursor != 2 {
		t.Fatalf("a different key was dropped: cursor %d", a.cursor)
	}
}
