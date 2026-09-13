package app

import (
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// X opens Filters from the list and a second X returns to the list, the
// row under the cursor unchanged; years still expand and collapse with
// Right and Left.
func TestSecondXClosesFilters(t *testing.T) {
	a, _, _, tap := menuApp()
	tap(platform.KeyDown)
	row := a.CursorKey()
	tap(platform.KeyTab)
	if a.screen != ScreenFilter {
		t.Fatalf("X on the list: screen %v", a.screen)
	}
	tap(platform.KeyDown)
	tap(platform.KeyTab)
	if a.screen != ScreenList || a.CursorKey() != row {
		t.Fatalf("X on Filters: screen %v, cursor on %q, want the list on %q", a.screen, a.CursorKey(), row)
	}
	tap(platform.KeyTab)
	if a.screen != ScreenFilter {
		t.Fatalf("X again: screen %v", a.screen)
	}
}
