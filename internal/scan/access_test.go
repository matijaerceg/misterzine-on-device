package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

func putCores(t *testing.T, card string, names ...string) {
	t.Helper()
	for _, n := range names {
		putFile(t, filepath.Join(card, "_Arcade", "cores", n), "x")
	}
}

func TestCoreAccess(t *testing.T) {
	card := t.TempDir()
	putCores(t, card, "jtcps3.rbf", "jtsimson_20260901.rbf", "blkheart_mister_20260909.rbf", "metamrph_mister_20260101.rbf",
		"jtmixed.rbf", "jtother.rbf", "foo.rbf", "foo_custom_20260101.rbf")
	arcade := func(core string, beta bool, gate string) data.Row {
		return data.Row{Base: "Arcade", Core: core, Beta: beta, Gate: gate}
	}
	catalogue := []data.Row{
		arcade("jtcps3", true, "jtbeta"),
		arcade("jtcps3", true, "jtbeta"),
		arcade("jtsimson", false, ""),
		arcade("blkheart_mister", true, "coinop-collection-beta"),
		arcade("metamrph_mister", true, "coinop-collection-alpha"),
		arcade("jtmixed", true, "jtbeta"),
		arcade("jtmixed", false, ""),
		arcade("jtgals", true, "jtbeta"), // not on the card
		arcade("jtold", true, ""),        // an older export: beta without a gate
		arcade("foo", true, "jtbeta"),
		{Base: "Console", Core: "jtother", Beta: true, Gate: "jtbeta"}, // not an arcade core
	}
	jt := Access{Known: true, Beta: true, Gate: "jtbeta"}
	ca := NewCoreAccess(catalogue, ScanCores(card))
	for _, tc := range []struct {
		rbf  string
		want Access
	}{
		{"jtcps3", jt},
		{"JTCPS3", jt},
		{"jtsimson", Access{Known: true}},
		// Coin-Op's MRAs name the core without _mister; Main finds the file
		{"blkheart", Access{Known: true, Beta: true, Gate: "coinop-collection-beta"}},
		{"metamrph", Access{Known: true, Beta: true, Gate: "coinop-collection-alpha"}},
		{"jtmixed", Access{}}, // rows that disagree
		{"jtnotlisted", Access{}},
		{"jtother", Access{}}, // only a console row names it
		{"jtgals", jt},        // not on the card: the name decides
		{"jtold", Access{Known: true, Beta: true}},
		// Main loads foo_custom_20260101.rbf, not the catalogue's foo.rbf
		{"foo", Access{}},
		{"", Access{}},
	} {
		if got := ca.Of(tc.rbf); got != tc.want {
			t.Errorf("Of(%q) = %+v, want %+v", tc.rbf, got, tc.want)
		}
	}
	if !ca.Of("jtold").JotegoBeta() || !jt.JotegoBeta() || ca.Of("blkheart").JotegoBeta() || ca.Of("jtsimson").JotegoBeta() {
		t.Error("JotegoBeta")
	}
	var none *CoreAccess
	if none.Of("jtcps3") != (Access{}) {
		t.Error("nil access knows a core")
	}
	// without a core index only the names are there to go by
	if got := NewCoreAccess(catalogue, nil).Of("foo"); got != jt {
		t.Errorf("no index: Of(foo) = %+v", got)
	}
}

