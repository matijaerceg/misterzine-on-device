package app

import (
	"image"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

const saverFrame = time.Second / 30

// saverFade is how long the screen takes to dim and blur into the saver's
// ground; the lettering enters once it is dark.
const saverFade = time.Second

// saverWake is how long a wake takes to bring the picture back sharp and
// bright: the fade run backwards, four times as fast.
const saverWake = saverFade / 4

// saverLevels is how many blur steps the fade climbs through, each with a
// larger radius and more bloom. A worker computes them from the frozen
// picture while the fade runs; the fade waits for a level that is not
// ready yet, so a slow board fades a little longer, never in jumps.
const saverLevels = 4

// saverGround is the picture under the lettering at each level: pix[0] as
// painted, pix[k] bloomed and blurred at k/saverLevels of the look, three
// channels a pixel with eight bits of fraction (8.8) so the dither has
// something to spread. The worker fills the levels in order and publishes
// each through ready.
type saverGround struct {
	pix   [saverLevels + 1][]uint16
	ready atomic.Int32
	done  sync.WaitGroup
}

// SaverLook tunes the picture under the lettering. It blooms: every
// channel above Knee is raised by Gain times the excess (256ths), without
// clamping, before the blur spreads it, so light text and pictures glow
// out into their surroundings while the dark ground stays dark. The blur
// is a box blur of radius width/BlurDiv run Passes times along the rows
// and down the columns (two passes are close to a Gaussian). The blurred
// picture is clamped and shown at Shade (256ths). The levels keep eight
// bits of fraction and Dither (1) spreads the last bit as an ordered
// pattern at the final write, since the dark ground's gradients band on
// a CRT otherwise. The debug API sets a look live (SetSaverLook) so it
// can be tuned on a CRT.
type SaverLook struct {
	Knee, Gain, Shade, BlurDiv, Passes, Dither int
}

// DefaultSaverLook was chosen on a CRT with the debug tuner (2026-09-13):
// only the brightest parts bloom, six times their excess, the ground shows
// at 57%, and the blur radius is a 15th of the width (21 px on 320).
var DefaultSaverLook = SaverLook{Knee: 171, Gain: 1538, Shade: 145, BlurDiv: 15, Passes: 2, Dither: 1}

// clamped keeps a look inside what the capture can do.
func (l SaverLook) clamped() SaverLook {
	l.Knee = min(max(l.Knee, 0), 255)
	l.Gain = min(max(l.Gain, 0), 8192)
	l.Shade = min(max(l.Shade, 16), 256)
	l.BlurDiv = min(max(l.BlurDiv, 4), 256)
	l.Passes = min(max(l.Passes, 1), 4)
	l.Dither = min(max(l.Dither, 0), 1)
	return l
}

// saverBayer is the 8x8 ordered dither: a threshold per pixel in 256ths
// of one 8-bit step. It is the same every frame on purpose; a pattern
// that changes crawls on a phosphor.
var saverBayer = func() (t [8][8]uint16) {
	m := [8][8]uint8{
		{0, 32, 8, 40, 2, 34, 10, 42}, {48, 16, 56, 24, 50, 18, 58, 26},
		{12, 44, 4, 36, 14, 46, 6, 38}, {60, 28, 52, 20, 62, 30, 54, 22},
		{3, 35, 11, 43, 1, 33, 9, 41}, {51, 19, 59, 27, 49, 17, 57, 25},
		{15, 47, 7, 39, 13, 45, 5, 37}, {63, 31, 55, 23, 61, 29, 53, 21},
	}
	for y := range m {
		for x := range m[y] {
			t[y][x] = uint16(m[y][x])*4 + 2
		}
	}
	return
}()

// saverQuantize turns a ground channel with eight bits of fraction into
// the byte the canvas takes: rounded, or dithered by the pixel's threshold.
func saverQuantize(v, x, y int, dither bool) uint8 {
	t := 128
	if dither {
		t = int(saverBayer[y&7][x&7])
	}
	return uint8(min((v+t)>>8, 255))
}

// saverPaintGround writes the ground into an RGBA frame: the level from,
// or its mix with to by frac (256ths), at the shade (256ths).
func saverPaintGround(dst []uint8, w, h, stride int, from, to []uint16, frac, shade int, dither bool) {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i, o := y*stride+x*4, (y*w+x)*3
			for ch := 0; ch < 3; ch++ {
				v := int(from[o+ch])
				if to != nil {
					v = (v*(256-frac) + int(to[o+ch])*frac) >> 8
				}
				dst[i+ch] = saverQuantize(v*shade>>8, x, y, dither)
			}
		}
	}
}

