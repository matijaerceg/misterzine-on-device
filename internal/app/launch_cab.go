package app

import (
	"image"
	"image/color"
	"math"
	"sort"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// The launch animation (rough first cut): the screen cuts to black, a
// low-poly arcade cabinet spins in from far away until it fills the frame,
// then the camera pushes slowly into its monitor, which shows the game's
// title shot and fades to black. Then the core loads. Back cancels.
//
// Everything is a pure function of elapsed time, so the harness records it
// frame by frame with -motion.
const (
	cabSpinDur = 700 * time.Millisecond // spinning approach
	cabPushDur = 600 * time.Millisecond // straight push into the monitor
	cabFadeDur = 400 * time.Millisecond // monitor fade, at the end of the push
	cabTurns   = 2                      // whole turns during the approach
	cabTexW    = 128                    // title shot texture size
	cabTexH    = 96
)

type launchCab struct {
	active   bool
	at, next time.Time
	path     string
	req      ImageReq // the title shot; fetched while the spin runs if not cached
	tex      *image.RGBA
	elapsed  time.Duration
}

// startLaunchCab plays the animation before launching path, or launches at
// once when animations are off (static hosts, tests, the preference).
func (a *App) startLaunchCab(row *data.Row, path string) {
	if !a.transition.enabled || !a.PageTransitions() {
		a.cfg.Launch(path)
		return
	}
	now := a.cfg.TimerNow()
	a.cab = launchCab{active: true, at: now, next: now, path: path}
	if key, slot := cabShot(row); key != "" {
		a.cab.req = ImageReq{Key: key, Slot: slot, W: cabTexW, H: cabTexH, Stretch: slot != "system"}
		a.cab.tex, _ = a.cfg.Images.Get(a.cab.req)
		if a.cab.tex == nil {
			a.cfg.Images.Want([]ImageReq{a.cab.req})
		}
	}
	a.all = true
}

// cabShot picks the picture on the cabinet's monitor: the title screen,
// else a gameplay shot, else the system photo.
func cabShot(r *data.Row) (key, slot string) {
	if r == nil {
		return "", ""
	}
	if r.Img != "" {
		for _, want := range []string{"title", "snap", "ingame"} {
			for _, s := range r.ImgSlots {
				if s == want {
					return r.Img, s
				}
			}
		}
	}
	if r.Core != "" && !r.IsArcade() {
		return r.Core, "system"
	}
	return "", ""
}

func (a *App) LaunchCabRunning() bool { return a.cab.active }

func (a *App) handleLaunchCab(ev platform.Event) bool {
	if !a.cab.active {
		return false
	}
	if ev.Pressed && ev.Key == platform.KeyBack {
		a.cab = launchCab{}
		a.all = true
	}
	return true
}

func (a *App) tickLaunchCab(now time.Time) bool {
	c := &a.cab
	if !c.active {
		return false
	}
	c.elapsed = now.Sub(c.at)
	if c.elapsed >= cabSpinDur+cabPushDur {
		path := c.path
		*c = launchCab{}
		a.all = true
		a.cfg.Launch(path)
		return true
	}
	if c.tex == nil && c.req.Key != "" {
		c.tex, _ = a.cfg.Images.Get(c.req)
	}
	c.next = now.Add(frameDur)
	return true
}

func (a *App) nextLaunchCabTick() time.Time { return a.cab.next }

// cabPose is the camera for elapsed: the cabinet's turn about its vertical
// axis, the focal length (zoom) and the monitor brightness.
func cabPose(elapsed time.Duration, w, h int) (angle, focal, bright float64) {
	fill := float64(h) * 3 / 2.2 // focal at which the cabinet's height fills the frame
	end := float64(w) * (cabCamera - 0.4) / 0.76 * 1.3
	if elapsed < cabSpinDur {
		t := float64(elapsed) / float64(cabSpinDur)
		angle = 2 * math.Pi * cabTurns * t
		focal = fill * math.Pow(4, t-1) // geometric zoom: constant perceived speed
		return angle, focal, 1
	}
	t := min(1, float64(elapsed-cabSpinDur)/float64(cabPushDur))
	focal = fill + (end-fill)*t
	bright = 1
	if left := cabSpinDur + cabPushDur - elapsed; left < cabFadeDur {
		bright = float64(left) / float64(cabFadeDur)
	}
	return 0, focal, bright
}

type vec3 struct{ x, y, z float64 }

type cabTri struct {
	p   [3]vec3
	uv  [3][2]float64
	col color.RGBA
	tex bool
	z   float64 // sort key after transform
}

// cabModel is the cabinet: an upright box with a marquee box on top, a
// control-panel wedge, and the monitor quad on the front. Units are
// arbitrary; the monitor's centre is the origin so the push ends on it.
func cabModel() []cabTri {
	body := color.RGBA{70, 78, 110, 255}
	marquee := color.RGBA{230, 190, 70, 255}
	panel := color.RGBA{60, 60, 70, 255}
	var tris []cabTri
	quad := func(col color.RGBA, tex bool, a, b, c, d vec3) {
		// a b c d counter-clockwise seen from outside
		tris = append(tris,
			cabTri{p: [3]vec3{a, b, c}, uv: [3][2]float64{{0, 1}, {1, 1}, {1, 0}}, col: col, tex: tex},
			cabTri{p: [3]vec3{a, c, d}, uv: [3][2]float64{{0, 1}, {1, 0}, {0, 0}}, col: col, tex: tex})
	}
	const hw, top, bot, fz, bz = 0.5, 0.65, -1.25, 0.4, -0.4
	// body box (no bottom)
	quad(body, false, vec3{-hw, bot, fz}, vec3{hw, bot, fz}, vec3{hw, top, fz}, vec3{-hw, top, fz})   // front
	quad(body, false, vec3{hw, bot, bz}, vec3{-hw, bot, bz}, vec3{-hw, top, bz}, vec3{hw, top, bz})   // back
	quad(body, false, vec3{-hw, bot, bz}, vec3{-hw, bot, fz}, vec3{-hw, top, fz}, vec3{-hw, top, bz}) // left
	quad(body, false, vec3{hw, bot, fz}, vec3{hw, bot, bz}, vec3{hw, top, bz}, vec3{hw, top, fz})     // right
	quad(body, false, vec3{-hw, top, fz}, vec3{hw, top, fz}, vec3{hw, top, bz}, vec3{-hw, top, bz})   // top
	// marquee box, protruding
	const mb, mt, mz = 0.42, top, fz + 0.08
	quad(marquee, false, vec3{-hw, mb, mz}, vec3{hw, mb, mz}, vec3{hw, mt, mz}, vec3{-hw, mt, mz})
	quad(body, false, vec3{-hw, mt, mz}, vec3{hw, mt, mz}, vec3{hw, mt, fz}, vec3{-hw, mt, fz})
	quad(body, false, vec3{-hw, mb, fz}, vec3{hw, mb, fz}, vec3{hw, mb, mz}, vec3{-hw, mb, mz})
	quad(body, false, vec3{-hw, mb, fz}, vec3{-hw, mb, mz}, vec3{-hw, mt, mz}, vec3{-hw, mt, fz})
	quad(body, false, vec3{hw, mb, mz}, vec3{hw, mb, fz}, vec3{hw, mt, fz}, vec3{hw, mt, mz})
	// control panel wedge
	const pb, pm, pt, pz = -0.55, -0.45, -0.38, fz + 0.22
	quad(panel, false, vec3{-hw, pb, fz}, vec3{hw, pb, fz}, vec3{hw, pm, pz}, vec3{-hw, pm, pz})
	quad(panel, false, vec3{-hw, pm, pz}, vec3{hw, pm, pz}, vec3{hw, pt, fz}, vec3{-hw, pt, fz})
	tris = append(tris,
		cabTri{p: [3]vec3{{-hw, pb, fz}, {-hw, pm, pz}, {-hw, pt, fz}}, col: body},
		cabTri{p: [3]vec3{{hw, pm, pz}, {hw, pb, fz}, {hw, pt, fz}}, col: body})
	// monitor, a hair in front of the face so it sorts on top
	const sw, sh, sz = 0.38, 0.285, fz + 0.005
	quad(color.RGBA{0, 0, 0, 255}, true, vec3{-sw, -sh, sz}, vec3{sw, -sh, sz}, vec3{sw, sh, sz}, vec3{-sw, sh, sz})
	return tris
}

var cabTris = cabModel()

const cabCamera = 3.0 // camera distance from the monitor centre

// paintLaunchCab draws the frame for the current elapsed time.
func (a *App) paintLaunchCab(c *gfx.Canvas) {
	w, h := c.W(), c.H()
	c.Fill(c.Rect, color.RGBA{0, 0, 0, 255})
	angle, focal, bright := cabPose(a.cab.elapsed, w, h)
	renderCab(c.RGBA, a.cab.tex, angle, focal, bright)
	c.DirtyAll()
}

type screenVert struct {
	x, y, u, v float64
}

// renderCab projects and paints the model: back faces culled, the rest
// painted far to near, flat shaded from a fixed light.
func renderCab(dst *image.RGBA, tex *image.RGBA, angle, focal, bright float64) {
	w, h := dst.Rect.Dx(), dst.Rect.Dy()
	cx, cy := float64(w)/2, float64(h)/2
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	light := vec3{0.4, 0.7, 1}
	ln := math.Sqrt(light.x*light.x + light.y*light.y + light.z*light.z)
	light = vec3{light.x / ln, light.y / ln, light.z / ln}
	type drawn struct {
		v   [3]screenVert
		col color.RGBA
		tex bool
		z   float64
	}
	list := make([]drawn, 0, len(cabTris))
	for _, t := range cabTris {
		var p [3]vec3
		for i, q := range t.p {
			p[i] = vec3{q.x*cosA + q.z*sinA, q.y, -q.x*sinA + q.z*cosA}
		}
		// outward normal
		e1 := vec3{p[1].x - p[0].x, p[1].y - p[0].y, p[1].z - p[0].z}
		e2 := vec3{p[2].x - p[0].x, p[2].y - p[0].y, p[2].z - p[0].z}
		n := vec3{e1.y*e2.z - e1.z*e2.y, e1.z*e2.x - e1.x*e2.z, e1.x*e2.y - e1.y*e2.x}
		view := vec3{-p[0].x, -p[0].y, cabCamera - p[0].z}
		if n.x*view.x+n.y*view.y+n.z*view.z <= 0 {
			continue
		}
		nl := math.Sqrt(n.x*n.x + n.y*n.y + n.z*n.z)
		shade := 0.3 + 0.7*max(0, (n.x*light.x+n.y*light.y+n.z*light.z)/nl)
		shade = math.Floor(shade*4+0.5) / 4 // four flat tones
		col := t.col
		if !t.tex {
			col = color.RGBA{uint8(float64(col.R) * shade), uint8(float64(col.G) * shade), uint8(float64(col.B) * shade), 255}
		}
		d := drawn{col: col, tex: t.tex}
		for i, q := range p {
			dz := cabCamera - q.z
			if dz < 0.05 {
				dz = 0.05
			}
			d.v[i] = screenVert{cx + focal*q.x/dz, cy - focal*q.y/dz, t.uv[i][0], t.uv[i][1]}
			d.z += q.z / 3
		}
		list = append(list, d)
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].z < list[j].z })
	for _, d := range list {
		if d.tex && tex != nil {
			fillTri(dst, d.v, d.col, tex, bright)
		} else {
			if d.tex {
				d.col = color.RGBA{uint8(40 * bright), uint8(40 * bright), uint8(48 * bright), 255}
			}
			fillTri(dst, d.v, d.col, nil, 1)
		}
	}
}

