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
// bits of fraction, and the final write posterises every channel to Bits
// bits (1 to 8) with Dither (1) choosing between the two nearest of those
// coarse steps by an ordered pattern of Cell by Cell thresholds (2, 4 or
// 8): the retro look of a display with few colours, whose dither is plain
// to see, and against the banding of the dark ground's gradients on a CRT.
// The whole saver frame goes through it: the sharp picture on the first
// frame, the blur levels, the fade, and the lettering's glints. At 8 bits
// the pattern only spreads the last bit and is all but invisible. The
// debug API sets a look live (SetSaverLook) so it can be tuned on a CRT.
type SaverLook struct {
	Knee, Gain, Shade, BlurDiv, Passes, Dither, Bits, Cell int
}

// DefaultSaverLook was chosen on a CRT with the debug tuner (2026-09-13):
// only the brightest parts bloom, six times their excess, the ground shows
// at 57%, and the blur radius is a 15th of the width (21 px on 320). The
// depth is three bits a channel (eight steps, so the dark ground spans two
// or three of them) dithered in a 4x4 cell, the plainest classic pattern.
var DefaultSaverLook = SaverLook{Knee: 171, Gain: 1538, Shade: 145, BlurDiv: 15, Passes: 2, Dither: 1, Bits: 3, Cell: 4}

// clamped keeps a look inside what the capture can do.
func (l SaverLook) clamped() SaverLook {
	l.Knee = min(max(l.Knee, 0), 255)
	l.Gain = min(max(l.Gain, 0), 8192)
	l.Shade = min(max(l.Shade, 16), 256)
	l.BlurDiv = min(max(l.BlurDiv, 4), 256)
	l.Passes = min(max(l.Passes, 1), 4)
	l.Dither = min(max(l.Dither, 0), 1)
	l.Bits = min(max(l.Bits, 1), 8)
	switch {
	case l.Cell < 3:
		l.Cell = 2
	case l.Cell < 6:
		l.Cell = 4
	default:
		l.Cell = 8
	}
	return l
}

// saverBayerCell is the ordered dither's threshold order for a cell of
// n by n (2, 4 or 8): the classic Bayer matrix, each pixel's rank from 0
// to n*n-1. It is the same every frame on purpose; a pattern that changes
// crawls on a phosphor.
func saverBayerCell(n int) [][]int {
	m := [][]int{{0}}
	for len(m) < n {
		k := len(m)
		next := make([][]int, 2*k)
		for y := range next {
			next[y] = make([]int, 2*k)
		}
		for y := 0; y < k; y++ {
			for x := 0; x < k; x++ {
				next[y][x] = 4 * m[y][x]
				next[y][x+k] = 4*m[y][x] + 2
				next[y+k][x] = 4*m[y][x] + 3
				next[y+k][x+k] = 4*m[y][x] + 1
			}
		}
		m = next
	}
	return m
}

// saverBayer is the 8x8 ordered dither as a threshold per pixel in 256ths
// of one 8-bit step (the screenshots saver's wipe edge and brightness).
var saverBayer = func() (t [8][8]uint16) {
	m := saverBayerCell(8)
	for y := range t {
		for x := range t[y] {
			t[y][x] = uint16(m[y][x])*4 + 2
		}
	}
	return
}()

// saverDither turns a channel with eight bits of fraction into the byte
// the canvas takes at a depth of a few bits: the value is scaled to the
// steps, so white is exactly the top one, the pixel's threshold is added
// and the step is read off, then spread back over the byte. Across a
// cell the steps average to the value, so the picture keeps its
// brightness. With the dither off every threshold is half a step, which
// rounds.
type saverDither struct {
	top    int        // the top step: 2^bits - 1
	mul    uint32     // top * 257: a channel times it, over 65536, is its level in 8.8 (a divide by 255 costs a board 15 ms a frame)
	thresh [8][8]int  // the pattern tiled over 8x8, in 256ths of a step
	out    [256]uint8 // each step as a byte, up to top; sized so a masked index needs no check
}

// newSaverDither builds the quantiser for bits a channel and a cell of
// n by n, or a rounding one with on false.
func newSaverDither(bits, cell int, on bool) *saverDither {
	d := &saverDither{top: 1<<bits - 1, mul: uint32(1<<bits-1) * 257}
	m := saverBayerCell(cell)
	for y := range d.thresh {
		for x := range d.thresh[y] {
			d.thresh[y][x] = 128
			if on {
				// rank m of n*n cells: (m + 0.5) / (n*n) of a step
				d.thresh[y][x] = (2*m[y%cell][x%cell] + 1) * 256 / (2 * cell * cell)
			}
		}
	}
	for q := 0; q <= d.top; q++ {
		d.out[q] = uint8((q*255 + d.top/2) / d.top)
	}
	return d
}

