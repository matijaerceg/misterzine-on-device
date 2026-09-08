package app

import (
	"image"

	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// Layout is every rectangle of one orientation at one canvas size, computed
// once from the logical size and the safe-zone inset. Horizontal puts the
// pane beside the list; tate puts it below.
type Layout struct {
	W, H     int
	Inset    int
	Portrait bool
	Root     image.Rectangle // safe area
	Status   image.Rectangle // top bar
	Hint     image.Rectangle // bottom bar
	List     image.Rectangle
	Scroll   image.Rectangle // scrollbar track beside the list
	Pane     image.Rectangle
	Body     image.Rectangle // list + pane, for modal panels
	Line     int             // list line height (body font height)
	Lines    int             // list lines that fit
	Cols     int             // list columns (body font)
	TitleCol int             // title width in columns
	Thumb    image.Rectangle // thumbnail box inside the pane
	PaneText image.Rectangle // text area of the pane
}

const (
	statusH = 12
	hintH   = 12
	paneW   = 100 // horizontal: pane width
	paneH   = 78  // tate: pane height
	thumbW  = 96
	thumbH  = 72
)

// NewLayout computes the layout for a logical W x H canvas.
func NewLayout(w, h, inset int, body *gfx.Font) Layout {
	l := Layout{W: w, H: h, Inset: inset, Portrait: h > w, Line: body.H}
	l.Root = image.Rect(inset, inset, w-inset, h-inset)
	l.Status = image.Rect(l.Root.Min.X, l.Root.Min.Y, l.Root.Max.X, l.Root.Min.Y+statusH)
	l.Hint = image.Rect(l.Root.Min.X, l.Root.Max.Y-hintH, l.Root.Max.X, l.Root.Max.Y)
	l.Body = image.Rect(l.Root.Min.X, l.Status.Max.Y, l.Root.Max.X, l.Hint.Min.Y)
	if l.Portrait {
		l.Pane = image.Rect(l.Body.Min.X, l.Body.Max.Y-paneH, l.Body.Max.X, l.Body.Max.Y)
		l.List = image.Rect(l.Body.Min.X, l.Body.Min.Y, l.Body.Max.X, l.Pane.Min.Y-1)
		l.Thumb = image.Rect(l.Pane.Min.X, l.Pane.Min.Y+3, l.Pane.Min.X+thumbW, l.Pane.Min.Y+3+thumbH)
		l.PaneText = image.Rect(l.Thumb.Max.X+4, l.Pane.Min.Y+2, l.Pane.Max.X, l.Pane.Max.Y)
	} else {
		l.Pane = image.Rect(l.Body.Max.X-paneW, l.Body.Min.Y, l.Body.Max.X, l.Body.Max.Y)
		l.List = image.Rect(l.Body.Min.X, l.Body.Min.Y, l.Pane.Min.X-1, l.Body.Max.Y)
		tx := l.Pane.Min.X + (paneW-thumbW)/2
		l.Thumb = image.Rect(tx, l.Pane.Min.Y+2, tx+thumbW, l.Pane.Min.Y+2+thumbH)
		l.PaneText = image.Rect(l.Pane.Min.X+2, l.Thumb.Max.Y+3, l.Pane.Max.X, l.Pane.Max.Y)
	}
	// a 2 px scrollbar beside the rows
	l.Scroll = image.Rect(l.List.Max.X-2, l.List.Min.Y, l.List.Max.X, l.List.Max.Y)
	l.List.Max.X -= 4
	l.Lines = l.List.Dy() / l.Line
	l.Cols = l.List.Dx() / body.W
	// columns: fav(1) sp title sp status(1) sp date(5)
	l.TitleCol = l.Cols - 1 - 1 - 1 - 1 - 1 - 5
	if l.TitleCol < 8 {
		l.TitleCol = 8
	}
	return l
}

// lineRect is the rectangle of list line n (0-based from the top of the list).
func (l *Layout) lineRect(n int) image.Rectangle {
	y := l.List.Min.Y + n*l.Line
	return image.Rect(l.List.Min.X, y, l.List.Max.X, y+l.Line)
}