// fillTri scanline-fills a screen triangle, flat or affine textured.
func fillTri(dst *image.RGBA, v [3]screenVert, col color.RGBA, tex *image.RGBA, bright float64) {
	w, h := dst.Rect.Dx(), dst.Rect.Dy()
	if v[0].y > v[1].y {
		v[0], v[1] = v[1], v[0]
	}
	if v[1].y > v[2].y {
		v[1], v[2] = v[2], v[1]
	}
	if v[0].y > v[1].y {
		v[0], v[1] = v[1], v[0]
	}
	y0, y2 := int(math.Ceil(v[0].y)), int(math.Ceil(v[2].y))-1
	if y0 < 0 {
		y0 = 0
	}
	if y2 >= h {
		y2 = h - 1
	}
	var tw, th int
	var bf int
	if tex != nil {
		tw, th = tex.Rect.Dx(), tex.Rect.Dy()
		bf = int(bright * 256)
	}
	// edge interpolation helpers
	lerp := func(a, b screenVert, y float64) screenVert {
		if b.y == a.y {
			return a
		}
		t := (y - a.y) / (b.y - a.y)
		return screenVert{a.x + (b.x-a.x)*t, y, a.u + (b.u-a.u)*t, a.v + (b.v-a.v)*t}
	}
	for y := y0; y <= y2; y++ {
		fy := float64(y) + 0.5
		var l, r screenVert
		if fy < v[1].y {
			l, r = lerp(v[0], v[1], fy), lerp(v[0], v[2], fy)
		} else {
			l, r = lerp(v[1], v[2], fy), lerp(v[0], v[2], fy)
		}
		if l.x > r.x {
			l, r = r, l
		}
		x0, x1 := int(math.Ceil(l.x-0.5)), int(math.Ceil(r.x-0.5))-1
		if x0 < 0 {
			x0 = 0
		}
		if x1 >= w {
			x1 = w - 1
		}
		if x0 > x1 {
			continue
		}
		row := dst.Pix[y*dst.Stride:]
		if tex == nil {
			for x := x0; x <= x1; x++ {
				i := x * 4
				row[i], row[i+1], row[i+2], row[i+3] = col.R, col.G, col.B, 255
			}
			continue
		}
		span := r.x - l.x
		du, dv := 0.0, 0.0
		if span > 0 {
			du, dv = (r.u-l.u)/span, (r.v-l.v)/span
		}
		u := l.u + (float64(x0)+0.5-l.x)*du
		vv := l.v + (float64(x0)+0.5-l.x)*dv
		for x := x0; x <= x1; x++ {
			tx, ty := int(u*float64(tw)), int(vv*float64(th))
			tx, ty = max(0, min(tw-1, tx)), max(0, min(th-1, ty))
			s := tex.Pix[ty*tex.Stride+tx*4:]
			i := x * 4
			row[i], row[i+1], row[i+2], row[i+3] = uint8(int(s[0])*bf>>8), uint8(int(s[1])*bf>>8), uint8(int(s[2])*bf>>8), 255
			u += du
			vv += dv
		}
	}
}