// boost is the bloom curve for one channel value.
func (l SaverLook) boost(v int) int {
	if v > l.Knee {
		v += (v - l.Knee) * l.Gain / 256
	}
	return v
}

// SaverLook is the saver ground's tuning.
func (a *App) SaverLook() SaverLook { return a.look }

// SetSaverLook applies a look; with the saver up the ground is taken
// again with it and the fade runs again from the sharp picture.
func (a *App) SetSaverLook(l SaverLook) {
	a.look = l.clamped()
	if a.ScreensaverActive() {
		s := &a.saver
		s.sharp, s.ground, s.dark = nil, nil, nil
		s.fadeT, s.darkAt = 0, time.Time{}
		a.all = true
	}
}

// SaverDemo starts the saver now, or wakes it, for tuning over the debug
// API; the wake acts like a key's wake without a key to swallow.
func (a *App) SaverDemo(on bool) {
	switch {
	case on && !a.ScreensaverActive():
		a.startSaver(a.cfg.TimerNow())
	case !on && a.ScreensaverActive():
		a.saverLeave(a.cfg.TimerNow())
	}
}

// SaverSettle waits for the ground's levels: the harness and the tests
// drive a scripted clock, which must not outrun the worker.
func (a *App) SaverSettle() {
	if g := a.saver.ground; g != nil {
		g.done.Wait()
	}
}

var saverValues = []string{"off", "1", "2", "5", "10"}

type saverKey struct {
	key  platform.Key
	code uint16
}

type screensaver struct {
	active    bool // painting the saver: the fade in, the lettering, or the fade back out
	leaving   bool // a wake is running the fade backwards
	lastInput time.Time
	started   time.Time
	next      time.Time
	ticked    time.Time     // when the fade last advanced
	fadeT     time.Duration // how far into saverFade the fade is
	darkAt    time.Time     // when the fade completed: the lettering's clock
	travel    int
	shade     int // brightness of the screen under the lettering, in 256ths
	fade      int // how far the fade has come, in 256ths; 256 once dark
	// the picture under the lettering, frozen at the first saver frame:
	// as painted, its blur levels, and the last level at the shade once
	// the fade is done
	sharp  []uint8
	ground *saverGround
	dark   []uint8
	cacheW int // width the picture was taken at (a rotation swaps it)
	mask   *image.Alpha
	waking map[saverKey]bool
}

func (a *App) Screensaver() string {
	switch a.cfg.Screensaver {
	case "off", "2", "5", "10":
		return a.cfg.Screensaver
	default:
		return "1"
	}
}

// ScreensaverActive reports the saver up: fading in or showing the
// lettering. The fade back out after a wake counts as awake, so keys act.
func (a *App) ScreensaverActive() bool { return a.saver.active && !a.saver.leaving }

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
	if !a.ScreensaverActive() {
		return false
	}
	if !ev.Pressed {
		delete(a.down, ev.Key)
		return true // releasing the preview button doesn't dismiss the preview
	}
	a.saverLeave(at)
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
	s := &a.saver
	s.active, s.leaving = true, false
	s.started, s.ticked, s.next = now, now, now.Add(saverFrame)
	s.fadeT, s.darkAt = 0, time.Time{}
	s.travel, s.shade, s.fade = 0, 256, 0
	s.sharp, s.ground, s.dark = nil, nil, nil
	a.rep = repeater{}
	a.all = true
}

// saverLeave starts a wake: the fade runs backwards over saverWake without
// the lettering, and the screen under it is painted again once the picture
// is back. With nothing frozen yet there is nothing to fade back from.
func (a *App) saverLeave(now time.Time) {
	s := &a.saver
	if s.ground == nil {
		a.saverEnd()
		return
	}
	s.leaving = true
	s.ticked, s.next = now, now
	a.all = true
}

