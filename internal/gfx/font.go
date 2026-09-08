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
	Name   string
	W, H   int
	glyphs [128][]byte // rows, top to bottom, MSB = leftmost pixel
	has    [128]bool
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
	f.W, f.H = fbw, fbh
	if !f.has['?'] {
		return nil, fmt.Errorf("bdf: font has no '?' glyph")
	}
	return f, nil
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
		c.glyph(cx, y, f, f.Glyph(ch), col)
		cx += f.W
	}
	c.Dirty(image.Rect(x, y, cx, y+f.H))
	return cx - x
}

func (c *Canvas) glyph(x, y int, f *Font, rows []byte, col color.RGBA) {
	for gy, r := range rows {
		py := y + gy
		if py < c.Rect.Min.Y || py >= c.Rect.Max.Y || r == 0 {
			continue
		}
		for gx := 0; gx < f.W; gx++ {
			if r&(0x80>>uint(gx)) == 0 {
				continue
			}
			px := x + gx
			if px < c.Rect.Min.X || px >= c.Rect.Max.X {
				continue
			}
			o := c.PixOffset(px, py)
			c.Pix[o], c.Pix[o+1], c.Pix[o+2], c.Pix[o+3] = col.R, col.G, col.B, 255
		}
	}
}

// Fit trims s to at most cols cells, ending in ".." when it had to cut.
func Fit(s string, cols int) string {
	if cols <= 0 {
		return ""
	}
	if len(s) <= cols {
		return s
	}
	if cols <= 2 {
		return s[:cols]
	}
	return s[:cols-2] + ".."
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
// last line is the rest of the text trimmed with "..".
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
