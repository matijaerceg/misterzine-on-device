package gfx

import (
	"image"
	"image/color"
)

// ground is the picture Fill paints in place of one colour: a faint
// pattern behind the whole app where the flat ground colour would be.
var ground struct {
	col color.RGBA
	img *image.RGBA
}

// SetGround makes every Fill in col copy img's pixels at the same place
// instead; a nil img goes back to flat fills. img covers the logical canvas.
func SetGround(col color.RGBA, img *image.RGBA) {
	ground.col, ground.img = col, img
}
