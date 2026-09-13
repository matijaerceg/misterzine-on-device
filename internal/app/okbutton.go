package app

import (
	"fmt"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

// Options -> OK button: which face button confirms, per pad.
//
// MiSTer's define buttons screen ends by asking which button is MENU OK and
// which MENU BACK for its own menu, and stores both in the pad's map file
// (support.Pad.OK and .Back, by slot name). Someone who put OK on the
// bottom button, slot B, has said which button confirms; by default the
// app follows: on that pad B gives Enter and A gives back, and the legends
// print the names the other way round (buttons.go). OK anywhere else, or
// unset, leaves A as the confirm button, as the define slots say.
//
// A or B chosen here overrides that for one pad, kept by vendor_product,
// the identity MiSTer's map file uses, so two pads of the same model share
// it as they share a map. The row is edited with the pad in hand: the pad
// that last pressed anything is the one it shows and changes, and with
// none yet, the only defined pad connected. A pad whose A or B MiSTer
// cannot read here reaches the app through MiSTer's translation, which
// already sends Enter for its OK, so the row has nothing to offer it.
type okButtons struct {
	pads  []support.Pad   // the pads the input reader has open, as last fetched
	known map[string]bool // every input source seen: true for a pad, false otherwise
	cur   string          // the pad that pressed last, by name; "" before any
}

// padID is the pad's identity in settings, as in MiSTer's map file name.
func padID(p support.Pad) string { return fmt.Sprintf("%04x_%04x", p.Vendor, p.Product) }

func (a *App) refreshPads() {
	a.oks.pads = nil
	if h := a.cfg.Support; h != nil && h.Pads != nil {
		a.oks.pads = h.Pads()
	}
	a.oks.known = map[string]bool{}
	for _, p := range a.oks.pads {
		a.oks.known[p.Name] = true
	}
}

func (a *App) padByName(name string) (support.Pad, bool) {
	for _, p := range a.oks.pads {
		if p.Name == name {
			return p, true
		}
	}
	return support.Pad{}, false
}

// padEvent notes which pad an event came from and applies that pad's OK
// button: on one whose OK is B, Enter and back trade places. Keys from
// keyboards, scripts and MiSTer's translation pass through untouched.
func (a *App) padEvent(ev platform.Event) platform.Event {
	switch ev.Source {
	case "", "script", "debug", "MiSTer virtual input":
		return ev
	}
	if _, seen := a.oks.known[ev.Source]; !seen {
		a.refreshPads() // a device seen for the first time: a pad just plugged in, or a keyboard
		if !a.oks.known[ev.Source] {
			a.oks.known[ev.Source] = false
		}
	}
	p, ok := a.padByName(ev.Source)
	if !ok {
		return ev
	}
	a.oks.cur = p.Name
	if a.swapped(p) {
		switch ev.Key {
		case platform.KeyEnter:
			ev.Key = platform.KeyBack
		case platform.KeyBack:
			ev.Key = platform.KeyEnter
		}
	}
	return ev
}

// okButton is the pad's confirm button, "A" or "B", and whether that is
// the automatic choice (from its MiSTer definition) rather than an override.
func (a *App) okButton(p support.Pad) (ok string, auto bool) {
	switch a.cfg.OKButtons[padID(p)] {
	case "a":
		return "A", false
	case "b":
		return "B", false
	}
	if p.OK == "B" {
		return "B", true
	}
	return "A", true
}

// swapped reports whether the pad's Enter and back trade places: it is read
// by slot here and its OK button is B.
func (a *App) swapped(p support.Pad) bool {
	ok, _ := a.okButton(p)
	return p.Direct && ok == "B"
}

// currentPad is the pad the OK button row and the legends follow: the one
// that pressed last, else the only defined pad connected.
func (a *App) currentPad() (support.Pad, bool) {
	if a.oks.known == nil {
		a.refreshPads()
	}
	if a.oks.cur != "" {
		if p, ok := a.padByName(a.oks.cur); ok {
			return p, true
		}
	}
	// one pad can be several event nodes (the DE10's Xbox 360 pad is two
	// with one name), so count identities, not nodes
	var only support.Pad
	ids := map[string]bool{}
	for _, p := range a.oks.pads {
		if p.Direct {
			only = p
			ids[padID(p)] = true
		}
	}
	return only, len(ids) == 1
}

// legendSwapped reports whether the legends name A and B the other way
// round, following the current pad.
func (a *App) legendSwapped() bool {
	p, ok := a.currentPad()
	return ok && a.swapped(p)
}

// OKButtons is every per-pad override, for saving; nil when there is none.
func (a *App) OKButtons() map[string]string {
	if len(a.cfg.OKButtons) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range a.cfg.OKButtons {
		out[k] = v
	}
	return out
}

// setOKButton is the Options row's choice for the current pad: "" for
// auto, "a" or "b".
func (a *App) setOKButton(v string) {
	p, ok := a.currentPad()
	if !ok {
		return
	}
	if v == "" {
		delete(a.cfg.OKButtons, padID(p))
		return
	}
	if a.cfg.OKButtons == nil {
		a.cfg.OKButtons = map[string]string{}
	}
	a.cfg.OKButtons[padID(p)] = v
}

// okNote says where the pad's OK button comes from, for the tester and the
// Options hint: MiSTer's definition, or an override, and when MiSTer's
// MENU OK could not be followed, why.
func (a *App) okNote(p support.Pad) string {
	if _, auto := a.okButton(p); !auto {
		return "set in Options"
	}
	switch p.OK {
	case "A", "B":
		return "MiSTer's MENU OK"
	case "":
		return "MiSTer's MENU OK is not defined"
	}
	return "MiSTer's MENU OK is on " + a.label(p.OK)
}

// sameAB reports the define mistake that leaves a pad with no back button:
// one button in both the A and the B slot.
func sameAB(p support.Pad) bool {
	ca, oka := p.Slots["A"]
	cb, okb := p.Slots["B"]
	return oka && okb && ca == cb
}

// okLine is the pad tester's line about the pad's OK button.
func (a *App) okLine(p support.Pad) string {
	if sameAB(p) {
		return "A and B are the same button in MiSTer: no back button here"
	}
	ok, _ := a.okButton(p)
	return "OK button: " + a.label(ok) + " (" + a.okNote(p) + ")"
}

// okButtonRow is the Options row. It is muted with the reason when no pad
// has pressed anything yet, when the current pad comes through MiSTer's
// translation, or when its A and B are one button.
func (a *App) okButtonRow() panelEntry {
	row := panelEntry{text: "OK button", kind: "ok-button"}
	p, ok := a.currentPad()
	if !ok {
		row.vals, row.disabled = []string{"no pad used yet"}, true
		row.help = "Which pad button confirms, set per pad with the pad in hand. Auto follows the MENU OK in its MiSTer definition. Press a pad button first."
		return row
	}
	name := shortName(p.Name, 14)
	if !p.Direct {
		row.vals, row.disabled = []string{"via MiSTer"}, true
		row.help = name + ": A or B is not in its MiSTer definition, so its buttons come through MiSTer's translation. Define it in the MiSTer menu."
		return row
	}
	if sameAB(p) {
		row.vals, row.disabled = []string{"A = B"}, true
		row.help = name + ": A and B are the same button in its MiSTer definition, so it has no back button here. Define it again in the MiSTer menu."
		return row
	}
	auto := "A"
	if p.OK == "B" {
		auto = "B"
	}
	row.vals = []string{"Auto from MiSTer (" + a.label(auto) + ")", a.label("A"), a.label("B")}
	switch a.cfg.OKButtons[padID(p)] {
	case "a":
		row.idx = 1
	case "b":
		row.idx = 2
	}
	why := "Auto follows the MENU OK in its MiSTer definition."
	switch {
	case p.OK == "":
		why = "Its MiSTer definition sets no MENU OK, so Auto is " + a.label("A") + "."
	case p.OK != "A" && p.OK != "B":
		why = "Its MiSTer MENU OK is on " + a.label(p.OK) + ", so Auto is " + a.label("A") + "."
	}
	row.help = name + ": " + why + " " + a.label("A") + " or " + a.label("B") + " overrides, for this pad only. Change it with the pad in hand."
	return row
}