// saverEnd takes the saver down and lets go of the frozen picture.
func (a *App) saverEnd() {
	s := &a.saver
	s.active, s.leaving = false, false
	s.sharp, s.ground, s.dark = nil, nil, nil
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
		if a.saver.leaving && a.saver.fadeT <= 0 {
			a.saverEnd() // the picture is back
		} else {
			// the next frame keeps the grid the last one was on: a tick
			// that ran late does not push every later one back, which
			// used to add up until the lettering skipped a column. A
			// whole frame lost starts a fresh grid from now.
			a.saver.next = a.saver.next.Add(saverFrame)
			if a.saver.next.Before(now) {
				a.saver.next = now.Add(saverFrame)
			}
			a.all = true
		}
	}
	return true
}

// saverAdvance moves the fade for now and sets the shade and the
// lettering's travel from it. The fade climbs over saverFade, held back by
// a level the worker has not finished, then the word slides one pixel a
// frame; a wake runs it back down over saverWake.
func (a *App) saverAdvance(now time.Time) {
	s := &a.saver
	dt := now.Sub(s.ticked)
	if dt < 0 {
		dt = 0
	}
	s.ticked = now
	switch {
	case s.leaving:
		s.fadeT -= dt * (saverFade / saverWake)
		if s.fadeT < 0 {
			s.fadeT = 0
		}
		s.darkAt = time.Time{}
	case s.fadeT < saverFade:
		s.fadeT += dt
		if g := s.ground; g != nil {
			if ready := saverFade * time.Duration(g.ready.Load()) / saverLevels; s.fadeT > ready {
				s.fadeT = ready
			}
		}
		if s.fadeT >= saverFade {
			// the moment the fade completed, not this tick, keeps the
			// lettering on the same clock as before the levels
			s.darkAt = now.Add(saverFade - s.fadeT)
			s.fadeT = saverFade
		}
	}
	s.fade = int(256 * s.fadeT / saverFade)
	s.shade = 256 - (256-a.look.Shade)*s.fade/256
	if s.darkAt.IsZero() {
		s.travel = 0
		return
	}
	s.travel = int(now.Sub(s.darkAt) / saverFrame)
}

// saverCached reports whether the picture under the lettering has been
// taken for this canvas.
func (a *App) saverCached(c *gfx.Canvas) bool {
	return len(a.saver.sharp) == len(c.Pix) && a.saver.cacheW == c.W()
}

// saverCapture freezes the picture under the lettering as the canvas
// holds it now and starts the worker that blurs it, level by level, in
// the background: one level costs the boards a couple of hundred
// milliseconds, far more than a frame, and the UI thread only ever mixes
// finished levels. The saver shows this frozen picture until a key wakes
// the app.
func (a *App) saverCapture(c *gfx.Canvas) {
	s := &a.saver
	s.sharp = append([]uint8(nil), c.Pix...) // a fresh copy: a worker from an earlier run may still read the old one
	s.dark = nil
	s.cacheW = c.W()
	w, h := c.W(), c.H()
	g := &saverGround{}
	g.pix[0] = make([]uint16, w*h*3)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i, o := c.PixOffset(x, y), (y*w+x)*3
			g.pix[0][o], g.pix[0][o+1], g.pix[0][o+2] = uint16(c.Pix[i])<<8, uint16(c.Pix[i+1])<<8, uint16(c.Pix[i+2])<<8
		}
	}
	s.ground = g
	g.done.Add(1)
	go saverBlurLevels(g, s.sharp, w, h, c.Stride, a.look)
}

