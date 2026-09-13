package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// A held Backspace waits the Hold delay like the navigation keys, then
// erases at the quick erase pace.
func TestHeldBackspaceErasesQuickly(t *testing.T) {
	now := time.Now()
	rows := []data.Row{{K: "one", Title: "Space Invaders"}, {K: "two", Title: "Invaders"}}
	a := New(Config{PhysW: 320, PhysH: 240, HoldDelay: 300, Now: func() time.Time { return now }}, data.Ingest(rows, "test", now), nil)
	for _, ch := range "inva" {
		a.Handle(platform.Event{Text: ch, Pressed: true, At: now})
	}
	a.Handle(platform.Event{Key: platform.KeyBackspace, Pressed: true, At: now})
	if a.Search() != "inv" || !a.NextTick().Equal(now.Add(300*time.Millisecond)) {
		t.Fatalf("press: %q, next tick %v", a.Search(), a.NextTick().Sub(now))
	}
	a.Tick(now.Add(299 * time.Millisecond))
	if a.Search() != "inv" {
		t.Fatal("repeated before the hold delay")
	}
	a.Tick(now.Add(300 * time.Millisecond))
	a.Tick(now.Add(300*time.Millisecond + repeatErase))
	a.Tick(now.Add(300*time.Millisecond + 2*repeatErase))
	if a.Search() != "" {
		t.Fatalf("after three erase steps: %q", a.Search())
	}
	a.Handle(platform.Event{Key: platform.KeyBackspace, At: now.Add(time.Second)})
	if a.Repeating() {
		t.Fatal("release did not stop the repeat")
	}
}
