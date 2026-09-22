package report

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func sample() Input {
	return Input{
		Version: "v1.1.2-dev (abc1234, 2026-09-22)",
		Created: time.Date(2026, 9, 22, 20, 0, 0, 0, time.UTC),
		System:  []string{"Kernel: 5.15.1-MiSTer", "Board: Terasic DE10-Nano"},
		App: AppPart{View: "build date", Rotation: "none", Rows: 1286, Catalogue: 1276, Shown: 1071,
			Effective: `{"match_rotation":"h"}`, Hidden: []string{"coinop"},
			Local: []LocalGame{
				{K: "local:gigandes", Title: "Gigandes (World bazset)", Core: "Gigandes_baz", File: "_Arcade/Gigandes (World bazset).mra", StandsFor: "gigandes"},
				{K: "local:volfied", Title: "Volfied (World, rev 1)", Core: "Volfied", File: "_Arcade/Volfied (World, rev 1) .mra", Hidden: "Filter by rotation (the screen shows horizontal games; this one is vertical)"},
			}},
		Scan:     []string{"cores: 212", "MRAs read outside the catalogue's folders: 1061"},
		Layout:   []string{"Card top level: _Arcade/ (1172), games/ (12)", "_Arcade/: 1166 MRAs, 5 folders"},
		Files:    []File{{"_Arcade/_Organized", "an organiser's folder"}, {"_Arcade/Odd.mra", "no setname"}},
		Settings: `{"filter_ini_rotation":true}`,
		Log:      []string{"12:00:00 scan: 10 local rows", "12:00:01 fetch https://misterzine.fyi/data.json?t=123 ok"},
	}
}

func TestBuildSections(t *testing.T) {
	s := string(Build(sample()))
	if !strings.HasPrefix(s, Magic+"\nApp: v1.1.2-dev (abc1234, 2026-09-22)\nCreated: 2026-09-22T20:00:00Z\n") {
		t.Fatalf("header:\n%s", s[:120])
	}
	at := -1
	for _, h := range []string{"== SYSTEM", "== LIST", "== LOCAL GAMES (2)", "== GAME FILES NOT IN THE LIST (2)", "== CARD SCAN", "== CARD LAYOUT", "== SETTINGS", "== LOG (last 2 lines)"} {
		i := strings.Index(s, h)
		if i <= at {
			t.Fatalf("section %q missing or out of order:\n%s", h, s)
		}
		at = i
	}
	for _, want := range []string{
		"Rows: 1286 (catalogue 1276, local 2); in the list now: 1071",
		"Sources hidden by Installed only: coinop",
		"local:gigandes | Gigandes (World bazset) | core Gigandes_baz | _Arcade/Gigandes (World bazset).mra | stands in for gigandes",
		"| HIDDEN: Filter by rotation",
		"_Arcade/_Organized | an organiser's folder",
		"https://misterzine.fyi/data.json?... ok",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(s, "t=123") {
		t.Error("a query string reached the report")
	}
}

func TestBuildCaps(t *testing.T) {
	in := sample()
	in.Files = nil
	for i := 0; i < MaxFiles+40; i++ {
		in.Files = append(in.Files, File{fmt.Sprintf("_Arcade/f%04d.mra", i), "no setname"})
	}
	in.Log = nil
	for i := 0; i < MaxLogLines+100; i++ {
		in.Log = append(in.Log, fmt.Sprintf("line %04d", i))
	}
	s := string(Build(in))
	if !strings.Contains(s, "== GAME FILES NOT IN THE LIST (540)") || !strings.Contains(s, "... and 40 more") {
		t.Fatal("the file list is not capped with a count")
	}
	if strings.Contains(s, "line 0099") || !strings.Contains(s, "line 0100") || !strings.Contains(s, "line 0399") {
		t.Fatal("the log must keep its newest lines")
	}
	// A report past MaxBytes loses log first, then files, and still fits.
	in.Log = nil
	for i := 0; i < 20000; i++ {
		in.Log = append(in.Log, strings.Repeat("x", 100))
	}
	in.Files = nil
	for i := 0; i < 5000; i++ {
		in.Files = append(in.Files, File{strings.Repeat("p", 200), "no setname"})
	}
	if b := Build(in); len(b) > MaxBytes || !strings.HasPrefix(string(b), Magic) {
		t.Fatalf("report of %d bytes", len(b))
	}
}

func TestScrub(t *testing.T) {
	for in, want := range map[string]string{
		"GET https://api.example.com/x?token=abc&y=1 failed": "GET https://api.example.com/x?... failed",
		"db_url = https://raw.example.com/db.json.zip#frag":  "db_url = https://raw.example.com/db.json.zip?...",
		"licence key=ABCD-1234 accepted":                     "licence key=... accepted",
		`Authorization: "Bearer xyz"`:                        `Authorization: ...`,
		"password:hunter2, next":                             "password:..., next",
		"scan: 10 local rows (650ms)":                        "scan: 10 local rows (650ms)",
	} {
		if got := Scrub(in); got != want {
			t.Errorf("Scrub(%q) = %q, want %q", in, got, want)
		}
	}
}
