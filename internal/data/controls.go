package data

import (
	"regexp"
	"strconv"
	"strings"
)

var buttonCount = regexp.MustCompile(`(?i)^(\d+) buttons?$`)

// ControlFacets splits the feed's combined description. Older feeds omitted
// zero counts; only an explicit count can distinguish zero from unknown.
func ControlFacets(ctl string) (directions, buttons string) {
	for _, part := range strings.Split(ctl, "·") {
		part = strings.ToLower(strings.TrimSpace(part))
		if m := buttonCount.FindStringSubmatch(part); m != nil {
			n, err := strconv.Atoi(m[1])
			if err == nil && n >= 0 {
				buttons = strconv.Itoa(n)
			}
			continue
		}
		if part == "" {
			continue
		}
		if _, err := strconv.Atoi(part); err == nil {
			continue // a bare number is not a documented direction or button count
		}
		if strings.HasPrefix(part, "double ") {
			part = strings.TrimPrefix(part, "double ") + " double"
		}
		part = strings.ReplaceAll(part, " - ", " + ")
		part = strings.ReplaceAll(part, ",", " + ")
		if part == "buttons" {
			part = "buttons only"
		}
		directions = part
	}
	return
}

// ControlFacets includes special controls that the website stores separately.
// A button count alone does not prove a game uses buttons only.
func (r *Row) ControlFacets() (controls, buttons string) {
	controls, buttons = ControlFacets(r.Ctl)
	if r.Buttons != nil && *r.Buttons >= 0 {
		buttons = strconv.Itoa(*r.Buttons)
	}
	special, _ := ControlFacets(r.Spc)
	if special != "" {
		if controls == "" {
			controls = special
		} else if special != controls {
			controls += " + " + special
		}
	}
	return
}