// dither is the look's quantiser.
func (l SaverLook) dither() *saverDither { return newSaverDither(l.Bits, l.Cell, l.Dither != 0) }

// level scales one channel v, over 0..255<<8, to its step in 8.8: white
// is exactly top<<8 (257/65536 falls short of 1/255 by a 65536th, which
// the constant makes up before the floor).
func (d *saverDither) level(v int) int { return int((uint32(v)*d.mul + 1<<16 - 1) >> 16) }

// byte quantises one channel v, over 0..255<<8, for the pixel at x, y.
func (d *saverDither) byte(v, x, y int) uint8 {
	return d.out[(d.level(v)+d.thresh[y&7][x&7])>>8]
}

// shaded folds a shade (256ths) into the quantiser: the copy takes a
// channel as if it had been dimmed by the shade first. The ground is
// painted through it, one multiply a channel: dimming first and scaling
// after cost the boards ten milliseconds more a fade frame.
func (d *saverDither) shaded(shade int) *saverDither {
	s := *d
	s.mul = (uint32(shade)*d.mul + 128) >> 8
	return &s
}

// saverQuantize is the plain 8-bit quantiser: rounded, or dithered by the
// 8x8 pattern's last-bit threshold (the screenshots saver's brightness).
func saverQuantize(v, x, y int, dither bool) uint8 {
	t := 128
	if dither {
		t = int(saverBayer[y&7][x&7])
	}
	return uint8(min((v+t)>>8, 255))
}

