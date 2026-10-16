package app

import "sort"

func pairedControlAction(id string) string {
	switch id {
	case "enter":
		return "back"
	case "back":
		return "enter"
	case "left":
		return "right"
	case "right":
		return "left"
	case "up":
		return "down"
	case "down":
		return "up"
	}
	return ""
}

// Only inputs physically proven during this visit can replace a basic control.
func (a *App) controlSwapCandidates() []uint16 {
	v := &a.mapping
	d := &v.check
	paired := pairedControlAction(v.draft.Inputs[d.editing])
	if paired == "" || !d.proven[d.editing] {
		return nil
	}
	codes := []uint16{}
	for code, id := range v.draft.Inputs {
		if id == paired && d.proven[code] {
			codes = append(codes, code)
		}
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	return codes
}

func (a *App) startControlSwap() {
	codes := a.controlSwapCandidates()
	if len(codes) == 0 {
		id := pairedControlAction(a.mapping.draft.Inputs[a.mapping.check.editing])
		a.mapping.message = "Test a " + actionName(id) + " button in the table first."
		return
	}
	if len(codes) == 1 {
		a.mapping.check.swap = codes[0]
		a.setControlMode("swap-confirm")
	} else {
		a.setControlMode("swap-target")
	}
}

func (a *App) commitControlSwap() {
	v := &a.mapping
	d := &v.check
	for _, code := range a.controlSwapCandidates() {
		if code == d.swap {
			candidate := v.draft.Clone()
			candidate.Inputs[d.editing], candidate.Inputs[code] = candidate.Inputs[code], candidate.Inputs[d.editing]
			a.commitControl(candidate)
			return
		}
	}
	v.message = "Test both buttons in the table first."
}
