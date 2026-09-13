package app

import (
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// shotImages is a picture provider for the tests: a bitmap per key/slot,
// or a state for one it has no bitmap for (missing when neither).
type shotImages struct {
	have  map[string]*image.RGBA
	state map[string]ImageState
	wants []ImageReq
}

func (f *shotImages) Get(req ImageReq) (*image.RGBA, ImageState) {
	k := req.Key + "/" + req.Slot
	if img := f.have[k]; img != nil {
		return img, ImageReady
	}
	if st, ok := f.state[k]; ok {
		return nil, st
	}
	return nil, ImageMissing
}
func (f *shotImages) Want(reqs []ImageReq) { f.wants = append(f.wants, reqs...) }
func (f *shotImages) SetPaused(bool)       {}

func flat(w, h int, col color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = col.R, col.G, col.B, 255
	}
	return img
}

var (
	shotRed  = color.RGBA{R: 220, G: 30, B: 30, A: 255}
	shotBlue = color.RGBA{R: 30, G: 30, B: 220, A: 255}
)

// shotsApp is an app on the screenshots saver with two arcade games, a red
// shot and a blue one, both on the card.
func shotsApp() (*App, *shotImages, *time.Time, *[]string) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	imgs := &shotImages{have: map[string]*image.RGBA{
		"red/snap": flat(320, 240, shotRed), "blue/snap": flat(320, 240, shotBlue),
	}, state: map[string]ImageState{}}
	var launches []string
	a := New(Config{PhysW: 320, PhysH: 240, SafeInsetX: 15, SafeInsetY: 15, SaverStyle: "shots",
		Now:    func() time.Time { return now },
		Images: imgs,
		Launch: func(p string) { launches = append(launches, p) },
		Exists: func(string) bool { return true },
		Status: func(int) data.Status { return data.StatusCurrent },
	}, data.Ingest([]data.Row{
		{K: "red", Title: "Red Game", Base: "Arcade", MRA: "_Arcade/Red.mra", Img: "red", ImgSlots: []string{"snap"}},
		{K: "blue", Title: "Blue Game", Base: "Arcade", MRA: "_Arcade/Blue.mra", Img: "blue", ImgSlots: []string{"title", "snap", "ingame"}},
		{K: "titled", Title: "Title Only", Base: "Arcade", MRA: "_Arcade/Titled.mra", Img: "titled", ImgSlots: []string{"title", "ingame"}},
		{K: "nes", Title: "NES", Base: "Console", Core: "NES"},
	}, "test", now), nil)
	return a, imgs, &now, &launches
}

// frames runs n saver frames a frame apart and paints each.
func frames(a *App, clock *time.Time, n int) {
	for i := 0; i < n; i++ {
		*clock = clock.Add(saverFrame)
		a.SaverFrame(*clock)
		a.Paint()
	}
}

func centre(a *App) color.RGBA { return a.logical.RGBAAt(160, 120) }

