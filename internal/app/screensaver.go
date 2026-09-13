package app

import (
	"image"
	"math"
	"sort"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

const saverFrame = time.Second / 30

// saverFade is how long the screen takes to dim from full brightness to the
// saver's quarter; the lettering enters once it is dark.
const saverFade = time.Second

// saverShade is the dimmed screen's brightness in 256ths: a quarter.
const saverShade = 64

// saverBlurDiv sets the blur under the lettering from the screen width:
// the box radius at full strength is width/saverBlurDiv, applied twice
// each way (close to a Gaussian), so an 11 px radius on a 320 px screen.
const saverBlurDiv = 28

var saverValues = []string{"off", "1", "2", "5", "10"}

type saverKey struct {
	key  platform.Key
	code uint16
}

type screensaver struct {
	active    bool
	lastInput time.Time
	started   time.Time
	next      time.Time
	travel    int
	shade     int // brightness of the screen under the lettering, in 256ths
	fade      int // how far the fade has come, in 256ths; 256 once dark
	// the picture under the lettering, frozen at the first saver frame:
	// as painted, blurred, and blurred and dimmed once the fade is done
	sharp, blurred, dark []uint8
	cacheW               int     // width the cache was taken at (a rotation swaps it)
	blur                 []uint8 // scratch for the blur passes, one canvas
	mask                 *image.Alpha
	waking               map[saverKey]bool
}

func (a *App) Screensaver() string {
	switch a.cfg.Screensaver {
	case "off", "2", "5", "10":
		return a.cfg.Screensaver
	default:
		return "1"
	}
}

func (a *App) ScreensaverActive() bool { return a.saver.active }

func (a *App) saverDelay() time.Duration {
	switch a.Screensaver() {
	case "off":
		return 0
	case "2":
		return 2 * time.Minute
	case "5":
		return 5 * time.Minute
	case "10":
		return 10 * time.Minute
	default:
		return time.Minute
	}
}

// Swallow the wake press, its duplicates and its release before normal actions
// see them. In particular, waking with Start must never launch a game, and
// holding B must not cancel an update until B has been released and pressed again.
func (a *App) handleSaverInput(ev platform.Event) bool {
	if ev.Key == platform.KeyNone && ev.Text == 0 {
		return false
	}
	at := ev.At
	if at.IsZero() {
		at = a.cfg.TimerNow()
	}
	if at.After(a.saver.lastInput) {
		a.saver.lastInput = at
	}
	id := saverKey{key: ev.Key}
	if ev.Key == platform.KeyOther {
		id.code = ev.Code
	}
	if a.saver.waking[id] {
		if !ev.Pressed {
			delete(a.saver.waking, id)
		}
		return true
	}
	if !a.saver.active {
		return false
	}
	if !ev.Pressed {
		delete(a.down, ev.Key)
		return true // releasing the preview button doesn't dismiss the preview
	}
	a.saver.active = false
	a.saver.sharp, a.saver.blurred, a.saver.dark = nil, nil, nil
	a.rep = repeater{}
	a.down = map[platform.Key]bool{}
	a.updateView.backAt = time.Time{}
	if ev.Key != platform.KeyNone { // harness text events have no release
		a.saver.waking = map[saverKey]bool{id: true}
	}
	a.all = true
	return true
}

func (a *App) startSaver(now time.Time) {
	a.saver.active = true
	a.saver.started = now
	a.saver.next = now.Add(saverFrame)
	a.saver.travel = 0
	a.saver.shade = 256
	a.saver.fade = 0
	a.saver.sharp, a.saver.blurred, a.saver.dark = nil, nil, nil
	a.rep = repeater{}
	a.all = true
}

func (a *App) nextSaverTick() time.Time {
	if a.screen == ScreenTroubleshooting {
		return time.Time{}
	}
	if a.saver.active {
		return a.saver.next
	}
	if delay := a.saverDelay(); delay > 0 && len(a.down) == 0 && len(a.saver.waking) == 0 {
		return a.saver.lastInput.Add(delay)
	}
	return time.Time{}
}

func (a *App) tickSaver(now time.Time) bool {
	next := a.nextSaverTick()
	if next.IsZero() || now.Before(next) {
		return false
	}
	if !a.saver.active {
		a.startSaver(now)
	} else {
		a.saverAdvance(now)
		a.saver.next = now.Add(saverFrame)
		a.all = true
	}
	return true
}

// saverAdvance sets the fade, the shade and the lettering's travel for
// now: the screen dims and blurs over saverFade, then the word slides one
// pixel a frame.
func (a *App) saverAdvance(now time.Time) {
	since := now.Sub(a.saver.started)
	if since < saverFade {
		a.saver.fade = int(256 * since / saverFade)
		a.saver.shade = 256 - (256-saverShade)*a.saver.fade/256
		a.saver.travel = 0
		return
	}
	a.saver.fade = 256
	a.saver.shade = saverShade
	a.saver.travel = int((since - saverFade) / saverFrame)
}

// saverCached reports whether the picture under the lettering has been
// taken for this canvas.
func (a *App) saverCached(c *gfx.Canvas) bool {
	return len(a.saver.sharp) == len(c.Pix) && a.saver.cacheW == c.W()
}

// saverCapture freezes the picture under the lettering as the canvas
// holds it now, and blurs a copy: a box blur of radius width/saverBlurDiv
// run twice along the rows and twice down the columns, edges repeated.
// Running sums make the cost independent of the radius. One pass over the
// frame costs the boards tens of milliseconds, so it happens once per
// saver run, not per frame; the saver shows this frozen picture until a
// key wakes the app.
func (a *App) saverCapture(c *gfx.Canvas) {
	s := &a.saver
	s.sharp = append(s.sharp[:0], c.Pix...)
	s.blurred = append(s.blurred[:0], c.Pix...)
	s.dark = nil
	s.cacheW = c.W()
	if len(s.blur) != len(c.Pix) {
		s.blur = make([]uint8, len(c.Pix))
	}
	w, h, r := c.W(), c.H(), c.W()/saverBlurDiv
	if r < 1 {
		return
	}
	for pass := 0; pass < 2; pass++ {
		for y := 0; y < h; y++ {
			blurLine(s.blurred[y*c.Stride:], s.blur[y*c.Stride:], w, 4, r)
		}
		for x := 0; x < w; x++ {
			blurLine(s.blur[x*4:], s.blurred[x*4:], h, c.Stride, r)
		}
	}
}

// blurLine box-blurs the three colour channels of one line of n pixels,
// step bytes apart, from src into dst; the alpha byte is copied.
func blurLine(src, dst []uint8, n, step, r int) {
	if r > n-1 {
		r = n - 1
	}
	span := 2*r + 1
	for ch := 0; ch < 3; ch++ {
		// the window for pixel 0 covers -r..r with the left edge repeated
		sum := int(src[ch]) * (r + 1)
		for i := 1; i <= r; i++ {
			sum += int(src[i*step+ch])
		}
		for i := 0; i < n; i++ {
			dst[i*step+ch] = uint8(sum / span)
			// slide: drop i-r, take i+r+1, both clamped to the line
			out, in := i-r, i+r+1
			if out < 0 {
				out = 0
			}
			if in > n-1 {
				in = n - 1
			}
			sum += int(src[in*step+ch]) - int(src[out*step+ch])
		}
	}
	for i := 0; i < n; i++ {
		dst[i*step+3] = src[i*step+3]
	}
}

func (a *App) saverMask(h int) *image.Alpha {
	if a.saver.mask != nil && a.saver.mask.Rect.Dy() == h {
		return a.saver.mask
	}
	const word = "MISTERZINE"
	w := 0
	for _, ch := range []byte(word) {
		w += saverLetters[ch].width
	}
	m := image.NewAlpha(image.Rect(0, 0, w*h/saverUnits, h))
	x := 0
	for _, ch := range []byte(word) {
		letter := saverLetters[ch]
		fillSaverPolygon(m, x, letter.outline, 255)
		for _, counter := range letter.counters {
			fillSaverPolygon(m, x, counter, 0)
		}
		x += letter.width
	}
	saverRim(m)
	a.saver.mask = m
	return m
}

// Rim lighting: the letters' one-pixel outline carries a facing code instead
// of solid ink, 1 + (nx+1) + 3*(ny+1) for the quantized outward normal
// (nx, ny), so the paint loop can light the edge without a second mask. Two
// still columns of light, tilted a little off vertical, stand on the screen:
// a violet one that only the left-facing edges reflect and a lime one,
// leaning the other way, that only the right-facing edges reflect (horizontal edges split by whether
// they face up or down). The letters scroll through them, so each glint
// slides along an edge purely as a result of the lettering's own motion.
// The glint never exceeds saverGlintMax and edges outside a column stay
// black.
const (
	saverInk       = 255
	saverGlintMax  = 102 // brightest any channel of the edge gets: 40% of white
	saverBandFloor = 0.3 // share of the glint a glancing facing shows
)

// saverLight is one still column of light: which facing side reflects it,
// where it stands as a share of the width, its lean in x per row (the glint
// travels vertically at 1/lean rows per frame, so a small lean is fast) and
// its hue at full strength.
type saverLight struct {
	side uint8
	at   float64
	tilt float64
	half int // half-width in pixels
	hue  [3]uint8
}

// saverLights are the theme's violet and lime, one per side leaning opposite
// ways, plus a dim fast glint on each side running the other way for depth.
var saverLights = []saverLight{
	{0, 0.3, 0.2, 40, [3]uint8{140, 40, 255}},
	{1, 0.7, -0.12, 40, [3]uint8{100, 215, 0}},  // more upright, so faster, and fatter for coverage
	{0, 0.55, -0.06, 40, [3]uint8{60, 110, 45}}, // faded green, fast, up
	{0, 0.15, -0.06, 40, [3]uint8{60, 110, 45}}, // the same again, elsewhere, so each letter meets it twice
	{1, 0.45, 0.06, 40, [3]uint8{90, 90, 90}},   // gray, fast, down
}

// saverBands are each light's smooth glint profile by distance from its
// column centre.
var saverBands = func() [][]uint8 {
	bands := make([][]uint8, len(saverLights))
	for i, l := range saverLights {
		t := make([]uint8, l.half)
		for d := range t {
			k := 1 - float64(d)/float64(l.half)
			t[d] = uint8(saverGlintMax * k * k)
		}
		bands[i] = t
	}
	return bands
}()

// saverSide says which light each facing code reflects, and saverFacing how
// strongly: squarely sideways facings fully, glancing ones less.
var saverSide, saverFacing = func() (side [10]uint8, f [10]uint8) {
	for ny := -1; ny <= 1; ny++ {
		for nx := -1; nx <= 1; nx++ {
			if nx == 0 && ny == 0 {
				continue
			}
			code := 1 + (nx + 1) + 3*(ny+1)
			if nx > 0 || (nx == 0 && ny > 0) {
				side[code] = 1
			}
			d := math.Abs(float64(nx)) / math.Hypot(float64(nx), float64(ny))
			f[code] = uint8(255 * (saverBandFloor + (1-saverBandFloor)*d))
		}
	}
	return side, f
}()

func saverRim(m *image.Alpha) {
	w, h := m.Rect.Dx(), m.Rect.Dy()
	ink := func(x, y int) bool {
		if y < 0 || y >= h {
			return true // the letters run past the top and bottom of the screen
		}
		return x >= 0 && x < w && m.Pix[y*m.Stride+x] != 0
	}
	sign := func(v int) int {
		if v < 0 {
			return -1
		}
		if v > 0 {
			return 1
		}
		return 0
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !ink(x, y) || (ink(x-1, y) && ink(x+1, y) && ink(x, y-1) && ink(x, y+1)) {
				continue
			}
			sx, sy := 0, 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if (dx != 0 || dy != 0) && !ink(x+dx, y+dy) {
						sx += dx
						sy += dy
					}
				}
			}
			m.Pix[y*m.Stride+x] = uint8(1 + (sign(sx) + 1) + 3*(sign(sy)+1))
		}
	}
}

