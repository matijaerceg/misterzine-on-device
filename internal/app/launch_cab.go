package app

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"math"
	"path"
	"sort"
	"strconv"
	"strings"
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
	cabTurns   = 1                      // whole turns during the approach
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
	fill := float64(h) * cabCamera / (cabBounds.height * 1.15) // the cabinet's height nearly fills the frame
	end := float64(w) * cabCamera / cabScreenW * 1.3           // the screen overfills it
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
	p    [3]vec3
	uv   [3][2]float64
	col  color.RGBA
	tex  bool        // the screen: shows the title shot
	skin *image.RGBA // a painted face: its material's map_Kd texture
	z    float64     // sort key after transform
}

// The cabinet is cab.obj next to this file, a Wavefront OBJ as Blockbench
// exports it (File > Export > OBJ), with cab.mtl for the face colours.
// The face whose material is "screen" shows the title shot; every other
// material takes its Kd colour and is flat shaded. Faces are wound
// counter-clockwise seen from outside, which OBJ and Blockbench both do.
// The model is moved so the screen's centre is the origin (the push ends
// there) and scaled so the screen is cabScreenW wide.
//
//go:embed cab*
var cabFiles embed.FS

const cabScreenW = 0.76

type cabModel struct {
	tris          []cabTri
	height, width float64 // extent after scaling, for the fill focal
}

var cabTris, cabBounds = loadCabModel()

func loadCabModel() ([]cabTri, cabModel) {
	obj, _ := cabFiles.ReadFile("cab.obj")
	mtl, _ := cabFiles.ReadFile("cab.mtl")
	return parseCabModel(string(obj), string(mtl))
}

func parseCabModel(obj, mtl string) ([]cabTri, cabModel) {
	colours := map[string]color.RGBA{}
	skins := map[string]*image.RGBA{}
	name := ""
	for _, line := range strings.Split(mtl, "\n") {
		f := strings.Fields(line)
		switch {
		case len(f) == 2 && f[0] == "newmtl":
			name = f[1]
		case len(f) == 4 && f[0] == "Kd" && name != "":
			var c [3]float64
			for i := range c {
				c[i], _ = strconv.ParseFloat(f[i+1], 64)
			}
			colours[name] = color.RGBA{uint8(c[0] * 255), uint8(c[1] * 255), uint8(c[2] * 255), 255}
		case len(f) == 2 && f[0] == "map_Kd" && name != "":
			if img := loadCabSkin(f[1]); img != nil {
				skins[name] = img
			}
		}
	}
	var verts []vec3
	var uvs [][2]float64
	var tris []cabTri
	mat := ""
	for _, line := range strings.Split(obj, "\n") {
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		switch f[0] {
		case "v":
			if len(f) >= 4 {
				x, _ := strconv.ParseFloat(f[1], 64)
				y, _ := strconv.ParseFloat(f[2], 64)
				z, _ := strconv.ParseFloat(f[3], 64)
				verts = append(verts, vec3{x, y, z})
			}
		case "vt":
			if len(f) >= 3 {
				u, _ := strconv.ParseFloat(f[1], 64)
				v, _ := strconv.ParseFloat(f[2], 64)
				uvs = append(uvs, [2]float64{u, 1 - v}) // OBJ v runs upward
			}
		case "usemtl":
			if len(f) >= 2 {
				mat = f[1]
			}
		case "f":
			var p []vec3
			var uv [][2]float64
			for _, ref := range f[1:] {
				parts := strings.Split(ref, "/")
				vi, _ := strconv.Atoi(parts[0])
				if vi < 1 || vi > len(verts) {
					continue
				}
				p = append(p, verts[vi-1])
				t := [2]float64{}
				if len(parts) > 1 {
					if ti, _ := strconv.Atoi(parts[1]); ti >= 1 && ti <= len(uvs) {
						t = uvs[ti-1]
					}
				}
				uv = append(uv, t)
			}
			tex := strings.EqualFold(mat, "screen")
			col, ok := colours[mat]
			if !ok {
				col = color.RGBA{128, 128, 128, 255}
			}
			var skin *image.RGBA
			if !tex {
				skin = skins[mat]
			}
			for i := 1; i+1 < len(p); i++ { // fan triangulation
				tris = append(tris, cabTri{p: [3]vec3{p[0], p[i], p[i+1]}, uv: [3][2]float64{uv[0], uv[i], uv[i+1]}, col: col, tex: tex, skin: skin})
			}
		}
	}
	// centre on the screen and scale it to cabScreenW
	var lo, hi, slo, shi vec3
	first, sfirst := true, true
	grow := func(lo, hi *vec3, first *bool, p vec3) {
		if *first {
			*lo, *hi, *first = p, p, false
			return
		}
		lo.x, lo.y, lo.z = min(lo.x, p.x), min(lo.y, p.y), min(lo.z, p.z)
		hi.x, hi.y, hi.z = max(hi.x, p.x), max(hi.y, p.y), max(hi.z, p.z)
	}
	for _, t := range tris {
		for _, p := range t.p {
			grow(&lo, &hi, &first, p)
			if t.tex {
				grow(&slo, &shi, &sfirst, p)
			}
		}
	}
	if sfirst { // no screen: centre on the model
		slo, shi = lo, hi
	}
	centre := vec3{(slo.x + shi.x) / 2, (slo.y + shi.y) / 2, shi.z}
	scale := 1.0
	if w := shi.x - slo.x; w > 0 {
		scale = cabScreenW / w
	}
	for i := range tris {
		for j := range tris[i].p {
			p := tris[i].p[j]
			tris[i].p[j] = vec3{(p.x - centre.x) * scale, (p.y - centre.y) * scale, (p.z - centre.z) * scale}
		}
	}
	return tris, cabModel{tris: tris, height: (hi.y - lo.y) * scale, width: (hi.x - lo.x) * scale}
}

