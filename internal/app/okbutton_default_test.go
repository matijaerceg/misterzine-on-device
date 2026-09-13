package app

import (
	"strings"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

// A pad MiSTer has not defined, read by the Linux default layout, confirms
// with its bottom button: the row reads Auto from Linux (B), the legends
// swap, the tester says where the layout came from, and an override still
// applies per pad.
func TestOKButtonLinuxDefaultLayout(t *testing.T) {
	pad := support.Pad{Node: "event12", Name: "8BitDo Ultimate 2C Wireless Controller", Vendor: 0x2dc8, Product: 0x310a, Direct: true, OK: "B", Back: "A", Map: "Linux default layout", Menu: "316", MenuStick: "axes 0/1",
		Slots: map[string]uint16{"Up": 802, "Down": 803, "Left": 800, "Right": 801, "A": 305, "B": 304, "X": 307, "Y": 308, "L": 310, "R": 311, "Select": 314, "Start": 315}}
	a := okTestApp(func() []support.Pad { return []support.Pad{pad} })
	row := okRow(a)
	if row.disabled || row.vals[row.idx] != "Auto from Linux (B)" || !strings.Contains(row.help, "not defined in MiSTer") {
		t.Fatalf("row: %+v", row)
	}
	checkOKHint(t, a, row)
	if !a.legendSwapped() || a.padEvent(platform.Event{Key: platform.KeyEnter, Code: 304, Pressed: true, Source: pad.Name}).Key != platform.KeyBack {
		t.Fatal("the bottom button should confirm")
	}
	lines := a.padLines(pad)
	if lines[2] != "  no MiSTer map: read by the Linux default layout; define it in MiSTer to change it" || lines[3] != "  OK button: B (Linux default layout)" {
		t.Fatalf("tester lines %q", lines)
	}
	a.cfg.OKButtons = map[string]string{"2dc8_310a": "a"}
	if a.legendSwapped() || okRow(a).idx != 1 || a.okNote(pad) != "set in Options" {
		t.Fatal("the override should put A back")
	}
}
