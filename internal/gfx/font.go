package gfx

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"
)

// Font is a monospace bitmap font parsed from BDF: one cell of W x H pixels
// per glyph, ASCII 32..126 only, one bitmask byte per row (cells are at most
// 8 px wide). Unknown characters draw as '?'.
type Font struct {
	Name    string
	W, H    int
	descent int
	glyphs  [128][]byte // rows, top to bottom, MSB = leftmost pixel
	has     [128]bool
	// proportional metrics: the first ink column and the advance (ink width
	// plus a one-pixel gap; half a cell for a blank glyph)
	lb  [128]uint8
	adv [128]uint8
}

// ParseBDF reads a BDF font with cells up to 8 px wide.
func ParseBDF(data []byte) (*Font, error) {
	f := &Font{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	var (
		enc                = -1
		bbw, bbh, bbx, bby int
		inBitmap           bool
		rows               []byte
		fbw, fbh, fbx, fby int
		haveFB             bool
		descent            int
	)
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch {
		case inBitmap && fields[0] == "ENDCHAR":
			inBitmap = false
			if enc >= 0 && enc < 128 {
				f.glyphs[enc] = placeGlyph(rows, bbw, bbh, bbx-fbx, bby, fbw, fbh, descent)
				f.has[enc] = true
			}
			enc = -1
		case inBitmap:
			v, err := strconv.ParseUint(fields[0], 16, 32)
			if err != nil {
				return nil, fmt.Errorf("bdf: bad bitmap row %q", line)
			}
			// rows are padded to whole bytes; keep the top byte (cells <= 8 wide)
			shift := uint(len(fields[0])*4 - 8)
			rows = append(rows, byte(v>>shift))
		case fields[0] == "FONT" && len(fields) > 1:
			f.Name = fields[1]
		case fields[0] == "FONTBOUNDINGBOX" && len(fields) >= 5:
			fbw, _ = strconv.Atoi(fields[1])
			fbh, _ = strconv.Atoi(fields[2])
			fbx, _ = strconv.Atoi(fields[3])
			fby, _ = strconv.Atoi(fields[4])
			descent = -fby
			haveFB = true
		case fields[0] == "ENCODING" && len(fields) > 1:
			enc, _ = strconv.Atoi(fields[1])
		case fields[0] == "BBX" && len(fields) >= 5:
			bbw, _ = strconv.Atoi(fields[1])
			bbh, _ = strconv.Atoi(fields[2])
			bbx, _ = strconv.Atoi(fields[3])
			bby, _ = strconv.Atoi(fields[4])
		case fields[0] == "BITMAP":
			inBitmap = true
			rows = rows[:0]
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if !haveFB || fbw <= 0 || fbw > 8 || fbh <= 0 {
		return nil, fmt.Errorf("bdf: need a FONTBOUNDINGBOX up to 8 px wide, got %dx%d", fbw, fbh)
	}
	f.W, f.H, f.descent = fbw, fbh, descent
	if !f.has['?'] {
		return nil, fmt.Errorf("bdf: font has no '?' glyph")
	}
	for ch := 0; ch < 128; ch++ {
		f.measure(byte(ch))
	}
	return f, nil
}

// measure records the proportional metrics of one glyph.
func (f *Font) measure(ch byte) {
	first, last := f.W, -1
	for _, r := range f.glyphs[ch] {
		if r == 0 {
			continue
		}
		for gx := 0; gx < f.W; gx++ {
			if r&(0x80>>uint(gx)) != 0 {
				first = min(first, gx)
				last = max(last, gx)
			}
		}
	}
	if last < 0 {
		f.lb[ch], f.adv[ch] = 0, uint8(max(1, (f.W+1)/2))
		return
	}
	f.lb[ch], f.adv[ch] = uint8(first), uint8(last-first+2)
}

// Advance is the proportional width of a character: its ink plus a
// one-pixel gap, half a cell when blank.
func (f *Font) Advance(ch byte) int {
	if ch >= 128 || !f.has[ch] {
		ch = '?'
	}
	return int(f.adv[ch])
}

// PropWidth is the pixel width of s drawn proportionally.
func (f *Font) PropWidth(s string) int {
	w := 0
	for i := 0; i < len(s); i++ {
		w += f.Advance(s[i])
	}
	return w
}

// FitProp trims s to at most maxW pixels drawn proportionally, ending in
// the ellipsis when it had to cut.
func FitProp(f *Font, s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if f.PropWidth(s) <= maxW {
		return s
	}
	room := maxW - f.Advance(Ellipsis[0])
	w := 0
	for i := 0; i < len(s); i++ {
		if w+f.Advance(s[i]) > room {
			return s[:i] + Ellipsis
		}
		w += f.Advance(s[i])
	}
	return s
}

// TextProp draws s proportionally spaced: each glyph's ink starts at the
// pen, followed by a one-pixel gap. Returns the width drawn.
func (c *Canvas) TextProp(x, y int, f *Font, s string, col color.RGBA) int {
	cx := x
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch >= 128 || !f.has[ch] {
			ch = '?'
		}
		c.glyphAt(cx-int(f.lb[ch]), y, f, f.Glyph(ch), col, c.Rect)
		cx += int(f.adv[ch])
	}
	c.Dirty(image.Rect(x, y, cx, y+f.H))
	return cx - x
}

// TextClip draws s with its top-left at (x, y), showing only what falls
// inside clip (a marquee's window).
func (c *Canvas) TextClip(x, y int, f *Font, s string, col color.RGBA, clip image.Rectangle) {
	clip = clip.Intersect(c.Rect)
	if clip.Empty() {
		return
	}
	cx := x
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch >= 128 {
			ch = '?'
		}
		if cx+f.W > clip.Min.X && cx < clip.Max.X {
			c.glyphAt(cx, y, f, f.Glyph(ch), col, clip)
		}
		cx += f.W
	}
	c.Dirty(image.Rect(x, y, cx, y+f.H).Intersect(clip))
}

