package app

// Options -> Credits: who made MisterZine, whose work it builds on, and
// the early adopters who tested it and sent feedback and encouragement.
// The page is a panel like Views: Up/Down move through the rows, B goes
// back to Options on the Credits row, and nothing on it can be changed.

// earlyAdopters lists the people thanked under "Special thanks to the
// early adopters", A to Z regardless of case. Add a name in its place to
// add a row.
var earlyAdopters = []string{
	"akeley",
	"CobolGuy007",
	"ctrain_1985",
	"dh3lix-pooch",
	"Fender",
	"gamecat666",
	"hammercompation",
	"LamerDeluxe",
	"Malento",
	"Porieux",
	"Virtualplayer",
}

// builtOn credits the people whose technology MisterZine runs on, one
// selectable row each, kept short enough for the tate layout; a following
// line (a plain row, not selectable) carries what did not fit.
var builtOn = [][]string{
	{"MiSTer FPGA: Alexey Melnikov (Sorgelig)", "and the MiSTer core developers"},
	{"Downloader and Update All: theypsilon", "Jose Manuel Barroso Galindo"},
	{"Spleen fonts: Frederic Cambus"},
	{"scientifica font: Akshay Oppiliappan"},
	{"Figtree lettering: Erik Kennedy"},
}

// creditsEntries builds the Credits page.
func (a *App) creditsEntries() []panelEntry {
	group := func(title string) panelEntry { return panelEntry{text: title, header: true, info: true} }
	spacer := panelEntry{header: true, info: true}
	row := func(text string) panelEntry { return panelEntry{text: text, kind: "credit"} }
	E := []panelEntry{group("Developer:"), row("Matija Erceg"), spacer, group("Built on:")}
	for _, lines := range builtOn {
		E = append(E, row(lines[0]))
		for _, more := range lines[1:] {
			E = append(E, panelEntry{text: "  " + more, info: true})
		}
	}
	E = append(E, spacer, group("Special thanks to the early adopters:"))
	for _, name := range earlyAdopters {
		E = append(E, row(name))
	}
	return E
}

// openCredits opens the Credits page from Options.
func (a *App) openCredits() {
	a.screen = ScreenCredits
	a.panel.entries = nil
	a.panel.cursor = 0
	a.panel.top = 0
	a.buildPanel()
	a.all = true
}

// closeCredits returns to Options on the Credits row.
func (a *App) closeCredits() {
	a.openOptions()
	for i, e := range a.panel.entries {
		if e.kind == "credits" {
			a.panel.cursor = i
			break
		}
	}
	a.all = true
}