func TestSaverShotsWipesEachShotInAndNamesIt(t *testing.T) {
	a, _, clock, _ := shotsApp()
	a.startSaver(*clock)
	s := a.saver.shots
	if s == nil || len(s.pool) != 2 || s.in.row < 0 { // the gameplay shots only: no title, no third slot, no console
		t.Fatalf("pool %+v", s)
	}
	for _, p := range s.pool {
		if p.slot != "snap" || a.ds.Rows[p.row].K == "titled" {
			t.Fatalf("pool holds %+v", p)
		}
	}
	first := s.in
	a.Paint()
	if c := centre(a); c != (color.RGBA{A: 255}) {
		t.Fatalf("before the first shot the screen is black, got %v", c)
	}
	// the first shot wipes in at once: half way, the new picture holds
	// the side it comes from and the old one the other
	frames(a, clock, 1)
	if s.wipe != 1 {
		t.Fatalf("the first wipe did not start: wipe %d", s.wipe)
	}
	frames(a, clock, saverShotWipe/2-1)
	want := shotRed
	if a.ds.Rows[first.row].K == "blue" {
		want = shotBlue
	}
	half := func(c color.RGBA) bool {
		near := func(v, w uint8) bool { return int(v) >= int(w)/2-1 && int(v) <= int(w)/2+1 }
		return near(c.R, want.R) && near(c.G, want.G) && near(c.B, want.B)
	}
	left, right := a.logical.RGBAAt(4, 120), a.logical.RGBAAt(315, 120)
	if half(left) == half(right) {
		t.Fatalf("mid-wipe both edges alike: %v %v", left, right)
	}
	frames(a, clock, saverShotWipe/2+1)
	if s.wipe != 0 || s.cur != first || s.in == first {
		t.Fatalf("after the wipe: wipe %d cur %+v in %+v", s.wipe, s.cur, s.in)
	}
	if !half(centre(a)) {
		t.Fatalf("the shot is not on screen at half brightness: %v", centre(a))
	}
	// the title tab: ink of the shot's hue, dimmed with the picture, on
	// black, inside the safe zone
	ink := saverShotDim(s.curCol, saverShotHalf)
	if ink == s.curCol || ink.B >= s.curCol.B && ink.R >= s.curCol.R {
		t.Fatalf("the ink is not dimmed: %v from %v", ink, s.curCol)
	}
	inked := 0
	for y := 0; y < 240; y++ {
		for x := 0; x < 320; x++ {
			if p := a.logical.RGBAAt(x, y); p == ink {
				inked++
				if x < 15 || x >= 305 || y < 15 || y >= 225 {
					t.Fatalf("tab ink outside the safe zone at %d,%d", x, y)
				}
			}
		}
	}
	if inked == 0 || s.curCol == (color.RGBA{A: 255}) {
		t.Fatalf("no title on the tab (ink %v, %d pixels)", s.curCol, inked)
	}
	// the dwell, then the next shot from the other side
	*clock = clock.Add(saverShotDwell - time.Second)
	a.SaverFrame(*clock)
	if s.wipe != 0 {
		t.Fatal("the next wipe started before the dwell was over")
	}
	*clock = clock.Add(time.Second)
	a.SaverFrame(*clock)
	if s.wipe != 1 || s.dir != -1 {
		t.Fatalf("the second wipe: wipe %d dir %d", s.wipe, s.dir)
	}
	frames(a, clock, saverShotWipe)
	if s.cur == first || s.wipe != 0 {
		t.Fatalf("the second shot did not arrive: cur %+v", s.cur)
	}
}

func TestSaverShotTypingSchedule(t *testing.T) {
	lines := []paneLine{{"ab", muted()}, {"c", muted()}}
	want := map[int]saverShotTyping{
		0:                                      {line: 0, chars: 0, cursor: true},
		1:                                      {line: 0, chars: 1, cursor: true},
		2:                                      {line: 0, chars: 2, cursor: true}, // the beat at the line's end
		1 + saverShotBeat:                      {line: 0, chars: 2, cursor: true},
		2 + saverShotBeat:                      {line: 1, chars: 0, cursor: true},
		3 + saverShotBeat:                      {line: 1, chars: 1, done: true, cursor: true},
		3 + saverShotBeat + saverShotBlink - 1: {line: 1, chars: 1, done: true, cursor: true},
		3 + saverShotBeat + saverShotBlink:     {line: 1, chars: 1, done: true, cursor: false},
		3 + saverShotBeat + 2*saverShotBlink:   {line: 1, chars: 1, done: true, cursor: true},
	}
	for f, w := range want {
		if got := saverShotTypingAt(lines, f); got != w {
			t.Errorf("frame %d: %+v, want %+v", f, got, w)
		}
	}
	if got := saverShotTypingAt(nil, 3); !got.done || !got.cursor {
		t.Errorf("no lines: %+v", got)
	}
}

