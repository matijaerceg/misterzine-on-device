package app

import (
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

// SupportHooks keeps file/device access in the host. Opening the support menu
// only loads a saved report; raw observation begins with an explicit test.
type SupportHooks struct {
	Start  func(from, until time.Time)
	Finish func() support.Report
	Load   func() support.Report
	Launch func(game, target string) support.Report
	Pads   func() []support.Pad // the gamepads the input reader has open, for the pad tester
}

type supportView struct {
	mode                   string // menu, capture, result, launch, pad
	cursor, page           int
	from, until, next, now time.Time
	report                 support.Report
	game, target           string
	// the pad tester: the pads as read, the latest presses, and when B
	// went down (held two seconds it leaves the tester)
	pads    []support.Pad
	presses []padPress
	backAt  time.Time
}

// padPress is one button press seen by the pad tester.
type padPress struct {
	at     time.Time
	source string
	code   uint16
	key    platform.Key
}

const (
	padPressKeep = 12
	padTestLeave = 2 * time.Second
)

// startPadTest opens the pad tester: every press is listed with its pad,
// raw code, MiSTer define-slot and what the app does with it, and nothing
// else acts until B has been held for two seconds.
func (a *App) startPadTest() {
	v := &a.support
	v.mode, v.presses, v.backAt, v.next = "pad", nil, time.Time{}, time.Time{}
	v.pads = nil
	if h := a.cfg.Support; h != nil && h.Pads != nil {
		v.pads = h.Pads()
	}
	a.rep = repeater{}
	a.all = true
}

func (a *App) handlePadTest(ev platform.Event) bool {
	v := &a.support
	if ev.Key == platform.KeyNone && ev.Code == 0 {
		return false
	}
	if !ev.Pressed {
		delete(a.down, ev.Key)
		if ev.Key == platform.KeyBack {
			v.backAt, v.next = time.Time{}, time.Time{}
			a.all = true
		}
		return true
	}
	v.presses = append([]padPress{{at: ev.At, source: ev.Source, code: ev.Code, key: ev.Key}}, v.presses...)
	if len(v.presses) > padPressKeep {
		v.presses = v.presses[:padPressKeep]
	}
	if ev.Key == platform.KeyBack && v.backAt.IsZero() {
		v.backAt = ev.At
		v.next = ev.At.Add(padTestLeave)
	}
	a.all = true
	return true
}

// padActions is what the app does with each key, in the tester's words.
var padActions = map[platform.Key]string{
	platform.KeyEnter: "details / confirm", platform.KeyBack: "back / Options", platform.KeyTab: "Filters", platform.KeySpace: "view / favorite",
	platform.KeyStart: "launch", platform.KeyUp: "up", platform.KeyDown: "down", platform.KeyLeft: "left", platform.KeyRight: "right",
	platform.KeyPageUp: "page up (L)", platform.KeyPageDown: "page down (R)", platform.KeyHome: "top", platform.KeyEnd: "bottom",
	platform.KeyBackspace: "erase", platform.KeyOther: "no action",
}

// padPressLine describes one press: which pad, which raw button, which
// MiSTer slot that is, and the app's action. Presses translated by Main
// say so, since they carry no pad name.
func (a *App) padPressLine(p padPress, prev *padPress) string {
	gap := ""
	if prev != nil && !prev.at.IsZero() && !p.at.IsZero() {
		gap = fmt.Sprintf(" +%dms", p.at.Sub(prev.at).Milliseconds())
	}
	action := padActions[p.key]
	if action == "" {
		action = p.key.String()
	}
	if p.source == "MiSTer virtual input" {
		return "MiSTer translation: " + action + gap
	}
	for _, pad := range a.support.pads {
		if pad.Name != p.source {
			continue
		}
		s := shortName(pad.Name, 14) + fmt.Sprintf(" btn %d", p.code)
		if slot := pad.Slot(p.code); slot != "" {
			s += " = " + a.btn(slot)
		}
		return s + ": " + action + gap
	}
	source := p.source
	if source == "" {
		source = "other input"
	}
	return shortName(source, 14) + ": " + action + gap
}

// padLines describes one pad for the tester: its define-slots and where
// they came from.
func (a *App) padLines(p support.Pad) []string {
	head := p.Name
	if p.Vendor != 0 || p.Product != 0 {
		head += fmt.Sprintf(" %04x:%04x", p.Vendor, p.Product)
	}
	slots := ""
	for _, name := range []string{"A", "B", "X", "Y", "L", "R", "Select", "Start"} {
		if code, ok := p.Slots[name]; ok {
			label := name
			if len(name) == 1 {
				label = a.btn(name)
			}
			slots += fmt.Sprintf("%s %d  ", label, code)
		}
	}
	lines := []string{head}
	switch {
	case p.Mapped && len(slots) > 0:
		lines = append(lines, "  "+strings.TrimSpace(slots), "  read by MiSTer slot from "+shortName(p.Map, 40))
	case p.Mapped:
		lines = append(lines, "  no usable buttons in "+shortName(p.Map, 40))
	default:
		lines = append(lines, "  no MiSTer map: face buttons come through MiSTer's translation", "  "+strings.TrimSpace(slots))
	}
	if p.Note != "" {
		lines = append(lines, "  "+p.Note)
	}
	return lines
}

func shortName(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if i := strings.LastIndexByte(s, '/'); i >= 0 && len(s)-i-1 <= n {
		return s[i+1:]
	}
	return s[:n-1] + gfx.Ellipsis
}

func (a *App) OpenTroubleshooting() {
	a.screen = ScreenTroubleshooting
	a.rep = repeater{}
	a.notice = ""
	a.support = supportView{mode: "menu"}
	if h := a.cfg.Support; h != nil && h.Load != nil {
		a.support.report = h.Load()
	}
	a.all = true
}

func (a *App) supportCapturing() bool {
	return a.screen == ScreenTroubleshooting && a.support.mode == "capture"
}

func (a *App) startSupportTest() {
	h := a.cfg.Support
	if h == nil || h.Start == nil || h.Finish == nil {
		a.Notice("Controller test unavailable", 4*time.Second)
		return
	}
	now := a.cfg.TimerNow()
	a.support.now = now
	a.support.mode = "capture"
	a.support.from = now.Add(3 * time.Second)
	a.support.until = a.support.from.Add(6 * time.Second)
	a.support.next = now.Add(time.Second)
	a.rep = repeater{}
	h.Start(a.support.from, a.support.until)
	a.all = true
}

func (a *App) tickSupport(now time.Time) bool {
	if a.screen == ScreenTroubleshooting && a.support.mode == "pad" {
		v := &a.support
		if v.next.IsZero() || now.Before(v.next) {
			return false
		}
		v.mode, v.next, v.backAt = "menu", time.Time{}, time.Time{}
		a.all = true
		return true
	}
	if !a.supportCapturing() || now.Before(a.support.next) {
		return false
	}
	a.support.now = now
	if !now.Before(a.support.until) {
		a.support.report = a.cfg.Support.Finish()
		a.support.mode, a.support.page = "result", 0
		a.support.next = time.Time{}
	} else {
		a.support.next = now.Add(time.Second)
		if a.support.next.After(a.support.until) {
			a.support.next = a.support.until
		}
	}
	a.all = true
	return true
}

func (a *App) actSupport(k platform.Key) bool {
	v := &a.support
	if v.mode == "capture" {
		return false
	}
	if k == platform.KeyBack {
		if v.mode == "menu" {
			a.openPanel(ScreenOptions)
			for i, e := range a.panel.entries {
				if e.kind == "troubleshooting" {
					a.panel.cursor = i
					break
				}
			}
		} else {
			v.mode, v.page = "menu", 0
		}
		a.all = true
		return true
	}
	switch v.mode {
	case "menu":
		switch k {
		case platform.KeyUp:
			v.cursor = max(0, v.cursor-1)
		case platform.KeyDown:
			v.cursor = min(3, v.cursor+1)
		case platform.KeyEnter:
			switch v.cursor {
			case 0:
				a.startSupportTest()
			case 1:
				a.startPadTest()
			case 2:
				row, d, i := a.current()
				v.game, v.target = "", ""
				if row != nil {
					v.game = d.Title
					entries := a.launchEntries(row, i)
					if len(entries) > 0 {
						v.target = entries[0].path
					}
				}
				v.mode = "launch"
			case 3:
				if v.report.HasResult() {
					v.mode, v.page = "result", 0
				} else {
					a.Notice("No troubleshooting result yet", 4*time.Second)
				}
			}
		}
	case "launch":
		if k == platform.KeyEnter && v.target != "" && a.cfg.Support != nil && a.cfg.Support.Launch != nil {
			v.report = a.cfg.Support.Launch(v.game, v.target)
			v.mode, v.page = "result", 0
		}
	case "result":
		pages := a.supportPages()
		switch k {
		case platform.KeyRight, platform.KeyDown, platform.KeyPageDown:
			v.page = min(len(pages)-1, v.page+1)
		case platform.KeyLeft, platform.KeyUp, platform.KeyPageUp:
			v.page = max(0, v.page-1)
		}
	}
	a.all = true
	return true
}

func yesNo(b bool) string {
	if b {
		return "YES"
	}
	return "NO"
}

func deviceLines(d support.Device) []string {
	lines := []string{"Device: " + d.Name, fmt.Sprintf("ID %04x:%04x  bus %04x", d.Vendor, d.Product, d.Bus),
		fmt.Sprintf("%s  revision %04x", d.Node, d.Version), d.Route()}
	if d.StartMapping != "" {
		lines = append(lines, "Start mapping: "+d.StartMapping)
	}
	return lines
}

func signalLine(s support.Signal) string {
	if s.Type == 3 {
		return fmt.Sprintf("Axis %d: %d changes, last %d", s.Code, s.Changes, s.Value)
	}
	return fmt.Sprintf("Button %d: down %d / up %d", s.Code, s.Down, s.Up)
}

func (a *App) supportLines() []string {
	r := a.support.report
	lines := []string{"MisterZine " + r.Version}
	if r.Tested {
		lines = append(lines, "START BUTTON TEST", r.Conclusion(),
			"Button received: "+yesNo(r.ButtonReceived()), "App saw Start: "+yesNo(r.StartDown > 0),
			fmt.Sprintf("Start down %d / up %d", r.StartDown, r.StartUp))
		// Put the first physical button on page one, ahead of translated input
		// and axis noise. All other signals remain available on later pages.
		found := false
		for _, virtual := range []bool{false, true} {
			for _, s := range r.Signals {
				if s.Type != 1 || s.Down == 0 {
					continue
				}
				for _, d := range r.Devices {
					if d.Node == s.Node && d.Virtual == virtual {
						lines = append(lines, "", "FIRST BUTTON", signalLine(s))
						lines = append(lines, deviceLines(d)...)
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if found {
				break
			}
		}
		if r.Dropped > 0 {
			lines = append(lines, fmt.Sprintf("Incomplete capture: %d dropped", r.Dropped))
		}
	}
	if r.Launch.Result != "" {
		lines = append(lines, "", "GAME LAUNCH TEST", r.Launch.Game, r.Launch.Result)
		if r.Launch.Version != "" && r.Launch.Version != r.Version {
			lines = append(lines, "Launch build: "+r.Launch.Version)
		}
		if r.Launch.Detail != "" {
			lines = append(lines, r.Launch.Detail)
		}
		lines = append(lines, "Target: "+r.Launch.Target)
	}
	if r.Note != "" {
		lines = append(lines, "", r.Note)
	}
	lines = append(lines, "", "SYSTEM", "Linux "+r.Kernel, r.Display, "Data "+r.Data, "Recorded "+r.Created)
	for _, d := range r.Devices {
		lines = append(lines, "", "DEVICE EVIDENCE")
		lines = append(lines, deviceLines(d)...)
		lines = append(lines, "Standard Start capability: "+yesNo(d.StandardStart))
		n := 0
		for _, s := range r.Signals {
			if s.Node == d.Node {
				lines = append(lines, signalLine(s))
				if s.Repeat > 0 {
					lines = append(lines, fmt.Sprintf("Repeats: %d", s.Repeat))
				}
				n++
			}
		}
		if n == 0 {
			lines = append(lines, "No events during this test")
		}
	}
	return lines
}

// Paginate instead of truncating device names, codes or errors, including with
// maximum safe-zone margins. The frozen report is stable while it is photographed.
func (a *App) supportPages() [][]string {
	box := a.lay.Body.Inset(4)
	cols := max(8, a.sm.Cols(box.Dx()))
	count := max(1, (box.Dy()-a.sm.H-4)/(a.sm.H+1))
	var lines []string
	for _, text := range a.supportLines() {
		text = data.ASCII(strings.Join(strings.Fields(text), " "))
		if text == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, gfx.Wrap(text, cols, len(text)+1)...)
	}
	var pages [][]string
	for len(lines) > 0 {
		n := min(count, len(lines))
		pages = append(pages, lines[:n])
		lines = lines[n:]
	}
	if len(pages) == 0 {
		pages = [][]string{{"No result yet"}}
	}
	return pages
}

func (a *App) paintSupport(c *gfx.Canvas) {
	a.paintStatus(c)
	if a.notice == "" {
		c.Fill(a.lay.Status, gen.Eva.Surface)
		c.Text(a.lay.Status.Min.X+2, a.lay.Status.Min.Y+2, a.sm, "Troubleshooting", gen.Eva.Accent)
	}
	c.Box(a.lay.Body, gen.Eva.Line)
	box := a.lay.Body.Inset(4)
	y := box.Min.Y
	cols := max(8, a.sm.Cols(box.Dx()))
	write := func(text string, col rgb) {
		if text == "" {
			y += a.sm.H + 1
			return
		}
		for _, line := range gfx.Wrap(data.ASCII(text), cols, len(text)+1) {
			if y+a.sm.H > box.Max.Y {
				return
			}
			c.Text(box.Min.X, y, a.sm, line, col)
			y += a.sm.H + 1
		}
	}
	v := &a.support
	switch v.mode {
	case "menu":
		for i, label := range []string{"Test Start button", "Test pad buttons", "Test game launch", "Last troubleshooting result"} {
			col := gen.Eva.Fg
			if i == v.cursor {
				c.Fill(image.Rect(box.Min.X-1, y, box.Max.X+1, y+a.sm.H+2), gen.Eva.Surface)
				col = gen.Eva.Accent
			}
			write(label, col)
			y += 3
		}
		write("", gen.Eva.Fg)
		if v.cursor == 0 {
			write("Press your Start button when asked. The result stays on screen for a photo.", gen.Eva.Muted)
		}
		if v.cursor == 1 {
			write("See which pad sent each press, its raw button, the MiSTer slot it is defined in and what MisterZine does. Hold "+a.btn("B")+" two seconds to leave.", gen.Eva.Muted)
		}
		if v.cursor == 2 {
			write("Try the highlighted game's main version using "+a.btn("A")+" / Enter.", gen.Eva.Muted)
		}
		if v.cursor == 3 {
			write("Review the saved result, including after restarting MisterZine.", gen.Eva.Muted)
		}
		a.paintHint(c, "Up/Down choose  A open  B back")
	case "pad":
		write("PAD TEST", gen.Eva.Accent)
		y += 2
		if len(v.pads) == 0 {
			write("No gamepad is being read. A pad needs a Start button or a MiSTer map (define it in the MiSTer menu).", gen.Eva.Muted)
		}
		for _, p := range v.pads {
			for i, line := range a.padLines(p) {
				col := gen.Eva.Muted
				if i == 0 {
					col = gen.Eva.Fg
				}
				write(line, col)
			}
		}
		write("", gen.Eva.Fg)
		write("PRESSES, newest first", gen.Eva.Accent)
		y += 2
		if len(v.presses) == 0 {
			write("Press any button.", gen.Eva.Muted)
		}
		for i, p := range v.presses {
			var prev *padPress
			if i+1 < len(v.presses) {
				prev = &v.presses[i+1]
			}
			col := gen.Eva.Fg
			if i > 0 {
				col = gen.Eva.Muted
			}
			write(a.padPressLine(p, prev), col)
		}
		if !v.backAt.IsZero() {
			a.paintHint(c, "Keep holding "+a.btn("B")+" to leave")
		} else {
			a.paintHint(c, "Hold B 2 s to leave")
		}
	case "capture":
		now := v.now
		if now.Before(v.from) {
			write(fmt.Sprintf("Get ready... %d", max(1, int(v.from.Sub(now).Seconds()+0.999))), gen.Eva.Accent)
			write("", gen.Eva.Fg)
			write("Release all buttons. Wait for GO.", gen.Eva.Fg)
		} else {
			write("GO - press and release START", gen.Eva.Accent)
			write("", gen.Eva.Fg)
			write("Press only that button, once or twice.", gen.Eva.Fg)
			write(fmt.Sprintf("Result in %d seconds", max(1, int(v.until.Sub(now).Seconds()+0.999))), gen.Eva.Muted)
		}
		write("", gen.Eva.Fg)
		write("The test ends automatically. Game launching is paused here.", gen.Eva.Muted)
		a.paintHint(c, "Wait for the result")
	case "launch":
		write("TEST GAME LAUNCH", gen.Eva.Accent)
		write("", gen.Eva.Fg)
		if v.target == "" {
			write("No launch target selected. Return to the list and highlight an installed game first.", gen.Eva.Fg)
			a.paintHint(c, "B back")
		} else {
			write(v.game, gen.Eva.Fg)
			write("", gen.Eva.Fg)
			write(a.btn("A")+" / Enter will launch this game's main version now.", gen.Eva.Fg)
			write("", gen.Eva.Fg)
			write("If it works, tell the person helping you. The last result is kept when you reopen MisterZine.", gen.Eva.Muted)
			a.paintHint(c, "A launch now  B cancel")
		}
	case "result":
		pages := a.supportPages()
		v.page = min(v.page, len(pages)-1)
		write(fmt.Sprintf("PHOTO RESULT  %d/%d", v.page+1, len(pages)), gen.Eva.Accent)
		y += 3
		for _, line := range pages[v.page] {
			write(line, gen.Eva.Fg)
		}
		a.paintHint(c, "L/R or arrows: pages  B back")
	}
}