// saverPaintGround writes the ground into an RGBA frame: the level from,
// or its mix with to by frac (256ths), through d (shaded already).
func saverPaintGround(dst []uint8, w, h, stride int, from, to []uint16, frac int, d *saverDither) {
	// the quantiser spelled out (saverDither.byte) with its parts hoisted:
	// a fade frame writes every channel of the screen inside its 33 ms
	mul, out := d.mul, &d.out
	for y := 0; y < h; y++ {
		row, line, src := &d.thresh[y&7], dst[y*stride:y*stride+w*4], from[y*w*3:y*w*3+w*3]
		if to == nil {
			for x := 0; x < w; x++ {
				t, i, o := row[x&7], x*4, x*3
				for ch := 0; ch < 3; ch++ {
					line[i+ch] = out[((int((uint32(src[o+ch])*mul+1<<16-1)>>16)+t)>>8)&255]
				}
			}
			continue
		}
		mix := to[y*w*3 : y*w*3+w*3]
		for x := 0; x < w; x++ {
			t, i, o := row[x&7], x*4, x*3
			for ch := 0; ch < 3; ch++ {
				s := int(src[o+ch])
				v := s + (int(mix[o+ch])-s)*frac>>8 // one multiply: the mix is on the same chain as the quantiser
				line[i+ch] = out[((int((uint32(v)*mul+1<<16-1)>>16)+t)>>8)&255]
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

var saverValues = []string{"1", "2", "5", "10"}

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
	travel    int           // the lettering's position: one pixel per frame once dark
	ticks     int           // saver frames this run
	shade     int           // brightness of the screen under the lettering, in 256ths
	fade      int           // how far the fade has come, in 256ths; 256 once dark
	// the picture under the lettering, frozen at the first saver frame:
	// as painted, its blur levels, and the last level at the shade once
	// the fade is done
	sharp  []uint8
	ground *saverGround
	dark   []uint8
	cacheW int // width the picture was taken at (a rotation swaps it)
	mask   *image.Alpha
	waking map[saverKey]bool
	// the run's style (Options -> Screensaver style, taken at the start) and the
	// screenshots saver's state while that is the style (saver_shots.go)
	style string
	shots *saverShots
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

func (a *App) SaverEnabled() bool { return !a.cfg.SaverDisabled && a.cfg.Screensaver != "off" }

func (a *App) saverDelay() time.Duration {
	if !a.SaverEnabled() {
		return 0
	}
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
	if a.saver.shots != nil && ev.Key == platform.KeyStart {
		// the screenshots saver: Start is a hold, never a wake
		if ev.Pressed {
			a.saverShotsPress(at)
		} else {
			a.saverShotsRelease(at)
		}
		return true
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
	s.ticks = 0
	s.sharp, s.ground, s.dark = nil, nil, nil
	s.style, s.shots = a.SaverStyle(), nil
	if s.style == "shots" {
		a.saverShotsStart(now)
	}
	a.rep = repeater{}
	a.all = true
}

// SaverStats reports the run's frame count and the lettering's position
// (the debug API).
func (a *App) SaverStats() map[string]int {
	s := &a.saver
	return map[string]int{"ticks": s.ticks, "travel": s.travel}
}

// SaverRunning reports the saver painting: fading in, showing the
// lettering, or fading back out. The host then paces frames by the
// vertical blank (SaverFrame) instead of the timer.
func (a *App) SaverRunning() bool { return a.saver.active }

// SaverFrame is Tick for the host's vertical-blank loop: the saver moves
// one frame whatever the clock says, and everything else that is due runs
// as usual.
func (a *App) SaverFrame(now time.Time) bool {
	if a.saver.active {
		a.saver.next = now
	}
	return a.Tick(now)
}

// saverLeave starts a wake: the fade runs backwards over saverWake without
// the lettering, and the screen under it is painted again once the picture
// is back. With nothing frozen yet there is nothing to fade back from, and
// the screenshots saver leaves with a cut.
func (a *App) saverLeave(now time.Time) {
	s := &a.saver
	if s.ground == nil || s.shots != nil {
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
	s.shots = nil
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
		a.saver.ticks++
		changed := true
		if a.saver.shots != nil {
			changed = a.tickSaverShots(now) // a launch at the end of a hold takes the saver down
		} else if a.saver.style != "dim" {
			a.saverAdvance(now)
			if a.saver.leaving && a.saver.fadeT <= 0 {
				a.saverEnd() // the picture is back
			}
		}
		if a.saver.active {
			// the next frame keeps the grid the last one was on: a tick
			// that ran late does not push every later one back, which
			// used to add up until the lettering skipped a column. A
			// whole frame lost starts a fresh grid from now.
			a.saver.next = a.saver.next.Add(saverFrame)
			if a.saver.next.Before(now) {
				a.saver.next = now.Add(saverFrame)
			}
			if changed {
				a.all = true
			}
		}
	}
	return true
}

// saverAdvance moves the fade for now and sets the shade from it, and
// moves the lettering one pixel a frame once the fade is done. The fade
// climbs over saverFade, held back by a level the worker has not
// finished; a wake runs it back down over saverWake.
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
			s.fadeT = saverFade
			s.darkAt = now
			s.travel = 0
			s.fade, s.shade = 256, a.look.Shade
			return
		}
	default:
		// dark: a frame is a pixel. Frames, not the clock: the host paces
		// them by the vertical blank, so counting keeps the motion even
		// however the clock jitters.
		s.travel++
		return
	}
	s.fade = int(256 * s.fadeT / saverFade)
	s.shade = 256 - (256-a.look.Shade)*s.fade/256
	s.travel = 0
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
	if a.saver.style == "dim" {
		if len(a.saver.dark) != len(c.Pix) {
			a.saver.dark = append([]uint8(nil), c.Pix...)
			keep := 33
			if a.SaverDim() == "66" {
				keep = 66
			}
			for y := 0; y < c.H(); y++ {
				for x := 0; x < c.W(); x++ {
					i := c.PixOffset(x, y)
					for ch := 0; ch < 3; ch++ {
						a.saver.dark[i+ch] = uint8(int(a.saver.dark[i+ch]) * keep / 100)
					}
				}
			}
		}
		copy(c.Pix, a.saver.dark)
		c.DirtyAll()
		return
	}

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
	d := a.look.dither() // the whole frame at the look's depth, the sharp first picture included
	if lo == saverLevels {
		// dark: the last level at the shade, made once
		if s.dark == nil {
			s.dark = append([]uint8(nil), s.sharp...) // the alpha bytes
			saverPaintGround(s.dark, c.W(), c.H(), c.Stride, g.pix[saverLevels], nil, 0, d.shaded(shade))
		}
		copy(c.Pix, s.dark)
	} else {
		var to []uint16
		if frac != 0 {
			to = g.pix[lo+1]
		}
		saverPaintGround(c.Pix, c.W(), c.H(), c.Stride, g.pix[lo], to, frac, d.shaded(shade))
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
		row := m.Pix[y*m.Stride : y*m.Stride+word]
		sx := 0
		if x0 < 0 {
			sx = -x0 % word
		}
		for x := max(x0, 0); x < c.W(); x++ {
			v := row[sx]
			if sx++; sx == word {
				sx = 0
			}
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
			// the glint's gradient at the ground's depth, so one pattern
			// covers the frame
			c.Pix[i], c.Pix[i+1], c.Pix[i+2] = d.byte(min(r, saverGlintMax)<<8, x, y), d.byte(min(g, saverGlintMax)<<8, x, y), d.byte(min(b, saverGlintMax)<<8, x, y)
		}
	}
	c.DirtyAll()
}