func TestSaverShotsTypesThePaneLinesAndBlinksAfter(t *testing.T) {
	a, _, clock, _ := shotsApp()
	a.startSaver(*clock)
	s := a.saver.shots
	frames(a, clock, saverShotWipe) // the swap: the caption starts
	if s.wipe != 0 || s.cur.row < 0 || s.typed != 0 {
		t.Fatalf("wipe %d cur %+v typed %d", s.wipe, s.cur, s.typed)
	}
	// the pane's lines: the title in the shot's hue, then the card answer
	// and the kind, cut to the width
	row := &a.ds.Rows[s.cur.row]
	if len(s.lines) < 3 || s.lines[0].text != row.Title || s.lines[0].col != s.curCol || s.lines[1].text != "current build" || s.lines[2].text != "Arcade / "+a.ds.Der[s.cur.row].SrcShort {
		t.Fatalf("caption %+v", s.lines)
	}
	ink := saverShotDim(s.curCol, saverShotHalf)
	inked := func() int {
		n := 0
		for y := 0; y < 240; y++ {
			for x := 0; x < 320; x++ {
				if a.logical.RGBAAt(x, y) == ink {
					n++
				}
			}
		}
		return n
	}
	cell := a.sm.W // the underline cursor: a cell wide, a pixel tall
	if n := inked(); n != cell {
		t.Fatalf("at the swap only the cursor is inked: %d pixels, an underline is %d", n, cell)
	}
	// a character a frame
	frames(a, clock, 3)
	if got := saverShotTypingAt(s.lines, s.typed); got != (saverShotTyping{line: 0, chars: 3, cursor: true}) {
		t.Fatalf("after three frames: %+v", got)
	}
	if n := inked(); n <= cell {
		t.Fatalf("three characters typed but only %d pixels inked", n)
	}
	total := saverShotBeat * (len(s.lines) - 1)
	for _, ln := range s.lines {
		total += len(ln.text)
	}
	frames(a, clock, total-3)
	if got := saverShotTypingAt(s.lines, s.typed); !got.done || !got.cursor {
		t.Fatalf("after %d frames: %+v", total, got)
	}
	// the cursor blinks once the block is complete: off for a blink, then
	// on again, and nothing else on the box changes
	on := inked()
	frames(a, clock, saverShotBlink)
	if off := inked(); off != on-cell {
		t.Fatalf("the cursor did not go off: %d inked, %d with it on", off, on)
	}
	frames(a, clock, saverShotBlink)
	if n := inked(); n != on {
		t.Fatalf("the cursor did not come back: %d inked, want %d", n, on)
	}
	// the strips are the picture at half: under a hold, with the picture
	// at full, that is the shot's colour halved, inside the safe zone
	// only
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	frames(a, clock, saverShotRamp+2)
	shot := shotRed
	if a.ds.Rows[s.cur.row].K == "blue" {
		shot = shotBlue
	}
	halved := color.RGBA{R: shot.R >> 1, G: shot.G >> 1, B: shot.B >> 1, A: 255}
	shaded := 0
	for y := 0; y < 240; y++ {
		for x := 0; x < 320; x++ {
			if p := a.logical.RGBAAt(x, y); p == halved {
				shaded++
				if x < 15 || x >= 305 || y < 15 || y >= 225 {
					t.Fatalf("a strip reaches outside the safe zone at %d,%d", x, y)
				}
			}
		}
	}
	if shaded < a.sm.W*a.sm.H*len(s.lines) {
		t.Fatalf("the strips cover only %d pixels", shaded)
	}
	a.Handle(platform.Event{Key: platform.KeyStart, At: *clock})
	// the next wipe takes the caption away, and the shot it brings starts
	// a fresh one
	first := s.cur
	*clock = clock.Add(saverShotDwell)
	a.SaverFrame(*clock) // the wipe starts
	typedAt := s.typed
	frames(a, clock, 5)
	if s.wipe != 6 || s.typed != typedAt {
		t.Fatalf("during the wipe: wipe %d typed %d (was %d)", s.wipe, s.typed, typedAt)
	}
	frames(a, clock, saverShotWipe-6)
	if s.cur == first || s.typed != 0 || s.lines[0].text != a.ds.Rows[s.cur.row].Title {
		t.Fatalf("after the wipe: cur %+v typed %d caption %+v", s.cur, s.typed, s.lines)
	}
}

func TestSaverShotsHoldPlaysTheOutgoingShot(t *testing.T) {
	a, _, clock, launches := shotsApp()
	a.startSaver(*clock)
	s := a.saver.shots
	frames(a, clock, saverShotWipe+1)
	first := s.cur
	*clock = clock.Add(saverShotDwell)
	a.SaverFrame(*clock)
	frames(a, clock, 10)
	if s.wipe != 11 {
		t.Fatalf("wipe %d", s.wipe)
	}
	// Start goes down mid-wipe: the wipe turns back, the picture comes up
	// to full, the line under the tab grows
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	if !a.ScreensaverActive() || !a.saverShotsHolding() || !s.holdOK {
		t.Fatal("Start woke the saver instead of holding")
	}
	frames(a, clock, 5)
	if s.wipe != 6 || s.bright <= saverShotHalf {
		t.Fatalf("under the hold: wipe %d bright %d", s.wipe, s.bright)
	}
	frames(a, clock, 10)
	if s.wipe != 0 || s.bright != 256 || s.cur != first {
		t.Fatalf("hold: wipe %d bright %d cur %+v", s.wipe, s.bright, s.cur)
	}
	if c := centre(a); c.R < 200 && c.B < 200 {
		t.Fatalf("not at full brightness: %v", c)
	}
	// at two seconds the outgoing shot's game launches and the saver is down
	frames(a, clock, int(saverShotHold/saverFrame)-15+1) // 60 frames fall a few nanoseconds short of two seconds
	if len(*launches) != 1 || (*launches)[0] != a.ds.Rows[first.row].MRA {
		t.Fatalf("launches %v, want %s", *launches, a.ds.Rows[first.row].MRA)
	}
	if a.ScreensaverActive() || a.saver.active || a.CursorKey() != a.ds.Rows[first.row].K {
		t.Fatalf("after the launch: active %v cursor %q", a.saver.active, a.CursorKey())
	}
	a.Handle(platform.Event{Key: platform.KeyStart, At: *clock})
	a.Tick(clock.Add(time.Second))
	if len(*launches) != 1 || a.Screen() != ScreenList {
		t.Fatalf("the Start release acted: launches %d screen %v", len(*launches), a.Screen())
	}
}