// Arcade Offset's three MRAs with a jtbeta.zip section, as ac3 found them,
// and a stand-in on another author's public core: a local game is beta when
// its own core is.
func TestDiscoverLocalMarksBetaCores(t *testing.T) {
	card := t.TempDir()
	putCores(t, card, "jtcps3.rbf", "jtsimson_20260901.rbf", "NMK16_Gunnail_20260920.rbf", "metamrph_mister_20260101.rbf")
	mk := func(rel, body string) {
		t.Helper()
		putFile(t, filepath.Join(card, filepath.FromSlash(rel)), body)
	}
	mk("_Arcade/_Arcade Offset/_CP System III/SF3 4rd Arrange.mra", mra("SF3 4rd Arrange", "sfiii4n", "jtcps3", ""))
	mk("_Arcade/_Arcade Offset/_CP System III/SF3 Makoto fix.mra", mra("SF3 Makoto fix", "sfiii3nr1", "jtcps3", ""))
	mk("_Arcade/_Arcade Offset/_The Simpsons/The Simpsons (2 Players Free Play).mra", mra("The Simpsons (2 Players Free Play)", "simpsons2pjfp", "jtsimson", ""))
	mk("_Arcade/_Kuze/Black Heart.mra", mra("Black Heart", "blkheart", "NMK16_Gunnail", ""))
	mk("_Arcade/_Extra/Metamorphic Hack.mra", mra("Metamorphic Hack", "metamrphh", "metamrph", ""))
	catalogue := []data.Row{
		{K: "sfiii3n", Base: "Arcade", Core: "jtcps3", SN: "sfiii3n", FamilySets: []string{"sfiii3nr1"},
			MRA: "_Arcade/Street Fighter III 3rd Strike.mra", Beta: true, Gate: "jtbeta"},
		{K: "simpsons", Base: "Arcade", Core: "jtsimson", SN: "simpsons", MRA: "_Arcade/The Simpsons.mra"},
		{K: "blkheart", Base: "Arcade", Core: "blkheart_mister", SN: "blkheart", MRA: "_Arcade/Black Heart.mra", Beta: true, Gate: "coinop-collection-beta"},
		{K: "metamrph", Base: "Arcade", Core: "metamrph_mister", SN: "metamrph", MRA: "_Arcade/Metamorphic Force.mra", Beta: true, Gate: "coinop-collection-alpha"},
	}
	// Jotego's own SF3 MRA is not on the card, so its row does not run and
	// the Makoto fix stands in for it
	status := []data.Status{data.StatusNotFound, data.StatusNotFound, data.StatusNotFound, data.StatusNotFound}
	res := DiscoverLocal(card, "", catalogue, ScanCores(card), status, nil, nil, false)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	type got struct {
		beta    bool
		gate    string
		standin bool
	}
	rows := map[string]got{}
	for _, r := range res.Rows {
		rows[r.K] = got{r.Beta, r.Gate, r.Standin}
	}
	want := map[string]got{
		"local:sfiii4n":       {true, "jtbeta", false},
		"local:sfiii3nr1":     {true, "jtbeta", true},
		"local:simpsons2pjfp": {false, "", false},
		"local:blkheart":      {false, "", true}, // Kuze's core, standing in for Coin-Op's beta
		"local:metamrphh":     {true, "coinop-collection-alpha", false},
	}
	for k, w := range want {
		if g, ok := rows[k]; !ok || g != w {
			t.Errorf("%s: got %+v (listed %v), want %+v", k, g, ok, w)
		}
	}
	if len(rows) != len(want) {
		t.Errorf("rows %v", rows)
	}
}

// Whether a missing jtbeta.zip stops Start is the MRA's core's to say.
func TestROMCheckJotegoKey(t *testing.T) {
	prog := crcOf("program")
	card := filepath.Join(t.TempDir(), "fat")
	putCores(t, card, "jtcps3.rbf", "jtsimson_20260901.rbf", "blkheart_mister_20260909.rbf")
	putZip(t, filepath.Join(card, "games/mame/game.zip"), member{name: "p", data: "p"})
	catalogue := []data.Row{
		{Base: "Arcade", Core: "jtcps3", Beta: true, Gate: "jtbeta"},
		{Base: "Arcade", Core: "jtsimson"},
		{Base: "Arcade", Core: "blkheart_mister", Beta: true, Gate: "coinop-collection-beta"},
	}
	key := `<rom index="17" zip="jtbeta.zip" md5="None"><part name="beta.bin" crc="` + prog + `"/></rom>`
	game := `<rom index="0" zip="game.zip" md5="None"><part name="p"/></rom>`
	const missing = "Missing game ROM: jtbeta.zip"
	for _, tc := range []struct {
		name, rbf, body string
		access          bool // the catalogue has been read
		key             []member
		want            string
		block           bool
	}{
		{"beta core", "jtcps3", game + key, true, nil,
			missing + "; the jtcps3 core is a Patreon beta and needs it", true},
		{"public core", "jtsimson", game + key, true, nil,
			missing + "; the jtsimson core isn't a beta: MiSTer shows an error, but the game plays", false},
		{"core the catalogue does not list", "jtnew", game + key, true, nil,
			missing + "; can't tell whether the jtnew core needs it", false},
		{"before the catalogue is read", "jtcps3", game + key, false, nil,
			missing + "; can't tell whether the jtcps3 core needs it", false},
		{"no rbf", "", game + key, true, nil,
			missing + "; can't tell whether the MRA's core needs it", false},
		// the key is Jotego's; a Coin-Op core gated some other way says nothing about it
		{"another gate", "blkheart", game + key, true, nil,
			missing + "; can't tell whether the blkheart core needs it", false},
		{"key without beta.bin", "jtcps3", game + key, true, []member{{name: "other", data: "o"}},
			"Incomplete ROM: jtbeta.zip (no beta.bin); the jtcps3 core is a Patreon beta and needs it", true},
		{"key present", "jtcps3", game + key, true, []member{{name: "beta.bin", data: "program"}}, "", false},
		// a readable key of another CRC is the MRA's older key: Main sends it
		{"older key on a beta core", "jtcps3", game + key, true, []member{{name: "beta.bin", data: "older"}},
			"Wrong ROM version: jtbeta.zip (beta.bin) is " + crcOf("older") + ", the MRA expects " + prog, false},
		// a key that only warns leaves the rest of the MRA to be looked at
		{"a missing game ROM still stops a public core", "jtsimson",
			`<rom index="0" zip="game.zip" md5="None"><part name="p"/></rom><rom index="17" zip="jtbeta.zip" md5="None"><part name="beta.bin"/><part name="q" zip="gone.zip"/></rom>`,
			true, nil, "Missing game ROM: gone.zip", true},
		{"a part's own zip list", "jtcps3", `<rom index="17" zip="game.zip"><part name="beta.bin" zip="jtbeta.zip"/></rom>`, true, nil,
			missing + "; the jtcps3 core is a Patreon beta and needs it", true},
		// only a part looked for in jtbeta.zip alone is the key
		{"jtbeta.zip among others", "jtsimson", `<rom index="17" zip="jtbeta.zip|game.zip"><part name="beta.bin"/></rom>`, true, nil,
			missing, true},
		{"coinopkey.zip stays a requirement", "jtsimson", `<rom index="37" zip="coinopkey.zip" md5="None"><part name="coinop.key"/></rom>`, true, nil,
			"Missing game ROM: coinopkey.zip", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			os.Remove(filepath.Join(card, "games/mame/jtbeta.zip"))
			if tc.key != nil {
				putZip(t, filepath.Join(card, "games/mame/jtbeta.zip"), tc.key...)
			}
			rbf := ""
			if tc.rbf != "" {
				rbf = "<rbf>" + tc.rbf + "</rbf>"
			}
			putFile(t, filepath.Join(card, "_Arcade/game.mra"), "<misterromdescription><setname>g</setname>"+rbf+tc.body+"</misterromdescription>")
			c := NewROMCheck(card)
			if tc.access {
				c.SetAccess(NewCoreAccess(catalogue, ScanCores(card)))
			}
			got := c.Check("_Arcade/game.mra", true)
			if got.Text != tc.want || got.Block != tc.block {
				t.Fatalf("got %q block=%v\nwant %q block=%v", got.Text, got.Block, tc.want, tc.block)
			}
		})
	}
}