// saverLetter is one capital: its advance width and outline polygon in
// saverUnits, plus the polygons of any counters. Generated by
// tools/saver_letters.py into saver_letters.go.
type saverLetter struct {
	width    int
	outline  []image.Point
	counters [][]image.Point
}

func fillSaverPolygon(m *image.Alpha, offset int, points []image.Point, ink uint8) {
	scale := float64(m.Rect.Dy()) / saverUnits
	xs := make([]float64, 0, len(points))
	for y := 0; y < m.Rect.Dy(); y++ {
		py := (float64(y) + .5) / scale
		xs = xs[:0]
		for i, p := range points {
			q := points[(i+1)%len(points)]
			if (float64(p.Y) <= py && float64(q.Y) > py) || (float64(q.Y) <= py && float64(p.Y) > py) {
				x := float64(p.X) + (py-float64(p.Y))*float64(q.X-p.X)/float64(q.Y-p.Y)
				xs = append(xs, (float64(offset)+x)*scale)
			}
		}
		sort.Float64s(xs)
		for i := 0; i+1 < len(xs); i += 2 {
			start := max(0, int(math.Ceil(xs[i]-.5)))
			end := min(m.Rect.Dx(), int(math.Ceil(xs[i+1]-.5)))
			for x := start; x < end; x++ {
				m.Pix[y*m.Stride+x] = ink
			}
		}
	}
}

