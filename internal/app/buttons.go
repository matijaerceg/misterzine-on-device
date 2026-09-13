package app

import (
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// Button labels. Main names the pad's face buttons A, B, X and Y in its
// define-buttons screen (the SNES layout: A right, B bottom, X top, Y left)
// and turns them into Enter, Esc, Tab and Space while the app runs, so the
// app only ever sees those four names. Options -> Button labels chooses how
// the legends print them; nothing about what the buttons do changes.
//
// Every label set is one cell per button, so hints keep their widths.
var (
	buttonLabelSets = []string{"mister", "xbox", "playstation", "numbers"}
	buttonLabels    = map[string]map[string]string{
		"mister": {"A": "A", "B": "B", "X": "X", "Y": "Y"},
		// an Xbox-lettered pad mapped by position: MiSTer's A (right) is
		// the pad's B, and so on
		"xbox": {"A": "B", "B": "A", "X": "Y", "Y": "X"},
		// PlayStation symbols by position: circle right, cross bottom,
		// triangle top, square left
		"playstation": {"A": gfx.Circle, "B": gfx.Cross, "X": gfx.Triangle, "Y": gfx.Square},
		// MiSTer's define order
		"numbers": {"A": "1", "B": "2", "X": "3", "Y": "4"},
	}
)

// buttonLabelValue is the Options value text for a set: its four labels in
// MiSTer's A B X Y order.
func buttonLabelValue(set string) string {
	m := buttonLabels[set]
	return m["A"] + " " + m["B"] + " " + m["X"] + " " + m["Y"]
}

// buttonLabelValues lists the Options values in buttonLabelSets order.
func buttonLabelValues() []string {
	vals := make([]string, len(buttonLabelSets))
	for i, set := range buttonLabelSets {
		vals[i] = buttonLabelValue(set)
	}
	return vals
}

// ButtonLabels is the label set in use, one of buttonLabelSets.
func (a *App) ButtonLabels() string {
	if _, ok := buttonLabels[a.cfg.ButtonLabels]; ok {
		return a.cfg.ButtonLabels
	}
	return buttonLabelSets[0]
}

// btn prints a button name for a legend: the button that does what the
// name stands for, in the current label set. On a pad whose OK button is B
// (okbutton.go) the names A and B trade places first, so "A" names the
// button that confirms and "Hold B" the one that goes back. Only the bare
// names A, B, X and Y change; "Start", "L/R", arrows and any other word
// pass through, so it is safe on a hint chunk like "Hold B" or "A < >" as
// well as on a single name.
func (a *App) btn(s string) string {
	if a.legendSwapped() {
		s = applyLabels(s, swapLabels)
	}
	return a.label(s)
}

// label prints a MiSTer slot name as the current label set shows it, with
// no regard to what the button does: the pad tester names slots with it.
func (a *App) label(s string) string {
	return applyLabels(s, buttonLabels[a.ButtonLabels()])
}

// swapLabels trades A and B.
var swapLabels = map[string]string{"A": "B", "B": "A"}

func applyLabels(s string, m map[string]string) string {
	if l, ok := m[s]; ok {
		return l
	}
	if !strings.ContainsAny(s, "ABXY") {
		return s
	}
	words := strings.Split(s, " ")
	for i, w := range words {
		if l, ok := m[w]; ok {
			words[i] = l
		}
	}
	return strings.Join(words, " ")
}