// placeGlyph puts a glyph's bitmap rows into the font cell using its BBX
// offsets: the cell's baseline sits `descent` rows above the cell bottom.
func placeGlyph(rows []byte, bbw, bbh, bbx, bby, cellW, cellH, descent int) []byte {
	cell := make([]byte, cellH)
	top := cellH - descent - bby - bbh // row of the cell where the glyph starts
	for i := 0; i < bbh && i < len(rows); i++ {
		y := top + i
		if y < 0 || y >= cellH {
			continue
		}
		r := rows[i]
		if bbx > 0 {
			r >>= uint(bbx)
		} else if bbx < 0 {
			r <<= uint(-bbx)
		}
		cell[y] = r
	}
	_ = bbw
	_ = cellW
	return cell
}

// Glyph returns the cell rows for a character, '?' for anything unknown.
func (f *Font) Glyph(ch byte) []byte {
	if ch < 128 && f.has[ch] {
		return f.glyphs[ch]
	}
	return f.glyphs['?']
}

// Arrow glyphs live on four control bytes so hints can show a real arrow
// for a direction key ("\x13 \x14 page").
const (
	ArrowUp    = "\x11"
	ArrowDown  = "\x12"
	ArrowLeft  = "\x13"
	ArrowRight = "\x14"
	Ellipsis   = "\x15" // three dots in one cell
	Beta       = "\x16" // a beta sign for Patreon beta cores
)

// SetGlyph installs a glyph (rows top to bottom, MSB = leftmost pixel).
func (f *Font) SetGlyph(ch byte, rows []byte) {
	if ch < 128 {
		f.glyphs[ch] = rows
		f.has[ch] = true
		f.measure(ch)
	}
}

