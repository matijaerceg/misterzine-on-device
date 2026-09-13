package app

import (
	"image"
	"math"
	"math/rand/v2"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// The screenshots saver (Options -> Saver style: screenshots) shows one
// arcade shot after another on the whole canvas, each wiped in from the
// side behind a soft, wandering edge, with the game's title on a small
// tab that moves from corner to corner. Start held on a shot plays that
// game: the hold brings the picture up to full brightness and fills a
// line under the tab; letting go fades it back down and the saver goes
// on with the shot it was about to show. Every other button wakes.
//
// The pool is the gameplay shot (the "snap" slot) of every arcade game in
// the catalogue, shuffled and cycled without repeats: not the title
// screens, and not the third slot, which is often a game-over screen. A
// shot the card lacks is downloaded when it is next, so online the cycle
// covers the site, and offline whatever is on the card. With nothing to
// show at all the run falls back to the lettering.

// saverStyles are Options -> Saver style: the lettering (the default) or
// the screenshots.
var saverStyles = []string{"word", "shots"}

// SaverStyle is Options -> Saver style: "word" (default) or "shots".
func (a *App) SaverStyle() string {
	if a.cfg.SaverStyle == "shots" {
		return "shots"
	}
	return "word"
}

// SaverBright is Options -> Saver brightness for the screenshots: "half"
// (default) or "full".
func (a *App) SaverBright() string {
	if a.cfg.SaverBright == "full" {
		return "full"
	}
	return "half"
}

const (
	saverShotDwell    = 12 * time.Second // a shot's time on screen before the next wipes in
	saverShotPatience = 10 * time.Second // how long past the dwell the next shot may take to arrive before it is skipped
	saverShotWipe     = 30               // frames a wipe takes: a second at the saver's pace
	saverShotRamp     = 12               // frames the hold's brightness ramp takes each way
	saverShotHold     = 2 * time.Second  // Start held this long plays the shot's game
	saverShotHalf     = 128              // half brightness, in 256ths
	saverShotOffset   = 12               // how far the title tab wanders from its corner, in pixels
	saverShotBandDiv  = 20               // the wipe's soft edge is this share of the width wide
)

// saverPick is one shot of the pool: a row and one of its slots.
type saverPick struct {
	row  int // index into the dataset's rows; -1 = nothing
	slot string
}

// saverCurve is one wipe's edge: two slow waves down the screen whose
// phases drift as the wipe travels, so the curve changes shape on the
// way, plus a little fixed roughness per line.
type saverCurve struct {
	a1, a2, f1, f2, p1, p2, d1, d2 float64
	jitter                         []int8
}

type saverShots struct {
	pool []saverPick
	pos  int // the next pick in the cycle
	rng  *rand.Rand

	cur, in  saverPick // on screen, and arriving next
	curFrame []uint8   // the canvas with the shot on it, as painted (black around)
	inFrame  []uint8
	curCol   rgb    // the tab's ink: the shot's dominant hue
	wanted   bool   // in's picture is registered with the provider
	version  int    // bumped at every swap
	shownAt  time.Time // when cur arrived: the dwell clock
	dueSince time.Time // when the dwell ran out with in not ready; zero while not waiting
	skips    int       // picks skipped in a row (missing or offline)

	wipe  int  // frames into the wipe, 0 = none
	back  bool // the wipe is turning back to its start (a hold began during it)
	dir   int  // +1: the new shot comes in from the left; -1 from the right
	curve saverCurve

	holdAt time.Time // Start went down; zero while up
	holdOK bool      // the shot's game is on the card, so the hold plays it
	bright int       // the picture's brightness now, in 256ths

	corner     int // the tab's corner: 0 top left, 1 bottom right, 2 top right, 3 bottom left
	offX, offY int // the tab's wander from that corner

	// the last frame composed, at these settings, so a frame that did not
	// change is a copy
	lit                            []uint8
	litVersion, litWipe, litBright int
}

// saverShotsStart sets the screenshots saver up for a run: the pool
// shuffled, the first shot on its way, a black screen until it arrives.
func (a *App) saverShotsStart(now time.Time) {
	s := &saverShots{cur: saverPick{row: -1}, in: saverPick{row: -1}, dir: 1}
	s.rng = rand.New(rand.NewPCG(uint64(now.UnixNano()), uint64(len(a.ds.Rows))))
	for i := range a.ds.Rows {
		r := &a.ds.Rows[i]
		if !r.IsArcade() || r.Img == "" {
			continue
		}
		for _, slot := range r.ImgSlots {
			if slot == "snap" {
				s.pool = append(s.pool, saverPick{i, slot})
			}
		}
	}
	s.shuffle()
	s.in = s.next()
	s.bright = a.saverShotRest()
	s.shownAt = now.Add(-saverShotDwell) // the first shot comes as soon as it is here
	s.corner = int(s.rng.IntN(4))
	a.saver.shots = s
}

func (s *saverShots) shuffle() {
	s.rng.Shuffle(len(s.pool), func(i, j int) { s.pool[i], s.pool[j] = s.pool[j], s.pool[i] })
	s.pos = 0
}

// next takes the next pick of the cycle, reshuffling at the end.
func (s *saverShots) next() saverPick {
	if len(s.pool) == 0 {
		return saverPick{row: -1}
	}
	if s.pos >= len(s.pool) {
		s.shuffle()
	}
	p := s.pool[s.pos]
	s.pos++
	return p
}

// saverShotRest is the brightness a shot rests at: half, or full when
// Options says so.
func (a *App) saverShotRest() int {
	if a.SaverBright() == "full" {
		return 256
	}
	return saverShotHalf
}

func (a *App) saverShotReq(p saverPick) ImageReq {
	c := a.logical
	return ImageReq{Key: a.ds.Rows[p.row].Img, Slot: p.slot, W: c.W(), H: c.H(), Native: true}
}

// saverShotsHolding reports Start down on the screenshots saver.
func (a *App) saverShotsHolding() bool {
	s := a.saver.shots
	return s != nil && !s.holdAt.IsZero()
}

// saverShotsPress starts the Start hold: the dwell clock restarts, a wipe
// under way turns back, the picture comes up to full brightness, and the
// tab says whether the game is on the card.
func (a *App) saverShotsPress(at time.Time) {
	s := a.saver.shots
	if s == nil || !s.holdAt.IsZero() {
		return
	}
	s.holdAt, s.shownAt, s.dueSince = at, at, time.Time{}
	s.back = s.wipe > 0
	s.holdOK = false
	if s.cur.row >= 0 {
		row := &a.ds.Rows[s.cur.row]
		entries := a.launchEntries(row, s.cur.row)
		pick := a.rememberedPickOf(row, s.cur.row)
		s.holdOK = a.cfg.Launch != nil && pick < len(entries) && entries[pick].ok
	}
	a.all = true
}

// saverShotsRelease ends a hold that did not reach the launch: the
// picture fades back down and the shot stays a full dwell longer; the
// wipe, if one was turned back, finishes turning back, and the shot it
// was bringing is still the next one.
func (a *App) saverShotsRelease(at time.Time) {
	s := a.saver.shots
	if s == nil || s.holdAt.IsZero() {
		return
	}
	s.holdAt, s.shownAt = time.Time{}, at
	a.all = true
}

// tickSaverShots moves the screenshots saver one frame: the brightness
// ramp, the wipe (forward, or back under a hold), the next shot's
// arrival, the hold's launch. True when the picture changed.
func (a *App) tickSaverShots(now time.Time) bool {
	s := a.saver.shots
	changed := false
	// the hold: brightness up, the line under the tab, the launch at the end
	holding := !s.holdAt.IsZero()
	target := a.saverShotRest()
	if holding {
		target = 256
		changed = true // the line under the tab grows every frame
		if now.Sub(s.holdAt) >= saverShotHold && s.holdOK {
			a.saverShotsLaunch()
			return true
		}
	}
	const step = (256 - saverShotHalf + saverShotRamp - 1) / saverShotRamp
	switch {
	case s.bright < target:
		s.bright = min(target, s.bright+step)
		changed = true
	case s.bright > target:
		s.bright = max(target, s.bright-step)
		changed = true
	}
	// the wipe
	switch {
	case s.wipe > 0 && s.back:
		s.wipe--
		s.back = s.wipe > 0
		changed = true
	case s.wipe > 0:
		s.wipe++
		changed = true
		if s.wipe >= saverShotWipe {
			a.saverShotsSwap(now)
		}
	case !holding:
		changed = a.saverShotsFetch(now) || changed
	}
	return changed
}

// saverShotsFetch (no wipe running, no hold) sees to the next shot: asks
// for its picture, skips one the site or the card lacks, and starts the
// wipe once the dwell is over and the picture is here. A picture that
// takes too long is skipped too. True when the wipe started.
func (a *App) saverShotsFetch(now time.Time) bool {
	s := a.saver.shots
	for tries := 0; tries <= len(s.pool); tries++ {
		if s.in.row < 0 {
			a.saverShotsFallback(now)
			return true
		}
		if s.inFrame != nil {
			break
		}
		req := a.saverShotReq(s.in)
		img, st := a.cfg.Images.Get(req)
		switch {
		case img != nil:
			s.inFrame = a.saverShotFrame(img)
			s.skips = 0
		case st == ImageLoading:
			if !s.wanted {
				a.cfg.Images.Want([]ImageReq{req})
				s.wanted = true
			}
			if !s.dueSince.IsZero() && now.Sub(s.dueSince) >= saverShotPatience {
				a.saverShotsSkip()
				continue
			}
		default: // missing on the site, or not on the card while offline
			a.saverShotsSkip()
			continue
		}
		break
	}
	if s.skips > len(s.pool) {
		a.saverShotsFallback(now)
		return true
	}
	if now.Sub(s.shownAt) < saverShotDwell {
		return false
	}
	if s.inFrame == nil {
		if s.dueSince.IsZero() {
			s.dueSince = now
		}
		return false
	}
	s.dueSince = time.Time{}
	s.wipe = 1
	s.curve = a.saverShotsCurve()
	return true
}

// saverShotsSkip passes over the next pick for the one after it.
func (a *App) saverShotsSkip() {
	s := a.saver.shots
	s.skips++
	s.in, s.inFrame, s.wanted, s.dueSince = s.next(), nil, false, time.Time{}
}

// saverShotsSwap ends a wipe: the arriving shot is the one on screen, the
// tab moves to the next corner with a fresh wander, and the pick after
// it is on its way. The next wipe comes from the other side.
func (a *App) saverShotsSwap(now time.Time) {
	s := a.saver.shots
	s.cur, s.curFrame = s.in, s.inFrame
	s.curCol = saverShotHue(s.inFrame, a.logical.W(), a.logical.H(), a.logical.Stride)
	s.in, s.inFrame, s.wanted = s.next(), nil, false
	s.wipe, s.dir = 0, -s.dir
	s.shownAt, s.dueSince = now, time.Time{}
	s.version++
	s.corner = (s.corner + 1) % 4
	s.offX, s.offY = int(s.rng.IntN(saverShotOffset+1)), int(s.rng.IntN(saverShotOffset+1))
}

// saverShotsLaunch plays the shot's game at the end of the hold: the
// cursor moves onto it when the view has it, the remembered version
// launches as from the list, and the saver is down. The Start release
// is swallowed like a wake's.
func (a *App) saverShotsLaunch() {
	s := a.saver.shots
	row := &a.ds.Rows[s.cur.row]
	a.moveToKey(row.K)
	a.saver.waking = map[saverKey]bool{{key: platform.KeyStart}: true}
	a.saverEnd()
	a.launchRow(row, s.cur.row, a.rememberedPickOf(row, s.cur.row))
}

// saverShotsFallback gives the run to the lettering: nothing to show.
func (a *App) saverShotsFallback(now time.Time) {
	a.saver.shots = nil
	a.saver.style = "word"
	a.saver.sharp, a.saver.ground, a.saver.dark = nil, nil, nil
	a.saver.started, a.saver.ticked = now, now
	a.all = true
}

// saverShotsCurve draws a fresh edge for a wipe.
func (a *App) saverShotsCurve() saverCurve {
	s := a.saver.shots
	w, h := float64(a.logical.W()), a.logical.H()
	f := func(lo, hi float64) float64 { return lo + (hi-lo)*s.rng.Float64() }
	c := saverCurve{
		a1: w * f(1.0/14, 1.0/10), a2: w * f(1.0/40, 1.0/24),
		f1: f(0.8, 1.6), f2: f(2.2, 3.5),
		p1: f(0, 2*math.Pi), p2: f(0, 2*math.Pi),
		d1: f(-math.Pi, math.Pi), d2: f(-math.Pi, math.Pi),
		jitter: make([]int8, h),
	}
	for y := range c.jitter {
		c.jitter[y] = int8(s.rng.IntN(5) - 2)
	}
	return c
}

// edges fills xe with the edge's position per line for a wipe at frame t
// of n coming from the left: the new picture shows to the left of it.
func (c *saverCurve) edges(xe []float64, w, h, t, n, margin int) {
	p := float64(t) / float64(n)
	base := -float64(margin) + p*float64(w+2*margin)
	for y := 0; y < h; y++ {
		v := float64(y) / float64(h)
		xe[y] = base + c.a1*math.Sin(2*math.Pi*c.f1*v+c.p1+c.d1*p) + c.a2*math.Sin(2*math.Pi*c.f2*v+c.p2+c.d2*p) + float64(c.jitter[y])
	}
}

// saverShotFrame composes a shot on a black canvas, centred, as the
// artwork view shows it.
func (a *App) saverShotFrame(img *image.RGBA) []uint8 {
	c := a.logical
	w, h := c.W(), c.H()
	out := make([]uint8, len(c.Pix))
	for i := 3; i < len(out); i += 4 {
		out[i] = 255
	}
	iw, ih := img.Rect.Dx(), img.Rect.Dy()
	dst := image.Rect((w-iw)/2, (h-ih)/2, (w-iw)/2+iw, (h-ih)/2+ih).Intersect(image.Rect(0, 0, w, h))
	for y := dst.Min.Y; y < dst.Max.Y; y++ {
		sy := y - (h-ih)/2
		sx := dst.Min.X - (w-iw)/2
		copy(out[y*c.Stride+dst.Min.X*4:y*c.Stride+dst.Max.X*4], img.Pix[img.PixOffset(img.Rect.Min.X+sx, img.Rect.Min.Y+sy):])
	}
	return out
}

// saverShotHue is the tab's ink for a shot: the hue most of its coloured
// pixels share, lightened so it reads on black. A shot with no colour to
// speak of gets the theme's text colour. Every shot lights a different
// set of phosphors under the tab that way.
func saverShotHue(frame []uint8, w, h, stride int) rgb {
	var bins [24]int
	for y := 0; y < h; y += 2 {
		for x := 0; x < w; x += 2 {
			i := y*stride + x*4
			r, g, b := int(frame[i]), int(frame[i+1]), int(frame[i+2])
			hi, lo := max(r, g, b), min(r, g, b)
			if hi < 48 || (hi-lo)*4 < hi {
				continue // too dark or too grey to have a hue
			}
			var hue float64
			d := float64(hi - lo)
			switch hi {
			case r:
				hue = math.Mod(float64(g-b)/d+6, 6)
			case g:
				hue = float64(b-r)/d + 2
			default:
				hue = float64(r-g)/d + 4
			}
			bins[int(hue*4)%24] += (hi - lo) * hi / 255
		}
	}
	best, weight := -1, 0
	for i, n := range bins {
		if n > weight {
			best, weight = i, n
		}
	}
	if best < 0 {
		return gen.Eva.Fg
	}
	// the bin's centre hue at moderate saturation and full value
	hue := (float64(best) + .5) / 4
	const sat = 0.55
	x := uint8(255 * (1 - sat*math.Abs(math.Mod(hue, 2)-1)))
	m := uint8(math.Round(255 * (1 - sat)))
	switch int(hue) {
	case 0:
		return rgb{R: 255, G: x, B: m, A: 255}
	case 1:
		return rgb{R: x, G: 255, B: m, A: 255}
	case 2:
		return rgb{R: m, G: 255, B: x, A: 255}
	case 3:
		return rgb{R: m, G: x, B: 255, A: 255}
	case 4:
		return rgb{R: x, G: m, B: 255, A: 255}
	}
	return rgb{R: 255, G: m, B: x, A: 255}
}

// paintSaverShots paints the screenshots saver: the shot on screen, the
// next one wiping over it, at the brightness, then the title tab.
func (a *App) paintSaverShots(c *gfx.Canvas) {
	s := a.saver.shots
	w, h := c.W(), c.H()
	if len(s.lit) != len(c.Pix) || s.litVersion != s.version || s.litWipe != s.wipe || s.litBright != s.bright {
		if len(s.lit) != len(c.Pix) {
			s.lit = make([]uint8, len(c.Pix))
		}
		cur, in := s.curFrame, s.inFrame
		if len(cur) != len(c.Pix) { // composed for another canvas (a rotation): black
			cur = nil
		}
		if len(in) != len(c.Pix) {
			in = nil
		}
		saverShotsCompose(s.lit, cur, in, w, h, c.Stride, s.wipe, s.dir, &s.curve, s.bright, a.look.Dither != 0)
		s.litVersion, s.litWipe, s.litBright = s.version, s.wipe, s.bright
	}
	copy(c.Pix, s.lit)
	a.paintSaverTab(c)
	c.DirtyAll()
}

// saverShotsCompose writes the frame: cur, or in where the wipe has
// reached, through a soft edge the width of a twentieth of the screen
// that the ordered dither breaks up pixel by pixel (no blend: each pixel
// is one picture or the other), then the brightness, dithered too. A
// nil picture is black.
func saverShotsCompose(dst, cur, in []uint8, w, h, stride, wipe, dir int, curve *saverCurve, bright int, dither bool) {
	black := func(row []uint8) {
		for i := range row {
			row[i] = 0
		}
		for i := 3; i < len(row); i += 4 {
			row[i] = 255
		}
	}
	line := func(src []uint8, y, x0, x1 int) {
		if x1 <= x0 {
			return
		}
		if src == nil {
			black(dst[y*stride+x0*4 : y*stride+x1*4])
			return
		}
		copy(dst[y*stride+x0*4:y*stride+x1*4], src[y*stride+x0*4:y*stride+x1*4])
	}
	band := max(w/saverShotBandDiv, 2)
	var xe []float64
	if wipe > 0 {
		xe = make([]float64, h)
		margin := band + int(curve.a1+curve.a2) + 3
		curve.edges(xe, w, h, wipe, saverShotWipe, margin)
	}
	for y := 0; y < h; y++ {
		if wipe <= 0 {
			line(cur, y, 0, w)
		} else {
			// the new picture is on the left of the edge (dir 1) or the
			// right of its mirror (dir -1); the band before the edge is
			// decided pixel by pixel
			e := xe[y]
			var lo, hi int
			if dir > 0 {
				lo, hi = int(math.Floor(e))-band, int(math.Ceil(e))
			} else {
				e = float64(w) - e
				lo, hi = int(math.Floor(e)), int(math.Ceil(e))+band
			}
			lo = min(max(lo, 0), w)
			hi = min(max(hi, lo), w)
			if dir > 0 {
				line(in, y, 0, lo)
				line(cur, y, hi, w)
			} else {
				line(cur, y, 0, lo)
				line(in, y, hi, w)
			}
			for x := lo; x < hi; x++ {
				d := e - float64(x)
				if dir < 0 {
					d = float64(x) - e
				}
				src := cur
				if int(d*256/float64(band)) > int(saverBayer[y&7][x&7]) {
					src = in
				}
				line(src, y, x, x+1)
			}
		}
		if bright < 256 {
			row := dst[y*stride : y*stride+w*4]
			for x := 0; x < w; x++ {
				i := x * 4
				row[i] = saverQuantize(int(row[i])*bright, x, y, dither)
				row[i+1] = saverQuantize(int(row[i+1])*bright, x, y, dither)
				row[i+2] = saverQuantize(int(row[i+2])*bright, x, y, dither)
			}
		}
	}
}

const saverShotNoCard = "not on the card"

// paintSaverTab draws the title of the shot on screen on a black tab in
// its corner of the safe zone, and under it, while Start is held, the
// line that fills up to the launch, or the word that the game is not on
// the card.
func (a *App) paintSaverTab(c *gfx.Canvas) {
	s := a.saver.shots
	if s.cur.row < 0 {
		return
	}
	root := a.lay.Root
	title := gfx.Fit(a.ds.Rows[s.cur.row].Title, a.sm.Cols(root.Dx()-saverShotOffset-6))
	tw, th := a.sm.Width(title)+6, a.sm.H+4
	strip := a.sm.H + 2
	right, bottom := s.corner == 1 || s.corner == 2, s.corner == 1 || s.corner == 3
	x, y := root.Min.X+s.offX, root.Min.Y+s.offY
	if right {
		x = root.Max.X - tw - s.offX
	}
	if bottom {
		y = root.Max.Y - th - strip - s.offY
	}
	c.Fill(image.Rect(x, y, x+tw, y+th), rgb{A: 255})
	c.Text(x+3, y+2, a.sm, title, s.curCol)
	if s.holdAt.IsZero() {
		return
	}
	sw := tw
	if !s.holdOK {
		sw = a.sm.Width(saverShotNoCard) + 6
	}
	sx := x
	if right {
		sx = x + tw - sw
	}
	c.Fill(image.Rect(sx, y+th, sx+sw, y+th+strip), rgb{A: 255})
	if !s.holdOK {
		c.Text(sx+3, y+th+1, a.sm, saverShotNoCard, gen.Eva.Muted)
		return
	}
	held := a.cfg.TimerNow().Sub(s.holdAt)
	bar := int(int64(sw-6) * int64(min(held, saverShotHold)) / int64(saverShotHold))
	c.Fill(image.Rect(sx+3, y+th+strip/2-1, sx+3+bar, y+th+strip/2+1), s.curCol)
}