// saverBlurLevels is the worker: level k is the sharp picture bloomed at
// k/saverLevels of the look's gain (SaverLook.boost, into 16-bit channels
// so nothing clamps before the blur) and box-blurred at k/saverLevels of
// its radius, the passes run along the rows and down the columns with the
// edges repeated; running sums make the cost independent of the radius.
// Each level is published as it finishes.
func saverBlurLevels(g *saverGround, sharp []uint8, w, h, stride int, look SaverLook) {
	defer g.done.Done()
	n := w * h * 3
	wide, tmp := make([]uint32, n), make([]uint32, n)
	for k := 1; k <= saverLevels; k++ {
		lk := look
		lk.Gain = look.Gain * k / saverLevels
		r := w / look.BlurDiv * k / saverLevels
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				i, o := y*stride+x*4, (y*w+x)*3
				wide[o] = uint32(lk.boost(int(sharp[i]))) << 8
				wide[o+1] = uint32(lk.boost(int(sharp[i+1]))) << 8
				wide[o+2] = uint32(lk.boost(int(sharp[i+2]))) << 8
			}
		}
		if r >= 1 {
			for pass := 0; pass < look.Passes; pass++ {
				for y := 0; y < h; y++ {
					blurLine(wide[y*w*3:], tmp[y*w*3:], w, 3, r)
				}
				for x := 0; x < w; x++ {
					blurLine(tmp[x*3:], wide[x*3:], h, w*3, r)
				}
			}
		}
		out := make([]uint16, n)
		for o, v := range wide {
			out[o] = uint16(min(v, 255<<8)) // white and beyond clamp; the fraction stays
		}
		g.pix[k] = out
		g.ready.Store(int32(k))
	}
}

// blurLine box-blurs the three channels of one line of n pixels, step
// values apart, from src into dst; the window for a pixel covers r on
// each side with the line's ends repeated.
func blurLine(src, dst []uint32, n, step, r int) {
	if r > n-1 {
		r = n - 1
	}
	span := 2*r + 1
	for ch := 0; ch < 3; ch++ {
		sum := int(src[ch]) * (r + 1)
		for i := 1; i <= r; i++ {
			sum += int(src[i*step+ch])
		}
		for i := 0; i < n; i++ {
			dst[i*step+ch] = uint32(sum / span)
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

// saverOrigin is the left edge of the first word for a travel: the word
// enters at the right edge one pixel a frame and, once it is in, the next
// copy follows it edge to edge (the letters carry their own side bearings,
// so the seam is an ordinary letter gap). The value stays within one word
// of the left edge however long the saver runs.
func saverOrigin(travel, w, word int) int {
	if travel < w {
		return w - travel
	}
	return -((travel - w) % word)
}

func (a *App) paintSaver(c *gfx.Canvas) {
	m := a.saverMask(c.H())
	// Safe-zone insets don't clip this overlay.
	x0 := saverOrigin(a.saver.travel, c.W(), m.Rect.Dx())
	shade, fade := a.saver.shade, a.saver.fade
	if shade <= 0 || shade > 256 {
		shade, fade = a.look.Shade, 256 // a saver frame set up outside tickSaver (tests, previews)
	}
	s := &a.saver
	if !a.saverCached(c) {
		a.saverCapture(c)
	}
	// the ground: the two levels either side of the fade, mixed, at the
	// shade; a level the worker has not finished yet is not shown
	g := s.ground
	n := int(g.ready.Load())
	lo, frac := fade*saverLevels>>8, fade*saverLevels&255
	if lo >= n {
		lo, frac = n, 0
	}
	dither := a.look.Dither != 0
	if lo == saverLevels {
		// dark: the last level at the shade, made once
		if s.dark == nil {
			s.dark = append([]uint8(nil), s.sharp...) // the alpha bytes
			saverPaintGround(s.dark, c.W(), c.H(), c.Stride, g.pix[saverLevels], nil, 0, shade, dither)
		}
		copy(c.Pix, s.dark)
	} else {
		var to []uint16
		if frac != 0 {
			to = g.pix[lo+1]
		}
		// the sharp picture is plain graphics: it only rounds
		saverPaintGround(c.Pix, c.W(), c.H(), c.Stride, g.pix[lo], to, frac, shade, dither && (lo > 0 || frac > 0))
	}
	if s.leaving {
		c.DirtyAll() // no lettering on the way back
		return
	}
	// the lettering, from the first word's edge to the right of the screen
	centres := make([]int, len(saverLights))
	for i, l := range saverLights {
		centres[i] = int(l.at*float64(c.W())) + int(l.tilt*float64(c.H())/2)
	}
	word := m.Rect.Dx()
	for y := 0; y < c.H(); y++ {
		for x := max(x0, 0); x < c.W(); x++ {
			v := m.Pix[y*m.Stride+(x-x0)%word]
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
