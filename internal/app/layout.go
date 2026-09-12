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
	TitleW   int             // title width in pixels
	Thumb    image.Rectangle // thumbnail box inside the pane
	PaneText image.Rectangle // text area of the pane
	// Style is the list layout: "list" (the pane beside or below a full
	// list), "split" (a wider pane with a bigger picture) or "picture" (the
	// picture across the screen with a few rows beside it).
	Style string
	// PaneTop puts the pane above the list (picture layout, horizontal).
	PaneTop bool
	// TextBeside: the pane text follows the picture's drawn width, sitting
	// beside a picture that leaves room and below one that does not.
	TextBeside bool
}

const (
	statusH = 12
	hintH   = 12
	paneW   = 100 // horizontal: pane width
	paneH   = 78  // tate: pane height
	thumbW  = 96
	thumbH  = 72
	// paneTextH is the room under a full-width picture for three lines of
	// the small font
	paneTextH = 30
)

// listLayouts are the Options -> List layout choices.
var listLayouts = []string{"list", "split", "picture"}

// NewLayout computes the layout for a logical W x H canvas. The insets are
// the safe-zone margins as the viewer sees the picture: ix at the left and
// right edges, iy at the top and bottom (the logical canvas is already the
// viewer's orientation, so in tate ix is the tube's short side).
func NewLayout(w, h, ix, iy int, body *gfx.Font, rowW, dateCols int, style string) Layout {
	l := Layout{W: w, H: h, Inset: ix, Portrait: h > w, Line: body.H, Style: style}
	l.Root = image.Rect(ix, iy, w-ix, h-iy)
	l.Status = image.Rect(l.Root.Min.X, l.Root.Min.Y, l.Root.Max.X, l.Root.Min.Y+statusH)
	l.Hint = image.Rect(l.Root.Min.X, l.Root.Max.Y-hintH, l.Root.Max.X, l.Root.Max.Y)
	l.Body = image.Rect(l.Root.Min.X, l.Status.Max.Y, l.Root.Max.X, l.Hint.Min.Y)
	b := l.Body
	switch {
	case l.Portrait && style == "split":
		// the bottom pane takes 40% of the body: a 4:3 picture at the left
		// and the text beside it (a vertical shot leaves it more room)
		ph := b.Dy() * 40 / 100
		th := ph - 6
		tw := th * 4 / 3
		if tw > b.Dx()-6 {
			tw = b.Dx() - 6
			th = tw * 3 / 4
		}
		l.Pane = image.Rect(b.Min.X, b.Max.Y-ph, b.Max.X, b.Max.Y)
		l.List = image.Rect(b.Min.X, b.Min.Y, b.Max.X, l.Pane.Min.Y-1)
		l.Thumb = image.Rect(l.Pane.Min.X, l.Pane.Min.Y+3, l.Pane.Min.X+tw, l.Pane.Min.Y+3+th)
		l.PaneText = image.Rect(l.Thumb.Max.X+4, l.Pane.Min.Y+2, l.Pane.Max.X, l.Pane.Max.Y)
		l.TextBeside = true
	case l.Portrait && style == "picture":
		// a picture across the bottom with three lines of text under it;
		// a vertical shot keeps its shape and the text moves beside it
		tw := b.Dx() - 6
		th := tw * 3 / 4
		ph := 3 + th + 3 + paneTextH
		l.Pane = image.Rect(b.Min.X, b.Max.Y-ph, b.Max.X, b.Max.Y)
		l.List = image.Rect(b.Min.X, b.Min.Y, b.Max.X, l.Pane.Min.Y-1)
		l.Thumb = image.Rect(l.Pane.Min.X+3, l.Pane.Min.Y+3, l.Pane.Min.X+3+tw, l.Pane.Min.Y+3+th)
		l.PaneText = image.Rect(l.Pane.Min.X+3, l.Thumb.Max.Y+3, l.Pane.Max.X, l.Pane.Max.Y)
		l.TextBeside = true
	case !l.Portrait && style == "split":
		// the pane takes 45% of the width, its picture 4:3 in that width
		pw := b.Dx() * 45 / 100
		tw := pw - 6
		th := tw * 3 / 4
		l.Pane = image.Rect(b.Max.X-pw, b.Min.Y, b.Max.X, b.Max.Y)
		l.List = image.Rect(b.Min.X, b.Min.Y, l.Pane.Min.X-1, b.Max.Y)
		tx := l.Pane.Min.X + 3
		l.Thumb = image.Rect(tx, l.Pane.Min.Y+3, tx+tw, l.Pane.Min.Y+3+th)
		l.PaneText = image.Rect(tx, l.Thumb.Max.Y+3, l.Pane.Max.X, l.Pane.Max.Y)
	case !l.Portrait && style == "picture":
		// the pane runs across the top, a 4:3 picture 55% of the body high
		// with the text beside it; the rows use the full width below
		th := b.Dy() * 55 / 100
		tw := th * 4 / 3
		l.Pane = image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Min.Y+th+6)
		l.PaneTop = true
		l.List = image.Rect(b.Min.X, l.Pane.Max.Y+1, b.Max.X, b.Max.Y)
		l.Thumb = image.Rect(l.Pane.Min.X+3, l.Pane.Min.Y+3, l.Pane.Min.X+3+tw, l.Pane.Min.Y+3+th)
		l.PaneText = image.Rect(l.Thumb.Max.X+4, l.Pane.Min.Y+3, l.Pane.Max.X, l.Pane.Max.Y)
		l.TextBeside = true
	case l.Portrait:
		l.Pane = image.Rect(l.Body.Min.X, l.Body.Max.Y-paneH, l.Body.Max.X, l.Body.Max.Y)
		l.List = image.Rect(l.Body.Min.X, l.Body.Min.Y, l.Body.Max.X, l.Pane.Min.Y-1)
		l.Thumb = image.Rect(l.Pane.Min.X, l.Pane.Min.Y+3, l.Pane.Min.X+thumbW, l.Pane.Min.Y+3+thumbH)
		l.PaneText = image.Rect(l.Thumb.Max.X+4, l.Pane.Min.Y+2, l.Pane.Max.X, l.Pane.Max.Y)
	default:
		l.Pane = image.Rect(l.Body.Max.X-paneW, l.Body.Min.Y, l.Body.Max.X, l.Body.Max.Y)
		l.List = image.Rect(l.Body.Min.X, l.Body.Min.Y, l.Pane.Min.X-1, l.Body.Max.Y)
		// picture and text share a left edge, 3 px in from the separator
		tx := l.Pane.Min.X + 3
		l.Thumb = image.Rect(tx, l.Pane.Min.Y+3, tx+thumbW, l.Pane.Min.Y+3+thumbH)
		l.PaneText = image.Rect(tx, l.Thumb.Max.Y+3, l.Pane.Max.X, l.Pane.Max.Y)
	}
	// Separate the main list from the header without moving modal panels.
	l.List.Min.Y += 2
	// a 2 px scrollbar beside the rows
	l.Scroll = image.Rect(l.List.Max.X-2, l.List.Min.Y, l.List.Max.X, l.List.Max.Y)
	l.List.Max.X -= 4
	l.Lines = l.List.Dy() / l.Line
	l.Cols = l.List.Dx() / body.W
	// a row: the favorite star (body font), the title, then a gap, the
	// status glyph, a gap and the date, all in rowW cells of the row font
	l.TitleW = l.List.Dx() - body.W - rowW*(3+dateCols)
	if l.TitleW < 8*body.W {
		l.TitleW = 8 * body.W
	}
	return l
}

// lineRect is the rectangle of list line n (0-based from the top of the list).
func (l *Layout) lineRect(n int) image.Rectangle {
	y := l.List.Min.Y + n*l.Line
	return image.Rect(l.List.Min.X, y, l.List.Max.X, y+l.Line)
}
