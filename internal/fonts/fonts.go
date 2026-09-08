// Package fonts embeds the bitmap fonts the UI draws with. Spleen is
// BSD-2-Clause (see SPLEEN-LICENSE); the BDF files are parsed once on first
// use, which costs under a millisecond.
package fonts

import (
	_ "embed"
	"sync"

	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

//go:embed spleen-6x12.bdf
var spleen6x12 []byte

//go:embed spleen-5x8.bdf
var spleen5x8 []byte

var (
	once6, once5 sync.Once
	f6, f5       *gfx.Font
)

// Body is the 6x12 body font: 53 columns on a 320 px canvas, 40 in tate.
func Body() *gfx.Font {
	once6.Do(func() {
		var err error
		if f6, err = gfx.ParseBDF(spleen6x12); err != nil {
			panic("fonts: spleen-6x12: " + err.Error())
		}
	})
	return f6
}

// Small is the 5x8 secondary font for pane text, captions and chips.
func Small() *gfx.Font {
	once5.Do(func() {
		var err error
		if f5, err = gfx.ParseBDF(spleen5x8); err != nil {
			panic("fonts: spleen-5x8: " + err.Error())
		}
	})
	return f5
}
