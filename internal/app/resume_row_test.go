package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// The launcher reopens the app on the game that just ran; that row lands
// centred on the page like a jump, not pinned to the bottom edge.
func TestMoveToKeyCentresTheRow(t *testing.T) {
	now := time.Now()
	rows := make([]data.Row, 100)
	for i := range rows {
		rows[i] = data.Row{K: fmt.Sprint(i), Title: fmt.Sprintf("Game %03d", i), Base: "Arcade"}
	}
	a := New(Config{PhysW: 320, PhysH: 240, TimerNow: func() time.Time { return now }}, data.Ingest(rows, "", now), nil)
	a.Paint()
	a.MoveToKey("40")
	if a.cursor != 40 {
		t.Fatalf("cursor %d, want 40", a.cursor)
	}
	if want := centeredTop(a.screenLine(40), a.totalLines(), a.lay.Lines); a.top != want {
		t.Fatalf("top %d, want the row centred at %d", a.top, want)
	}
	if _, dirty := a.Paint(); len(dirty) == 0 {
		t.Fatal("the move did not repaint")
	}
}
