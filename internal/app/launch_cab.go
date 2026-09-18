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
	"unsafe"

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
	cabSpinDur = 2000 * time.Millisecond // spinning approach
	cabPushDur = 1200 * time.Millisecond // straight push into the monitor
	cabFadeDur = 1000 * time.Millisecond // monitor fade, at the end of the push
	cabTurns   = 1                       // whole turns during the approach (0: straight in)
	cabFar     = 6.0                     // how many times smaller than frame-filling the approach starts
	cabTilt    = 14.0                    // degrees the cabinet leans toward the viewer by the end of the approach
	cabTexW    = 128                     // title shot texture size
	cabTexH    = 96
)

// cabHoldStart is how long Start stays down before a launch in the "hold"
// mode plays the animation; a shorter press launches plainly on release.
const cabHoldStart = 200 * time.Millisecond

// holdLaunch is a launch waiting on Start: the row and path, and when Start
// went down.
type holdLaunch struct {
	row  *data.Row
	path string
	at   time.Time
}

// LaunchTransition is the saved preference: "always" or "hold".
func (a *App) LaunchTransition() string {
	if a.cfg.LaunchTransition == "hold" {
		return "hold"
	}
	return "always"
}

// releaseHoldLaunch launches plainly when Start comes up before the hold
// matured; true when it did.
func (a *App) releaseHoldLaunch() bool {
	if a.holdLaunch.path == "" {
		return false
	}
	path := a.holdLaunch.path
	a.holdLaunch = holdLaunch{}
	a.cfg.Launch(path)
	return true
}

func (a *App) tickHoldLaunch(now time.Time) bool {
	h := &a.holdLaunch
	if h.path == "" || now.Sub(h.at) < cabHoldStart {
		return false
	}
	row, path := h.row, h.path
	*h = holdLaunch{}
	a.startLaunchCab(row, path)
	return true
}

