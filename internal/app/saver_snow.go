package app

import (
	"math"
	"sort"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

const snowRelease = 10 * time.Minute

type saverSnow struct {
	order    []int
	released int
	active   []int
	ground   []byte
}

// A spatial hash scatters equal and nearly equal shades without using a
// random generator in the frame loop. Rank spreads even a flat image's
// departures across the full ten minutes.
func snowHash(i int) uint32 {
	x := uint32(i) + 0x9e3779b9
	x = (x ^ (x >> 16)) * 0x85ebca6b
	x = (x ^ (x >> 13)) * 0xc2b2ae35
	return x ^ (x >> 16)
}

func snowStart(rank, count int) time.Duration {
	return snowRelease * time.Duration(rank) / time.Duration(max(1, count-1))
}

func (a *App) paintSaverSnow(c *gfx.Canvas) {
	s := &a.saver
	if s.snow == nil || !a.saverCached(c) {
		s.sharp = append([]byte(nil), c.Pix...)
		s.cacheW = c.W()
		n := c.W() * c.H()
		f := &saverSnow{order: make([]int, n), ground: append([]byte(nil), c.Pix...)}
		keys := make([]uint32, n)
		for i := range f.order {
			f.order[i] = i
			o := c.PixOffset(i%c.W(), i/c.W())
			// Integer luma, grouped into small bands; higher is brighter.
			luma := (77*int(s.sharp[o]) + 150*int(s.sharp[o+1]) + 29*int(s.sharp[o+2])) >> 8
			keys[i] = uint32(luma/4)<<26 | snowHash(i)>>6
		}
		sort.Slice(f.order, func(i, j int) bool { return keys[f.order[i]] > keys[f.order[j]] })
		snowScatter(f.order)
		s.snow = f
	}
	f := s.snow
	elapsed := max(time.Duration(0), s.ticked.Sub(s.started))
	for f.released < len(f.order) && snowStart(f.released, len(f.order)) <= elapsed {
		i := f.order[f.released]
		o := c.PixOffset(i%c.W(), i/c.W())
		for ch := 0; ch < 3; ch++ {
			f.ground[o+ch] = 0
		}
		f.active = append(f.active, f.released)
		f.released++
	}
	copy(c.Pix, f.ground)
	alive := f.active[:0]
	for _, rank := range f.active {
		i := f.order[rank]
		hash := snowHash(i)
		age := (elapsed - snowStart(rank, len(f.order))).Seconds()
		speed := 12 + float64(hash&255)*12/255
		y := i/c.W() + int(age*speed)
		if y >= c.H() {
			continue
		}
		alive = append(alive, rank)
		phase := float64((hash>>8)&255) * 2 * math.Pi / 255
		x := i%c.W() + int(age*(float64((hash>>16)&255)/255-0.5)*2+2*(math.Sin(age*1.7+phase)-math.Sin(phase)))
		if x < 0 || x >= c.W() {
			continue
		}
		src, dst := c.PixOffset(i%c.W(), i/c.W()), c.PixOffset(x, y)
		copy(c.Pix[dst:dst+3], s.sharp[src:src+3])
	}
	f.active = alive
	c.DirtyAll()
}

func (a *App) saverStyleHelp() string {
	if a.SaverStyle() == "snow" {
		return "Pixels scatter from bright to dark over 10 minutes, leaving black. Press " + a.btn("A") + " to preview; other buttons wake."
	}
	return "Lettering: the MISTERZINE sweep. Screenshots: arcade shots; hold Start to play. Dim: darken the current screen. Other buttons wake."
}

// Scatter sorted brightness ranks through a moving 20%-wide window.
// A random offset up to one fifth of the image makes nearby ranks overlap:
// the first departure must be in the brightest fifth, and the last in the
// darkest fifth. Sorting once keeps the animation loop allocation-free.
func snowScatter(order []int) {
	type candidate struct {
		pixel    int
		priority uint64
	}
	candidates := make([]candidate, len(order))
	width := uint64(max(1, (len(order)+4)/5))
	for rank, pixel := range order {
		candidates[rank] = candidate{pixel, uint64(rank)<<32 + uint64(snowHash(pixel+len(order)))*width}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].priority == candidates[j].priority {
			return candidates[i].pixel < candidates[j].pixel
		}
		return candidates[i].priority < candidates[j].priority
	})
	for i, c := range candidates {
		order[i] = c.pixel
	}
}