// loadCabSkin decodes a texture the mtl names, from the embedded files;
// nil when it is not there. Blockbench exports the path as written in
// the project, so only the base name counts.
func loadCabSkin(name string) *image.RGBA {
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	b, err := cabFiles.ReadFile(name)
	if err != nil {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil
	}
	rgba := image.NewRGBA(img.Bounds().Sub(img.Bounds().Min))
	draw.Draw(rgba, rgba.Rect, img, img.Bounds().Min, draw.Src)
	return rgba
}

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
		v     [3]screenVert
		col   color.RGBA
		tex   bool
		skin  *image.RGBA
		shade float64
		z     float64
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
		if !t.tex && t.skin == nil {
			col = color.RGBA{uint8(float64(col.R) * shade), uint8(float64(col.G) * shade), uint8(float64(col.B) * shade), 255}
		}
		d := drawn{col: col, tex: t.tex, skin: t.skin, shade: shade}
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
			if d.skin != nil {
				fillTri(dst, d.v, d.col, d.skin, d.shade)
			} else {
				fillTri(dst, d.v, d.col, nil, 1)
			}
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
	var dim [256]uint8 // the fade, as a lookup instead of a multiply a channel
	if tex != nil {
		tw, th = tex.Rect.Dx(), tex.Rect.Dy()
		bf = int(bright * 256)
		if bf != 256 {
			for i := range dim {
				dim[i] = uint8(i * bf >> 8)
			}
		}
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
			// one pixel, then doubling copies across the span
			span := row[x0*4 : (x1+1)*4]
			span[0], span[1], span[2], span[3] = col.R, col.G, col.B, 255
			for n := 4; n < len(span); n *= 2 {
				copy(span[n:], span[:n])
			}
			continue
		}
		// 16.16 fixed-point texture walk, in texel units
		// The texel coordinates at both span ends are clamped once, so the
		// walk between them never leaves the texture and needs no clamp
		// per pixel (the interpolation is monotonic).
		span := max(r.x-l.x, 1e-9)
		at := func(x int) (int, int) {
			t := (float64(x) + 0.5 - l.x) / span
			u := (l.u + (r.u-l.u)*t) * float64(tw)
			v := (l.v + (r.v-l.v)*t) * float64(th)
			return int(max(0, min(float64(tw)-1, u)) * 65536), int(max(0, min(float64(th)-1, v)) * 65536)
		}
		u, vv := at(x0)
		u1, v1 := at(x1)
		du, dv := 0, 0
		if n := x1 - x0; n > 0 {
			du, dv = (u1-u)/n, (v1-vv)/n
		}
		pix, stride := tex.Pix, tex.Stride
		if bf == 256 {
			for x := x0; x <= x1; x++ {
				o := (vv>>16)*stride + (u>>16)*4
				i := x * 4
				row[i], row[i+1], row[i+2], row[i+3] = pix[o], pix[o+1], pix[o+2], 255
				u += du
				vv += dv
			}
			continue
		}
		for x := x0; x <= x1; x++ {
			o := (vv>>16)*stride + (u>>16)*4
			i := x * 4
			row[i], row[i+1], row[i+2], row[i+3] = dim[pix[o]], dim[pix[o+1]], dim[pix[o+2]], 255
			u += du
			vv += dv
		}
	}
}
