// Package fonts embeds the bitmap fonts the UI draws with. Spleen is
// BSD-2-Clause (see SPLEEN-LICENSE) and scientifica is under the SIL Open
// Font License (see SCIENTIFICA-LICENSE); the BDF files are parsed once on
// first use, which costs under a millisecond.
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

//go:embed scientifica-11.bdf
var scientifica11 []byte

var (
	once6, once5, onceN sync.Once
	f6, f5, fN          *gfx.Font
)

// Body is the 6x12 body font: 53 columns on a 320 px canvas, 40 in tate.
func Body() *gfx.Font {
	once6.Do(func() {
		var err error
		if f6, err = gfx.ParseBDF(spleen6x12); err != nil {
			panic("fonts: spleen-6x12: " + err.Error())
		}
		f6.AddArrows()
	})
	return f6
}

// Narrow is scientifica at 11 px in a 5x12 cell: the list's narrow title
// font, drawn proportionally so a row fits about a third more title than
// the body font. Its baseline sits one pixel below the body font's.
func Narrow() *gfx.Font {
	onceN.Do(func() {
		var err error
		if fN, err = gfx.ParseBDF(scientifica11); err != nil {
			panic("fonts: scientifica-11: " + err.Error())
		}
		fN.AddArrows()
	})
	return fN
}

// Small is the 5x8 secondary font for pane text, captions and chips.
func Small() *gfx.Font {
	once5.Do(func() {
		var err error
		if f5, err = gfx.ParseBDF(spleen5x8); err != nil {
			panic("fonts: spleen-5x8: " + err.Error())
		}
		f5.AddArrows()
	})
	return f5
}