func TestSaverShotsReleasedHoldResumesTheIncomingShot(t *testing.T) {
	a, _, clock, launches := shotsApp()
	a.startSaver(*clock)
	s := a.saver.shots
	frames(a, clock, saverShotWipe+1)
	first, second := s.cur, s.in
	*clock = clock.Add(saverShotDwell)
	a.SaverFrame(*clock)
	frames(a, clock, 10)
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	frames(a, clock, 4)
	released := *clock
	a.Handle(platform.Event{Key: platform.KeyStart, At: *clock})
	if a.saverShotsHolding() || !a.ScreensaverActive() {
		t.Fatal("the release did not end the hold, or woke the saver")
	}
	// the wipe keeps turning back to its start, the brightness comes down
	frames(a, clock, 20)
	if s.wipe != 0 || s.bright != saverShotHalf || s.cur != first || s.in != second {
		t.Fatalf("after the release: wipe %d bright %d cur %+v in %+v", s.wipe, s.bright, s.cur, s.in)
	}
	// a full dwell from the release, then the same shot comes in
	*clock = released.Add(saverShotDwell - time.Second)
	a.SaverFrame(*clock)
	if s.wipe != 0 {
		t.Fatal("the dwell did not restart at the release")
	}
	*clock = clock.Add(time.Second)
	a.SaverFrame(*clock)
	frames(a, clock, saverShotWipe)
	if s.cur != second || len(*launches) != 0 {
		t.Fatalf("cur %+v want %+v, launches %v", s.cur, second, *launches)
	}
}

