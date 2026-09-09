package fonts

import (
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"testing"
)

func TestEllipsisMatchesPeriodBaseline(t *testing.T) {
	for _, f := range []*gfx.Font{Body(), Small()} {
		period, dots := -1, -1
		for y, row := range f.Glyph('.') {
			if row != 0 {
				period = y
			}
		}
		for y, row := range f.Glyph(gfx.Ellipsis[0]) {
			if row != 0 {
				dots = y
			}
		}
		if period < 0 || dots != period {
			t.Fatalf("%s: ellipsis row %d, period row %d", f.Name, dots, period)
		}
	}
}