func (a *App) paintSaver(c *gfx.Canvas) {
	m := a.saverMask(c.H())
	// One logical pixel per frame; the whole word enters at the right and
	// leaves at the left. Safe-zone insets don't clip this overlay.
	x0 := c.W() - a.saver.travel%(c.W()+m.Rect.Dx())
	shade, fade := a.saver.shade, a.saver.fade
	if shade <= 0 || shade > 256 {
		shade, fade = saverShade, 256 // a saver frame set up outside tickSaver (tests, previews)
	}
	s := &a.saver
	if !a.saverCached(c) {
		a.saverCapture(c)
	}
	if fade >= 256 {
		// dark: the blurred picture at the saver's shade, made once
		if s.dark == nil {
			s.dark = append(s.dark[:0], s.blurred...)
			for i := 0; i+3 < len(s.dark); i += 4 {
				s.dark[i] = uint8(int(s.dark[i]) * shade >> 8)
				s.dark[i+1] = uint8(int(s.dark[i+1]) * shade >> 8)
				s.dark[i+2] = uint8(int(s.dark[i+2]) * shade >> 8)
			}
		}
		copy(c.Pix, s.dark)
	} else {
		// mid-fade: the sharp picture crossing into the blurred one, dimmed
		for i := 0; i+3 < len(c.Pix); i += 4 {
			for ch := i; ch < i+3; ch++ {
				v := (int(s.sharp[ch])*(256-fade) + int(s.blurred[ch])*fade) >> 8
				c.Pix[ch] = uint8(v * shade >> 8)
			}
		}
	}
	// the lettering, over the columns it covers this frame
	centres := make([]int, len(saverLights))
	for i, l := range saverLights {
		centres[i] = int(l.at*float64(c.W())) + int(l.tilt*float64(c.H())/2)
	}
	from, to := max(x0, 0), min(x0+m.Rect.Dx(), c.W())
	for y := 0; y < c.H(); y++ {
		for x := from; x < to; x++ {
			v := m.Pix[y*m.Stride+x-x0]
			if v == 0 {
				continue
			}
			i := c.PixOffset(x, y)
			if v == saverInk {
				c.Pix[i], c.Pix[i+1], c.Pix[i+2] = 0, 0, 0
				continue
			}
			var r, g, b int
			for li, l := range saverLights {
				if l.side != saverSide[v] {
					continue
				}
				if d := x + int(l.tilt*float64(y)) - centres[li]; d < l.half && d > -l.half {
					k := int(saverBands[li][max(d, -d)]) * int(saverFacing[v]) / 255
					r += int(l.hue[0]) * k / 255
					g += int(l.hue[1]) * k / 255
					b += int(l.hue[2]) * k / 255
				}
			}
			c.Pix[i], c.Pix[i+1], c.Pix[i+2] = uint8(min(r, saverGlintMax)), uint8(min(g, saverGlintMax)), uint8(min(b, saverGlintMax))
		}
	}
	c.DirtyAll()
}