func (a *App) nextHoldLaunchTick() time.Time {
	if a.holdLaunch.path == "" {
		return time.Time{}
	}
	return a.holdLaunch.at.Add(cabHoldStart)
}

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
	if !a.transition.enabled { // static hosts and tests without a clock
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

// previewLaunchCab plays the animation for the current row without
// launching anything (the Options row's Preview) and returns to the
// screen it started from.
func (a *App) previewLaunchCab() {
	row, _, _ := a.current()
	now := a.cfg.TimerNow()
	a.cab = launchCab{active: true, at: now, next: now}
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

// PreviewLaunchCab is the debug API's way to play the animation.
func (a *App) PreviewLaunchCab() { a.previewLaunchCab() }

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
		if path != "" { // a preview ends where it began
			a.cfg.Launch(path)
		}
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
func cabPose(elapsed time.Duration, w, h int) (angle, tilt, focal, bright float64) {
	fill := float64(h) * cabCamera / (cabBounds.height * 1.15) // the cabinet's height nearly fills the frame
	end := float64(w) * cabCamera / cabScreenW * 1.05          // the screen just fills it; more costs 60 fps on the boards
	if elapsed < cabSpinDur {
		t := float64(elapsed) / float64(cabSpinDur)
		e := 1 - (1-t)*(1-t)*(1-t) // the turn eases out: fast at first, settling to face front
		angle = 2 * math.Pi * cabTurns * e
		tilt = cabTilt * math.Pi / 180 * e   // leans forward as it comes closer
		focal = fill * math.Pow(cabFar, t-1) // geometric zoom: constant perceived speed
		return angle, tilt, focal, 1
	}
	t := min(1, float64(elapsed-cabSpinDur)/float64(cabPushDur))
	focal = fill + (end-fill)*t
	bright = 1
	if left := cabSpinDur + cabPushDur - elapsed; left < cabFadeDur {
		// full black two frames before the end, so the last frames shown
		// before the core loads are black and not nearly so
		bright = max(0, float64(left-2*frameDur)/float64(cabFadeDur-2*frameDur))
	}
	return 0, cabTilt * math.Pi / 180, focal, bright
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
	mat, group := "", false
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
		case "o", "g": // an element or group named screen is the monitor too
			group = len(f) >= 2 && strings.EqualFold(f[1], "screen")
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
			tex := group || strings.EqualFold(mat, "screen")
			if tex && len(p) >= 3 {
				// of a screen box only the face looking out the front is the
				// picture; its sides and back are ordinary faces
				e1 := vec3{p[1].x - p[0].x, p[1].y - p[0].y, p[1].z - p[0].z}
				e2 := vec3{p[2].x - p[0].x, p[2].y - p[0].y, p[2].z - p[0].z}
				nz := e1.x*e2.y - e1.y*e2.x
				nl := math.Sqrt(math.Pow(e1.y*e2.z-e1.z*e2.y, 2) + math.Pow(e1.z*e2.x-e1.x*e2.z, 2) + nz*nz)
				tex = nl > 0 && nz/nl > 0.5
			}
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
	// the screen's UVs point at its patch of the atlas; the picture wants
	// the whole face, so spread them over the screen faces' UV box
	ulo, uhi := [2]float64{math.Inf(1), math.Inf(1)}, [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, t := range tris {
		if t.tex {
			for _, uv := range t.uv {
				ulo[0], ulo[1] = min(ulo[0], uv[0]), min(ulo[1], uv[1])
				uhi[0], uhi[1] = max(uhi[0], uv[0]), max(uhi[1], uv[1])
			}
		}
	}
	for i := range tris {
		if tris[i].tex && uhi[0] > ulo[0] && uhi[1] > ulo[1] {
			for j := range tris[i].uv {
				uv := &tris[i].uv[j]
				uv[0], uv[1] = (uv[0]-ulo[0])/(uhi[0]-ulo[0]), (uv[1]-ulo[1])/(uhi[1]-ulo[1])
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
	// the spin axis stands on the screen's back edge, so the picture swings
	// round it rather than the picture's own face
	centre := vec3{(slo.x + shi.x) / 2, (slo.y + shi.y) / 2, slo.z}
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

// The camera's distance from the spin axis. Closer means more perspective;
// the zoom is scaled with it so the cabinet's size on screen stays put.
const cabCamera = 1.8

// paintLaunchCab draws the frame for the current elapsed time.
func (a *App) paintLaunchCab(c *gfx.Canvas) {
	w, h := c.W(), c.H()
	c.Fill(c.Rect, color.RGBA{0, 0, 0, 255})
	angle, tilt, focal, bright := cabPose(a.cab.elapsed, w, h)
	renderCab(c.RGBA, a.cab.tex, angle, tilt, focal, bright)
	c.DirtyAll()
}

type screenVert struct {
	x, y, u, v float64
}

// renderCab projects and paints the model: back faces culled, the rest
// painted far to near, flat shaded from a fixed light.
func renderCab(dst *image.RGBA, tex *image.RGBA, angle, tilt, focal, bright float64) {
	w, h := dst.Rect.Dx(), dst.Rect.Dy()
	cx, cy := float64(w)/2, float64(h)/2
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	sinT, cosT := math.Sin(tilt), math.Cos(tilt) // about x: the top comes toward the viewer
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
	list := make([]*drawn, 0, len(cabTris))
	for _, t := range cabTris {
		var p [3]vec3
		for i, q := range t.p {
			x, y, z := q.x*cosA+q.z*sinA, q.y, -q.x*sinA+q.z*cosA
			p[i] = vec3{x, y*cosT - z*sinT, y*sinT + z*cosT}
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
			col = color.RGBA{uint8(float64(col.R) * shade * bright), uint8(float64(col.G) * shade * bright), uint8(float64(col.B) * shade * bright), 255}
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
		if d.tex {
			d.z += 100 // the picture paints over whatever the model has behind the glass
		}
		// slivers and degenerate faces draw as stray lines: skip them
		if a := (d.v[1].x-d.v[0].x)*(d.v[2].y-d.v[0].y) - (d.v[2].x-d.v[0].x)*(d.v[1].y-d.v[0].y); math.Abs(a) < 1 {
			continue
		}
		list = append(list, &d)
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].z < list[j].z })
	var shotTexels *cabTexels
	if tex != nil {
		shotTexels = dimmedTexels(tex, bright)
	}
	for _, d := range list {
		if d.tex && tex != nil {
			fillTri(dst, d.v, d.col, shotTexels)
		} else {
			if d.tex {
				d.col = color.RGBA{uint8(40 * bright), uint8(40 * bright), uint8(48 * bright), 255}
			}
			if d.skin != nil {
				fillTri(dst, d.v, d.col, skinTexels(d.skin, d.shade*bright)) // the fade darkens the paint too
			} else {
				fillTri(dst, d.v, d.col, nil)
			}
		}
	}
}

// fillTri scanline-fills a screen triangle, flat or affine textured.
// cabTexels is a texture as whole pixels: the span writer moves one word
// per pixel, which the boards do several times faster than four bytes
// with their bounds checks. The bytes are RGBA in memory either way.
type cabTexels struct {
	px     []uint32
	w, h   int
	stride int // in pixels
}

func pixels32(pix []uint8) []uint32 {
	if len(pix) < 4 {
		return nil
	}
	return unsafe.Slice((*uint32)(unsafe.Pointer(&pix[0])), len(pix)/4)
}

func rgbaTexels(img *image.RGBA) *cabTexels {
	return &cabTexels{px: pixels32(img.Pix), w: img.Rect.Dx(), h: img.Rect.Dy(), stride: img.Stride / 4}
}

// The fade and the flat shades multiply the small texture once a frame
// instead of every screen pixel. One scratch buffer per distinct level.
var cabDimCache = map[*image.RGBA]map[int]*cabTexels{}

func dimmedTexels(img *image.RGBA, bright float64) *cabTexels {
	bf := int(bright * 256)
	if bf >= 256 {
		return rgbaTexels(img)
	}
	levels := cabDimCache[img]
	if levels == nil {
		levels = map[int]*cabTexels{}
		cabDimCache[img] = levels
	}
	if t := levels[bf]; t != nil {
		return t
	}
	if len(levels) > 96 { // a new texture or a long fade: start over
		clear(levels)
	}
	src := pixels32(img.Pix)
	dst := make([]uint32, len(src))
	stride := img.Stride / 4
	var dim [256]uint32
	for i := range dim {
		dim[i] = uint32(i * bf >> 8)
	}
	for i, p := range src {
		dst[i] = dim[p&255] | dim[p>>8&255]<<8 | dim[p>>16&255]<<16 | 255<<24
	}
	t := &cabTexels{px: dst, w: img.Rect.Dx(), h: img.Rect.Dy(), stride: stride}
	levels[bf] = t
	return t
}

// skinTexels shades a painted face: four flat tones, so four cached copies.
func skinTexels(img *image.RGBA, shade float64) *cabTexels { return dimmedTexels(img, shade) }

func fillTri(dst *image.RGBA, v [3]screenVert, col color.RGBA, tex *cabTexels) {
	w, h := dst.Rect.Dx(), dst.Rect.Dy()
	out := pixels32(dst.Pix)
	dstStride := dst.Stride / 4
	solid := uint32(col.R) | uint32(col.G)<<8 | uint32(col.B)<<16 | 255<<24
	if v[0].y > v[1].y {
		v[0], v[1] = v[1], v[0]
	}
	if v[1].y > v[2].y {
		v[1], v[2] = v[2], v[1]
	}
	if v[0].y > v[1].y {
		v[0], v[1] = v[1], v[0]
	}
	// rows whose centre lies inside the triangle (top-left rule), so the
	// edge interpolation never runs past a vertex
	y0, y2 := int(math.Ceil(v[0].y-0.5)), int(math.Ceil(v[2].y-0.5))-1
	if y0 < 0 {
		y0 = 0
	}
	if y2 >= h {
		y2 = h - 1
	}
	var tw, th int
	if tex != nil {
		tw, th = tex.w, tex.h
	}
	// edge interpolation helpers
	lerp := func(a, b screenVert, y float64) screenVert {
		if b.y == a.y {
			return a
		}
		t := max(0, min(1, (y-a.y)/(b.y-a.y)))
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
		row := out[y*dstStride : y*dstStride+w]
		if tex == nil {
			span := row[x0 : x1+1]
			for i := range span {
				span[i] = solid
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
		px, stride := tex.px, tex.stride
		out := row[x0 : x1+1]
		for i := range out {
			out[i] = px[(vv>>16)*stride+(u>>16)]
			u += du
			vv += dv
		}
	}
}