// AddArrows installs triangle arrows sized for this font's cell, centred
// vertically.
func (f *Font) AddArrows() {
	place := func(shape []byte) []byte {
		rows := make([]byte, f.H)
		top := (f.H - len(shape)) / 2
		copy(rows[top:], shape)
		return rows
	}
	up := []byte{0x20, 0x70, 0xF8}
	down := []byte{0xF8, 0x70, 0x20}
	left := []byte{0x20, 0x60, 0xE0, 0x60, 0x20}
	right := []byte{0x80, 0xC0, 0xE0, 0xC0, 0x80}
	if f.W >= 6 { // one pixel wider in the body font
		up = []byte{0x20, 0x70, 0xF8, 0xFC}
		down = []byte{0xFC, 0xF8, 0x70, 0x20}
		left = []byte{0x10, 0x30, 0x70, 0xF0, 0x70, 0x30, 0x10}
		right = []byte{0x80, 0xC0, 0xE0, 0xF0, 0xE0, 0xC0, 0x80}
	}
	f.SetGlyph(ArrowUp[0], place(up))
	f.SetGlyph(ArrowDown[0], place(down))
	f.SetGlyph(ArrowLeft[0], place(left))
	f.SetGlyph(ArrowRight[0], place(right))
	// the ellipsis: three dots on the baseline in a single cell
	dots := make([]byte, f.H)
	base := max(0, min(f.H-1, f.H-f.descent-1))
	dots[base] = 0xA8
	f.SetGlyph(Ellipsis[0], dots)
	// a plus with one-pixel arms, so "a+b" does not touch its neighbours
	plus := make([]byte, f.H)
	top := (f.H-3)/2 + 1
	plus[top], plus[top+1], plus[top+2] = 0x20, 0x70, 0x20
	f.SetGlyph('+', plus)
	// a beta sign: two loops on a stem that reaches below the baseline
	beta := make([]byte, f.H)
	if f.H >= 12 {
		copy(beta[1:], []byte{0x70, 0x88, 0x88, 0xB0, 0x88, 0x88, 0x88, 0xB0, 0x80, 0x80})
	} else {
		copy(beta[max(0, f.H-8):], []byte{0x60, 0x90, 0xA0, 0x90, 0x90, 0xA0, 0x80, 0x80})
	}
	f.SetGlyph(Beta[0], beta)
}

// Cols is how many cells fit in w pixels.
func (f *Font) Cols(w int) int { return w / f.W }

// Width is the pixel width of s.
func (f *Font) Width(s string) int { return len(s) * f.W }

// Text draws s with its top-left at (x, y) and returns the width drawn.
// Non-ASCII bytes draw as '?'. No clipping beyond the canvas edge.
func (c *Canvas) Text(x, y int, f *Font, s string, col color.RGBA) int {
	cx := x
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch >= 128 {
			ch = '?'
		}
		c.glyphAt(cx, y, f, f.Glyph(ch), col, c.Rect)
		cx += f.W
	}
	c.Dirty(image.Rect(x, y, cx, y+f.H))
	return cx - x
}

func (c *Canvas) glyphAt(x, y int, f *Font, rows []byte, col color.RGBA, clip image.Rectangle) {
	for gy, r := range rows {
		py := y + gy
		if py < clip.Min.Y || py >= clip.Max.Y || r == 0 {
			continue
		}
		for gx := 0; gx < f.W; gx++ {
			if r&(0x80>>uint(gx)) == 0 {
				continue
			}
			px := x + gx
			if px < clip.Min.X || px >= clip.Max.X {
				continue
			}
			o := c.PixOffset(px, py)
			c.Pix[o], c.Pix[o+1], c.Pix[o+2], c.Pix[o+3] = col.R, col.G, col.B, 255
		}
	}
}

// Fit trims s to at most cols cells, ending in the one-cell ellipsis when
// it had to cut.
func Fit(s string, cols int) string {
	if cols <= 0 {
		return ""
	}
	if len(s) <= cols {
		return s
	}
	if cols <= 1 {
		return s[:cols]
	}
	return s[:cols-1] + Ellipsis
}

// TextFit draws s trimmed to maxW pixels.
func (c *Canvas) TextFit(x, y int, f *Font, s string, col color.RGBA, maxW int) int {
	return c.Text(x, y, f, Fit(s, f.Cols(maxW)), col)
}

// TextRight draws s ending at x (right-aligned).
func (c *Canvas) TextRight(x, y int, f *Font, s string, col color.RGBA) int {
	return c.Text(x-f.Width(s), y, f, s, col)
}

// Wrap breaks s into lines of at most cols cells on spaces, hard-breaking
// words longer than a line. At most maxLines lines; when text remains, the
// last line is the rest of the text trimmed with the ellipsis.
func Wrap(s string, cols, maxLines int) []string {
	if cols <= 0 || maxLines <= 0 {
		return nil
	}
	var lines []string
	line := ""
	for _, w := range strings.Fields(s) {
		for len(w) > cols {
			if line != "" {
				lines = append(lines, line)
				line = ""
			}
			lines = append(lines, w[:cols])
			w = w[cols:]
		}
		switch {
		case line == "":
			line = w
		case len(line)+1+len(w) <= cols:
			line += " " + w
		default:
			lines = append(lines, line)
			line = w
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	if len(lines) > maxLines {
		rest := strings.Join(lines[maxLines-1:], " ")
		lines = append(lines[:maxLines-1], Fit(rest, cols))
	}
	return lines
}