func TestSaverShotsHoldOnAGameNotOnTheCardLaunchesNothing(t *testing.T) {
	a, _, clock, launches := shotsApp()
	a.cfg.Exists = func(string) bool { return false }
	a.startSaver(*clock)
	s := a.saver.shots
	frames(a, clock, saverShotWipe+1)
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	if s.holdOK {
		t.Fatal("hold on a game not on the card counts as launchable")
	}
	frames(a, clock, int(saverShotHold/saverFrame)+5)
	if len(*launches) != 0 || !a.ScreensaverActive() || !a.saverShotsHolding() {
		t.Fatalf("launches %v active %v holding %v", *launches, a.ScreensaverActive(), a.saverShotsHolding())
	}
	// the tab says so: the muted text is on screen while the hold lasts
	found := false
	for y := 0; y < 240 && !found; y++ {
		for x := 0; x < 320; x++ {
			if a.logical.RGBAAt(x, y) == saverShotDim(muted(), 256) { // the hold brought the picture, and the tab, up to full
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("the not-on-the-card word is not on screen")
	}
	a.Handle(platform.Event{Key: platform.KeyStart, At: *clock})
	if a.saverShotsHolding() || !a.ScreensaverActive() {
		t.Fatal("release")
	}
}

func muted() color.RGBA { return color.RGBA{R: 162, G: 147, B: 199, A: 255} }

func TestSaverShotsOtherButtonsWakeWithoutActing(t *testing.T) {
	for _, key := range []platform.Key{platform.KeyEnter, platform.KeyBack, platform.KeyDown, platform.KeySpace} {
		a, _, clock, launches := shotsApp()
		a.startSaver(*clock)
		frames(a, clock, saverShotWipe+1)
		a.Handle(platform.Event{Key: key, Pressed: true, At: *clock})
		a.Handle(platform.Event{Key: key, At: *clock})
		a.Tick(clock.Add(time.Second))
		if a.ScreensaverActive() || a.saver.active || a.Screen() != ScreenList || len(*launches) != 0 {
			t.Fatalf("%v: active %v screen %v launches %v", key, a.saver.active, a.Screen(), *launches)
		}
		a.Paint()
		if c := centre(a); c == (color.RGBA{A: 255}) {
			t.Fatal("the list was not painted back")
		}
	}
}

func TestSaverShotsFullBrightnessOption(t *testing.T) {
	a, _, clock, _ := shotsApp()
	a.cfg.SaverBright = "full"
	a.startSaver(*clock)
	s := a.saver.shots
	frames(a, clock, saverShotWipe+1)
	if s.bright != 256 {
		t.Fatalf("bright %d", s.bright)
	}
	c := centre(a)
	if c.R < 200 && c.B < 200 {
		t.Fatalf("not full: %v", c)
	}
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	frames(a, clock, 3)
	a.Handle(platform.Event{Key: platform.KeyStart, At: *clock})
	frames(a, clock, 3)
	if s.bright != 256 {
		t.Fatalf("the hold changed a full brightness: %d", s.bright)
	}
}

func TestSaverShotsSkipsMissingWaitsForLoadingAndFallsBack(t *testing.T) {
	a, imgs, clock, _ := shotsApp()
	delete(imgs.have, "red/snap")
	delete(imgs.have, "blue/snap")
	imgs.state["blue/snap"] = ImageLoading
	a.startSaver(*clock)
	s := a.saver.shots
	frames(a, clock, 1)
	if s.in.slot != "snap" || len(imgs.wants) != 1 || imgs.wants[0].Key != "blue" || imgs.wants[0].W != 320 {
		t.Fatalf("in %+v wants %+v", s.in, imgs.wants)
	}
	if a.saver.shots == nil {
		t.Fatal("fell back while a picture is on its way")
	}
	// the picture is here: the wipe starts
	imgs.have["blue/snap"] = flat(320, 240, shotBlue)
	frames(a, clock, 2)
	if s.wipe == 0 {
		t.Fatal("the picture's arrival did not start the wipe")
	}
	// nothing at all: the lettering takes the run
	a2, imgs2, clock2, _ := shotsApp()
	imgs2.have = nil
	a2.startSaver(*clock2)
	frames(a2, clock2, 1)
	if a2.saver.shots != nil || a2.saver.style != "word" || !a2.saver.active {
		t.Fatalf("no fallback: shots %v style %q active %v", a2.saver.shots != nil, a2.saver.style, a2.saver.active)
	}
	a2.SaverSettle()
	frames(a2, clock2, 2)
	if a2.saver.ground == nil {
		t.Fatal("the lettering did not take the screen")
	}
	// a picture that never comes is skipped after the patience; with the
	// other one missing that exhausts the pool
	a3, imgs3, clock3, _ := shotsApp()
	imgs3.have = nil
	imgs3.state["blue/snap"], imgs3.state["red/snap"] = ImageLoading, ImageLoading
	a3.startSaver(*clock3)
	frames(a3, clock3, 1)
	*clock3 = clock3.Add(saverShotPatience + time.Second)
	frames(a3, clock3, 1)
	if a3.saver.shots == nil || a3.saver.shots.skips != 1 {
		t.Fatalf("patience: shots %v", a3.saver.shots)
	}
}

func TestSaverShotsComposeWipeCoversFromOneSideToTheOther(t *testing.T) {
	const w, h, stride = 64, 32, 64 * 4
	cur := make([]uint8, w*h*4)
	in := make([]uint8, w*h*4)
	for i := 0; i < len(in); i += 4 {
		in[i], in[i+1], in[i+2], in[i+3] = 255, 255, 255, 255
		cur[i+3] = 255
	}
	curve := &saverCurve{a1: 4, a2: 2, f1: 1, f2: 3, jitter: make([]int8, h)}
	dst := make([]uint8, w*h*4)
	for _, dir := range []int{1, -1} {
		last := -1
		for wipe := 0; wipe <= saverShotWipe; wipe++ {
			saverShotsCompose(dst, cur, in, w, h, stride, wipe, dir, curve, 256, true)
			white := 0
			for i := 0; i < len(dst); i += 4 {
				if dst[i] == 255 {
					white++
				} else if dst[i] != 0 {
					t.Fatalf("a blended pixel: %d", dst[i])
				}
			}
			if white < last {
				t.Fatalf("dir %d wipe %d: the new picture shrank from %d to %d pixels", dir, wipe, last, white)
			}
			last = white
			if wipe == 0 && white != 0 {
				t.Fatalf("dir %d: %d new pixels before the wipe", dir, white)
			}
			if wipe == saverShotWipe && white != w*h {
				t.Fatalf("dir %d: %d of %d pixels at the end", dir, white, w*h)
			}
			if wipe == saverShotWipe/2 {
				// the new picture is on its side of the screen
				from, to := 0, w-1
				if dir < 0 {
					from, to = to, from
				}
				if dst[(h/2)*stride+from*4] != 255 || dst[(h/2)*stride+to*4] != 0 {
					t.Fatalf("dir %d mid-wipe: from-side %d to-side %d", dir, dst[(h/2)*stride+from*4], dst[(h/2)*stride+to*4])
				}
			}
		}
	}
	// half brightness: the white picture at 128, dithered by a step at most
	saverShotsCompose(dst, in, nil, w, h, stride, 0, 1, curve, saverShotHalf, true)
	for i := 0; i < len(dst); i += 4 {
		if dst[i] < 127 || dst[i] > 128 {
			t.Fatalf("half of white is %d", dst[i])
		}
	}
	// a nil picture is black
	saverShotsCompose(dst, nil, nil, w, h, stride, 0, 1, curve, 256, true)
	if dst[0] != 0 || dst[3] != 255 {
		t.Fatalf("black %v", dst[:4])
	}
}

func TestSaverShotHueFollowsThePicture(t *testing.T) {
	c := a4Canvas()
	fill := func(col color.RGBA) []uint8 {
		f := flat(c.w, c.h, col)
		return f.Pix
	}
	blue := saverShotHue(fill(shotBlue), c.w, c.h, c.w*4)
	red := saverShotHue(fill(shotRed), c.w, c.h, c.w*4)
	if blue.B != 255 || blue.R >= blue.B || red.R != 255 || red.B >= red.R || blue == red {
		t.Fatalf("blue %v red %v", blue, red)
	}
	grey := saverShotHue(fill(color.RGBA{R: 90, G: 90, B: 90, A: 255}), c.w, c.h, c.w*4)
	if grey != rgb(color.RGBA{R: 233, G: 227, B: 246, A: 255}) {
		t.Fatalf("grey picture: %v", grey)
	}
}

type a4 struct{ w, h int }

func a4Canvas() a4 { return a4{32, 16} }

// Options: the style row sits under Screensaver, the brightness row only
// with the screenshots, A on either previews, and the choices are saved.
func TestSaverShotsOptionsRows(t *testing.T) {
	a, _, clock, _ := shotsApp()
	a.cfg.SaverStyle = ""
	saved := 0
	a.cfg.SettingsChanged = func() { saved++ }
	a.openPanel(ScreenOptions)
	find := func(kind string) int {
		for i, e := range a.panel.entries {
			if e.kind == kind {
				return i
			}
		}
		return -1
	}
	style := find("saver-style")
	if style != find("screensaver")+1 || find("saver-bright") != -1 {
		t.Fatalf("rows: screensaver %d style %d bright %d", find("screensaver"), style, find("saver-bright"))
	}
	a.panel.cursor = style
	a.actPanel(platform.KeyRight)
	if a.SaverStyle() != "shots" || find("saver-bright") != style+1 || saved != 1 {
		t.Fatalf("style %q bright row %d saved %d", a.SaverStyle(), find("saver-bright"), saved)
	}
	a.panel.cursor = style + 1
	a.actPanel(platform.KeyRight)
	if a.SaverBright() != "full" || saved != 2 {
		t.Fatalf("bright %q saved %d", a.SaverBright(), saved)
	}
	a.Handle(platform.Event{Key: platform.KeyEnter, Pressed: true, At: *clock})
	a.Handle(platform.Event{Key: platform.KeyEnter, At: *clock})
	if !a.ScreensaverActive() || a.saver.shots == nil || a.saver.shots.bright != 256 {
		t.Fatal("A on the brightness row did not preview the screenshots at full")
	}
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: *clock})
	if a.Screen() != ScreenOptions || a.ScreensaverActive() {
		t.Fatal("the preview's wake navigated away")
	}
	a.panel.cursor = style
	a.actPanel(platform.KeyLeft)
	if a.SaverStyle() != "word" || find("saver-bright") != -1 {
		t.Fatal("the brightness row stayed with the lettering")
	}
}