// The ROM check reads the core as the header does: a child of the root, in
// any case, trimmed, the last one winning as in Main.
func TestParseROMSectionsRBF(t *testing.T) {
	for _, tc := range []struct{ mra, want string }{
		{`<misterromdescription><RBF> jtcps3 </RBF><rom index="0" zip="a.zip"><part name="p"/></rom></misterromdescription>`, "jtcps3"},
		{`<misterromdescription><rom index="0" zip="a.zip"><rbf>inside</rbf></rom><rbf>jtsimson</rbf></misterromdescription>`, "jtsimson"},
		{`<misterromdescription><rbf>first</rbf><rbf>second</rbf><rbf> </rbf></misterromdescription>`, "second"},
		{`<misterromdescription><rom index="0" zip="a.zip"><part name="p"/></rom></misterromdescription>`, ""},
	} {
		p := filepath.Join(t.TempDir(), "game.mra")
		putFile(t, p, tc.mra)
		_, rbf, issue := parseROMSections(p, false)
		if rbf != tc.want || issue != "" {
			t.Errorf("%s: rbf %q issue %q, want %q", tc.mra, rbf, issue, tc.want)
		}
		if a, ok := ParseMRAHeader(p); ok && a.RBF != strings.ToLower(tc.want) {
			t.Errorf("%s: header reads %q", tc.mra, a.RBF)
		}
	}
}

// A result made before the catalogue was read is checked again once it is.
func TestROMCheckRechecksUnderNewAccess(t *testing.T) {
	card := t.TempDir()
	putFile(t, filepath.Join(card, "_Arcade/game.mra"), "<misterromdescription/>")
	c := NewROMCheck(card)
	var seen []*CoreAccess
	c.run = func(_ string, acc *CoreAccess) ROMResult {
		seen = append(seen, acc)
		if acc != nil {
			return ROMResult{Text: "under the catalogue"}
		}
		return ROMResult{}
	}
	c.Check("_Arcade/game.mra", true)
	c.Check("_Arcade/game.mra", false) // fresh enough: reused
	if len(seen) != 1 || seen[0] != nil {
		t.Fatalf("runs %v", seen)
	}
	acc := NewCoreAccess(nil, nil)
	c.SetAccess(acc)
	if got := c.Check("_Arcade/game.mra", false); got.Text != "" {
		t.Fatalf("the older answer is shown while checking: %+v", got)
	}
	<-c.Ready() // the refresh changed the answer
	if len(seen) != 2 || seen[1] != acc {
		t.Fatalf("runs %v", seen)
	}
	if got, _ := c.Known("_Arcade/game.mra"); got.Text != "under the catalogue" {
		t.Fatalf("known %+v", got)
	}
}
