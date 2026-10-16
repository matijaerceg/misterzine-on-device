package app

import (
	"fmt"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"image"
	"time"
)

func (a *App) controlVisibleRows() int {
	extra := 0
	if a.sm.Cols(a.lay.Body.Dx()-8) < 36 {
		extra = 1
	}
	return max(1, (a.lay.Body.Dy()-(5+extra)*(a.sm.H+3)-14)/(a.sm.H+3))
}
func (a *App) physicalControlName(code uint16) string {
	if code == 0 {
		return "--"
	}
	p := a.mapping.device
	if p.Keyboard {
		return a.controlCode(code)
	}
	if slot := p.Slot(code); slot != "" {
		return slot
	}
	return "--"
}
func (a *App) currentControlName(code uint16) string {
	if code == 0 {
		return "--"
	}
	if name := a.mapping.draft.Names[code]; name != "" {
		return displayButtonName(name)
	}
	if name := controlDirectionName(a.mapping.draft.Inputs[code]); name != "" {
		return name
	}
	if name := a.physicalControlName(code); name != "--" {
		return name
	}
	return a.controlCode(code)
}
func (a *App) paintControls(c *gfx.Canvas) {
	v := &a.mapping
	d := &v.check
	f := a.sm
	l := a.lay
	line := f.H + 3
	box := l.Body.Inset(4)
	cols := f.Cols(box.Dx())
	y := box.Min.Y
	c.Fill(l.Status, pal.Surface)
	title := "Controller check"
	if v.roleOptions && (v.mode == "devices" || v.mode == "role") {
		title = "Keyboard use"
	}
	c.Text(l.Status.Min.X+2, l.Status.Min.Y+2, f, title, pal.Accent)
	c.Fill(l.Body, pal.Bg)
	c.Fill(l.Hint, pal.Surface)
	write := func(s string, col rgb) { c.Text(box.Min.X, y, f, gfx.Fit(s, cols), col); y += line }
	if v.mode == "devices" {
		title := "Choose a controller"
		if v.roleOptions {
			title = "Choose a keyboard"
		}
		write(title, pal.Fg)
		if len(v.devices) == 0 {
			write("No readable controllers found.", pal.Muted)
		}
		for i, p := range v.devices {
			col := pal.Muted
			prefix := "  "
			if i == v.cursor {
				col = pal.Accent
				prefix = "> "
			}
			write(prefix+data.ASCII(p.Name), col)
		}
		a.hintLine(c, l.Hint.Min.X+2, l.Hint.Min.Y+2, l.Hint.Dx()-4, "A Choose  B Back")
		return
	}
	if v.mode == "initial-role" {
		write("New keyboard detected", pal.Accent)
		write(data.ASCII(v.device.Name), pal.Fg)
		y += line / 2
		for i, label := range []string{"Keyboard", "Controller / pad"} {
			col := pal.Fg
			prefix := "  "
			if i == v.cursor {
				col = pal.Accent
				prefix = "> "
			}
			write(prefix+label, col)
		}
		y += line
		write("Tap any button to switch", pal.Muted)
		one, two := a.controlInvitationProgress()
		y = a.holdText(c, box.Min.X, y, cols, line, "Hold one button to choose", one) + line/2
		a.holdText(c, box.Min.X, y, cols, line, "Not now: hold two buttons or Esc", two)
		if v.message != "" {
			c.Text(box.Min.X, box.Max.Y-line, f, gfx.Fit(v.message, cols), pal.Accent)
		}
		a.hintLine(c, l.Hint.Min.X+2, l.Hint.Min.Y+2, l.Hint.Dx()-4, "Arrows / Enter / Esc")
		return
	}
	if v.mode == "invitation" {
		write("New controller detected", pal.Accent)
		for _, name := range gfx.Wrap(data.ASCII(v.device.Name), cols, 2) {
			write(name, pal.Fg)
		}
		y += line
		one, two := a.controlInvitationProgress()
		y = a.holdText(c, box.Min.X, y, cols, line, "Begin setup: hold one button", one) + line
		notNow := "Not now: hold two buttons on any device"
		if a.controlKeyboardAvailable(false) {
			notNow += " or press Esc on keyboard"
		}
		a.holdText(c, box.Min.X, y, cols, line, notNow, two)
		return
	}
	heading := "Currently mapping:"
	if d.automatic && v.mode != "table" {
		heading = "New controller detected"
	}
	write(heading, pal.Muted)
	write(data.ASCII(v.device.Name), pal.Fg)
	if v.mode == "table" {
		widths := []int{5, 8, max(4, cols-21), 8}
		if cols >= 42 {
			widths = []int{5, 11, 12, cols - 28}
		}
		heads := []string{"Code", "MiSTer/key", "MZ label", "Action"}
		if cols < 36 {
			heads = []string{"Code", "MiSTer", "MZ", "Action"}
		}
		x := box.Min.X
		for i, h := range heads {
			c.Text(x, y, f, gfx.Fit(h, widths[i]-1), pal.Muted)
			x += widths[i] * f.W
		}
		y += line
		if cols < 36 {
			c.Text(box.Min.X+widths[0]*f.W, y, f, "/ key", pal.Muted)
			c.Text(box.Min.X+(widths[0]+widths[1])*f.W, y, f, "label", pal.Muted)
			y += line
		}
		viewport := image.Rect(box.Min.X, y, box.Max.X, y+a.controlVisibleRows()*line)
		rows := a.controlTableRows()
		for i, r := range rows {
			yy := y + i*line - int(d.scroll*float64(line))
			if yy+line <= viewport.Min.Y || yy >= viewport.Max.Y {
				continue
			}
			col := pal.Muted
			if r.code != 0 && d.tested[r.code] {
				col = pal.Fg
			}
			if r.code != 0 && d.held[r.code] {
				col = pal.Accent
				c.Fill(image.Rect(box.Min.X-1, yy-1, box.Max.X, yy+line-1).Intersect(viewport), pal.Surface)
			}
			code := "--"
			if r.code != 0 {
				code = fmt.Sprint(r.code)
			}
			name := actionName(r.action)
			if cols < 42 {
				switch r.action {
				case "pageup":
					name = "PgUp"
				case "pagedown":
					name = "PgDn"
				}
			}
			// The MZ label cell stays muted when its name cannot reach a
			// legend, even while the row is held; the other cells and the
			// row background still identify the press.
			label := a.currentControlName(r.code)
			switch {
			case r.action == "menu":
				label = "N/A"
			case r.action == "" && r.code != 0:
				label = "--"
			}
			values := []string{code, a.physicalControlName(r.code), label, name}
			if r.action == "menu" && r.code == 0 && v.draft.LegacyMenuChord {
				values[1] = "MiSTer combo"
			}
			x = box.Min.X
			for j, s := range values {
				cell := col
				if j == 2 && !controlLabelEditable(r.action) {
					cell = pal.Muted
				}
				c.TextClip(x, yy, f, gfx.Fit(s, widths[j]-1), cell, viewport)
				x += widths[j] * f.W
			}
		}
		if len(rows) > a.controlVisibleRows() {
			track := image.Rect(box.Max.X-2, viewport.Min.Y, box.Max.X, viewport.Max.Y)
			c.Fill(track, pal.Line)
			h := max(3, track.Dy()*a.controlVisibleRows()/len(rows))
			offset := int(d.scroll * float64(track.Dy()-h) / float64(max(1, len(rows)-a.controlVisibleRows())))
			c.Fill(image.Rect(track.Min.X, track.Min.Y+offset, track.Max.X, track.Min.Y+offset+h), pal.Muted)
		}
		one, two := time.Time{}, time.Time{}
		if !d.cancelled {
			if d.gesture == 1 {
				one = d.at
			}
			if d.gesture == 2 {
				two = d.at
			}
		}
		y = box.Max.Y - 2*line
		a.holdText(c, box.Min.X, y, cols, line, "Hold any button > Edit it", one)
		a.holdText(c, box.Min.X, y+line, cols, line, "Hold any two buttons > Exit", two)
	} else if v.mode == "basic" || v.mode == "retry-basic" || v.mode == "unsupported" {
		y += line / 2
		switch v.mode {
		case "basic":
			write(fmt.Sprintf("Press %s (%d / 6)", actionName(basicOrder[d.step]), d.step+1), pal.Accent)
			suggestion := []string{"D-pad or stick right", "D-pad or stick left", "D-pad or stick down", "D-pad or stick up", "Typically A / Cross", "Typically B / Circle"}[d.step]
			if v.device.Keyboard {
				suggestion = []string{"Typically Right arrow", "Typically Left arrow", "Typically Down arrow", "Typically Up arrow", "Typically Enter", "Typically Esc"}[d.step]
			}
			write(suggestion, pal.Muted)
			for i := max(0, d.step-max(0, (box.Max.Y-4*line-y)/line)); i < d.step; i++ {
				for code, action := range d.basic {
					if action == basicOrder[i] {
						write(fmt.Sprintf("%d: %s", code, actionName(action)), pal.Fg)
					}
				}
			}
		case "retry-basic":
			write("Basic controls ready", pal.Fg)
		case "unsupported":
			for _, s := range gfx.Wrap("Raw inputs unavailable. Use MiSTer's controller setup.", cols, 3) {
				write(s, pal.Fg)
			}
		}
		y = box.Max.Y - 3*line
		if v.message != "" {
			write(v.message, pal.Accent)
		} else {
			y += line
		}
		at := d.at
		if d.cancelled {
			at = time.Time{}
		}
		if v.mode == "retry-basic" {
			a.holdText(c, box.Min.X, y, cols, line, "Hold one input to retry save", at)
			a.holdText(c, box.Min.X, y+line, cols, line, "Hold on another pad > Cancel", d.dismissAt)
		} else {
			if !d.dismissAt.IsZero() {
				at = d.dismissAt
			}
			a.holdText(c, box.Min.X, y, cols, line, "Hold button on any pad > Cancel", at)
		}

	} else {
		a.paintControlPopup(c)
	}
	latest := a.controlLatestText()
	c.Text(l.Hint.Min.X+2, l.Hint.Min.Y+2, f, gfx.Fit(latest, f.Cols(l.Hint.Dx()-4)), pal.Muted)
	if v.mode != "table" && v.mode != "basic" && v.mode != "retry-basic" && v.mode != "unsupported" {
		c.Fill(l.Hint, pal.Surface)
		a.hintLine(c, l.Hint.Min.X+2, l.Hint.Min.Y+2, l.Hint.Dx()-4, "A Choose  B Cancel")

	}
}
func (a *App) paintControlPopup(c *gfx.Canvas) {
	v := &a.mapping
	d := &v.check
	f := a.sm
	line := f.H + 3
	outer := a.lay.Body.Inset(5)
	c.Fill(outer, pal.Bg)
	c.Box(outer, pal.Accent)
	box := outer.Inset(4)
	cols := f.Cols(box.Dx())
	y := box.Min.Y
	title := fmt.Sprintf("Input %d / %s", d.editing, a.currentControlName(d.editing))
	if v.mode == "role" {
		title = "Use this device as:"
	}
	c.Text(box.Min.X, y, f, gfx.Fit(title, cols), pal.Fg)
	y += line + 2
	if v.mode == "role" {
		for _, name := range gfx.Wrap(data.ASCII(v.device.Name), cols, 2) {
			c.Text(box.Min.X, y, f, name, pal.Muted)
			y += line
		}
		y += 2
	}
	if v.mode == "swap-confirm" {
		old := v.draft.Inputs[d.editing]
		paired := pairedControlAction(old)
		for _, change := range []struct {
			code     uint16
			from, to string
		}{{d.editing, old, paired}, {d.swap, paired, old}} {
			text := fmt.Sprintf("%d (%s): %s > %s", change.code, a.physicalControlName(change.code), actionName(change.from), actionName(change.to))
			for _, text := range gfx.Wrap(text, cols, 2) {
				c.Text(box.Min.X, y, f, text, pal.Fg)
				y += line
			}
		}
		y += line / 2
	}
	if v.mode == "labels" {
		color := pal.Fg
		if d.pick == -2 {
			color = pal.Accent
		}
		c.Text(box.Min.X, y, f, "Use default name", color)
		y += line
		color = pal.Fg
		if d.pick == -1 {
			color = pal.Accent
		}
		group := nameGroups[d.group]
		c.Text(box.Min.X, y, f, gfx.Fit(gfx.ArrowLeft+" "+group.title+" "+gfx.ArrowRight, cols), color)
		y += line + 2
		cell := box.Dx() / 4
		visible := max(1, (box.Max.Y-y-2*line)/line)
		first := max(0, d.pick/4-visible+1) * 4
		for i := first; i < len(group.labels) && i < first+visible*4; i++ {
			x := box.Min.X + (i%4)*cell
			yy := y + (i-first)/4*line
			col := pal.Fg
			if i == d.pick {
				col = pal.Accent
				c.Fill(image.Rect(x-1, yy-1, x+cell-1, yy+line-1), pal.Surface)
			}
			c.Text(x, yy, f, gfx.Fit(displayButtonName(group.labels[i]), f.Cols(cell)-1), col)
		}
	} else {
		rows := []string{}
		switch v.mode {
		case "button":
			rows = a.controlButtonRows()
		case "actions":
			rows = append(append([]string{}, checkNames...), "--")
		case "swap-target":
			for _, code := range a.controlSwapCandidates() {
				rows = append(rows, fmt.Sprintf("Swap with %d (%s)", code, a.physicalControlName(code)))
			}
		case "swap-confirm":
			rows = []string{"Swap buttons", "Cancel"}
		case "role":
			rows = []string{"Keyboard", "Controller / pad"}
		}
		visible := max(1, (box.Max.Y-y-3*line)/line)
		top := centeredTop(v.cursor, len(rows), visible)
		for i := top; i < len(rows) && i < top+visible; i++ {
			col := pal.Fg
			prefix := "  "
			if i == v.cursor {
				col = pal.Accent
				prefix = "> "
				c.Fill(image.Rect(box.Min.X, y-1, box.Max.X, y+line-1), pal.Surface)
			}
			c.Text(box.Min.X, y, f, gfx.Fit(prefix+rows[i], cols), col)
			y += line
		}
		if v.mode == "role" {
			help := "Other keys can type into search."
			if v.cursor == 1 {
				help = "Other keys never type into search."
			}
			y = box.Max.Y - 3*line
			for _, s := range gfx.Wrap(help, cols, 2) {
				c.Text(box.Min.X, y, f, s, pal.Muted)
				y += line
			}
		}
	}
	if v.message != "" {
		y = box.Max.Y - 2*line
		for _, s := range gfx.Wrap(v.message, cols, 2) {
			c.Text(box.Min.X, y, f, s, pal.Accent)
			y += line
		}
	}
}
