package scan

import (
	"reflect"
	"strings"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

var fullHeader = `<misterromdescription>
	<name>Grobda</name>
	<region>World</region>
	<homebrew>no</homebrew>
	<bootleg>yes</bootleg>
	<year>1984</year>
	<manufacturer>Namco</manufacturer>
	<category>Shooter - Multidirectional</category>
	<setname>grobda</setname>
	<parent>grobda</parent>
	<rbf>druaga</rbf>
	<rotation>vertical (cw)</rotation>
	<players>2 (alternating)</players>
	<joystick>8-way</joystick>
	<num_buttons>2</num_buttons>
	<buttons names="Cannon Beam,Sealed,Start 1P,Start 2P,Coin" default="A,B,Start,Select,R"></buttons>
	<switches default="36,00,00"><dip name="Lives" bits="8,9" ids="3,1,2,5"/></switches>
	<rom index="0" zip="grobda.zip|namco.zip" type="merged"><part>` + strings.Repeat("00 ", 30000) + `</part></rom>
	<rom index="1"><part>ff</part></rom>
</misterromdescription>`

func TestParseMRAHeaderFull(t *testing.T) {
	a, ok := parseMRA(strings.NewReader(fullHeader))
	if !ok {
		t.Fatal("full header rejected")
	}
	want := Alt{RBF: "druaga", Setname: "grobda", Parent: "grobda", Zips: []string{"grobda.zip", "namco.zip"},
		Name: "Grobda", Year: "1984", Manufacturer: "Namco", Category: "Shooter - Multidirectional",
		Rotation: "vertical (cw)", Region: "World", Players: "2 (alternating)", Joystick: "8-way",
		NumButtons: 2, ButtonNames: []string{"Cannon Beam", "Sealed", "Start 1P", "Start 2P", "Coin"}, Bootleg: true}
	if !reflect.DeepEqual(a, want) {
		t.Fatalf("parse =\n%+v\nwant\n%+v", a, want)
	}
	// The same header cut inside the ROM data still yields everything.
	cut := fullHeader[:strings.Index(fullHeader, "00 00 00 00 00 00")+20]
	a2, ok := parseMRA(strings.NewReader(cut))
	if !ok || !reflect.DeepEqual(a2, want) {
		t.Fatalf("truncated parse = %+v %v", a2, ok)
	}
	// Elements below the root (a <name> inside a <rom>) are not header text.
	nested := `<misterromdescription><rbf>x</rbf><setname>s</setname><rom index="0"><name>ignore</name></rom><name>Real</name></misterromdescription>`
	if a3, ok := parseMRA(strings.NewReader(nested)); !ok || a3.Name != "Real" {
		t.Fatalf("nested name leaked: %+v %v", a3, ok)
	}
	// A file cut short before any <rom> is still unreadable.
	if _, ok := parseMRA(strings.NewReader(`<misterromdescription><rbf>x</rbf><setname>s`)); ok {
		t.Fatal("short header accepted")
	}
}

func TestRowFromAltRotationAndControls(t *testing.T) {
	a, _ := parseMRA(strings.NewReader(fullHeader))
	a.Path = "_Arcade/_Extra/Grobda (World).mra"
	r := rowFromAlt(a)
	two := 2
	want := data.Row{Title: "Grobda", Base: "Arcade", Src: data.SrcLocal, K: "local:grobda", SN: "grobda", Family: "grobda",
		Core: "druaga", MRA: a.Path, Year: "1984", Manufacturer: "Namco", Reg: "World", Rot: "Vertical (CW)", Plr: "2",
		Ctl: "8-way · 2 buttons", Buttons: &two, Note: "MRA category: Shooter - Multidirectional"}
	if r.Buttons == nil || *r.Buttons != 2 {
		t.Fatalf("buttons = %v", r.Buttons)
	}
	r.Buttons, want.Buttons = nil, nil
	if !reflect.DeepEqual(r, want) {
		t.Fatalf("row =\n%+v\nwant\n%+v", r, want)
	}
	if r.RotGroup() != "v" || !r.IsLocal() || !r.IsArcade() {
		t.Fatal("derived flags")
	}
	dirs, btn := r.ControlFacets()
	if dirs != "8-way" || btn != "2" {
		t.Fatalf("facets %q %q", dirs, btn)
	}

	for in, out := range map[string]string{
		"vertical (ccw)": "Vertical (CCW)", "Vertical (CW)": "Vertical (CW)", "vertical": "Vertical",
		"horizontal (180)": "Horizontal (180)", "horizontal": "Horizontal", "": "", "tate": "",
	} {
		if got := mraRotation(in); got != out {
			t.Errorf("rotation %q = %q, want %q", in, got, out)
		}
	}

	// No <name>: the filename stem is the title; names-only buttons are
	// counted without start, coin and service entries; no header data at all
	// leaves the controls unknown.
	b := Alt{Path: "_Arcade/Some Game (set 1).mra", RBF: "core", Setname: "SomeGame", ButtonNames: []string{"Fire", "Jump", "", "Start 1P", "Coin", "Service"}}
	rb := rowFromAlt(b)
	if rb.Title != "Some Game (set 1)" || rb.K != "local:somegame" || rb.SN != "somegame" || rb.Buttons == nil || *rb.Buttons != 2 || rb.Ctl != "2 buttons" {
		t.Fatalf("row = %+v buttons=%v", rb, rb.Buttons)
	}
	rc := rowFromAlt(Alt{Path: "_Arcade/x.mra", RBF: "core", Setname: "x", Joystick: "4-way"})
	if rc.Buttons != nil || rc.Ctl != "4-way" || rc.Rot != "" || rc.Note != "" {
		t.Fatalf("row = %+v", rc)
	}
	if rd := rowFromAlt(Alt{Path: "_Arcade/y.mra", RBF: "core", Setname: "y", NumButtons: 1}); rd.Ctl != "1 button" {
		t.Fatalf("singular: %q", rd.Ctl)
	}
}
