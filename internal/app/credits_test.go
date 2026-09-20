package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// Credits sits above Quit at the end of Options; A opens a page naming the developer,
// the projects MisterZine builds on and every early adopter, with only the
// credited rows under the cursor, and B returns to Options on Credits.
func TestCreditsPage(t *testing.T) {
	rows := []data.Row{{Base: "Arcade", K: "a", Title: "Alpha"}, {Base: "Arcade", K: "b", Title: "Beta"}}
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(rows, "", time.Now()), nil)
	a.actList(platform.KeyBack)
	a.actPanel(platform.KeyEnd)
	a.actPanel(platform.KeyUp)
	if e := a.panel.entries[a.panel.cursor]; e.kind != "credits" || a.panel.entries[a.panel.cursor+1].kind != "quit" {
		t.Fatalf("End, Up lands on %q at %d of %d", e.text, a.panel.cursor, len(a.panel.entries))
	}
	a.actPanel(platform.KeyEnter)
	if a.screen != ScreenCredits {
		t.Fatalf("A opened %v", a.screen)
	}
	var texts []string
	for _, e := range a.panel.entries {
		texts = append(texts, e.text)
	}
	page := strings.Join(texts, "\n")
	want := append([]string{"Matija Erceg", "Sorgelig", "theypsilon", "Frederic Cambus", "Akshay Oppiliappan", "Erik Kennedy"}, earlyAdopters...)
	for _, w := range want {
		if !strings.Contains(page, w) {
			t.Errorf("credits page lacks %q", w)
		}
	}
	if e := a.panel.entries[a.panel.cursor]; e.text != "Matija Erceg" {
		t.Fatalf("opened on %q", e.text)
	}
	// Up/Down stop only on credited rows; the plain continuation lines and
	// the headings are passed over, and nothing on the page acts on A
	seen := 0
	for i := 0; i < len(a.panel.entries); i++ {
		e := a.panel.entries[a.panel.cursor]
		if e.kind != "credit" {
			t.Fatalf("cursor on %q (kind %q)", e.text, e.kind)
		}
		seen++
		before := a.panel.cursor
		a.actPanel(platform.KeyEnter)
		if a.screen != ScreenCredits || a.panel.cursor != before {
			t.Fatalf("A acted on %q", e.text)
		}
		if a.actPanel(platform.KeyDown); a.panel.cursor == before {
			break
		}
	}
	ns := len(a.supporters.Current) + len(a.supporters.Past)
	if seen != 1+len(builtOn)+ns+len(earlyAdopters) {
		t.Fatalf("%d credited rows, want %d", seen, 1+len(builtOn)+ns+len(earlyAdopters))
	}
	if e := a.panel.entries[a.panel.cursor]; e.text != earlyAdopters[len(earlyAdopters)-1] {
		t.Fatalf("Down ends on %q", e.text)
	}
	a.actPanel(platform.KeyBack)
	if a.screen != ScreenOptions || a.panel.entries[a.panel.cursor].kind != "credits" {
		t.Fatalf("B returned to %v on %q", a.screen, a.panel.entries[a.panel.cursor].text)
	}
	a.actPanel(platform.KeyBack)
	if a.screen != ScreenList {
		t.Fatalf("B from Options went to %v", a.screen)
	}
}

// The Credits page lists the Patreon supporters between Built on and the
// early adopters: the embedded snapshot at first, then whatever
// SetSupporters brings (the cached or freshly fetched site file), with
// past supporters in their own group only once there are any, and a
// rebuild in place when the page is open.
func TestCreditsSupporters(t *testing.T) {
	rows := []data.Row{{Base: "Arcade", K: "a", Title: "Alpha"}}
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(rows, "", time.Now()), nil)
	if len(a.supporters.Current) == 0 {
		t.Fatal("the embedded supporters snapshot is empty")
	}
	s, err := DecodeSupporters([]byte(`{"updated":"2026-10-01","current":[{"name":" Zed ","since":"2026-10"},{"name":"","since":"2026-10"}],"past":[{"name":"Gone","since":"2026-05","until":"2026-08"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Current) != 1 || s.Current[0].Name != "Zed" || len(s.Past) != 1 {
		t.Fatalf("decoded %+v", s)
	}
	a.actList(platform.KeyBack)
	a.actPanel(platform.KeyEnd)
	a.actPanel(platform.KeyUp)
	a.actPanel(platform.KeyEnter)
	a.SetSupporters(s)
	var texts []string
	for _, e := range a.panel.entries {
		texts = append(texts, e.text)
	}
	page := strings.Join(texts, "\n")
	for _, w := range []string{"Patreon supporters:", "Zed", "Past supporters:", "Gone", "Special thanks to the early adopters:"} {
		if !strings.Contains(page, w) {
			t.Errorf("credits page lacks %q in\n%s", w, page)
		}
	}
	if strings.Index(page, "Built on:") > strings.Index(page, "Zed") || strings.Index(page, "Zed") > strings.Index(page, "Gone") || strings.Index(page, "Gone") > strings.Index(page, "Special thanks") {
		t.Errorf("groups out of order:\n%s", page)
	}
	if a.screen != ScreenCredits || a.panel.entries[a.panel.cursor].text != "Matija Erceg" {
		t.Fatalf("rebuild moved the page to %v on %q", a.screen, a.panel.entries[a.panel.cursor].text)
	}
	a.SetSupporters(Supporters{})
	texts = texts[:0]
	for _, e := range a.panel.entries {
		texts = append(texts, e.text)
	}
	page = strings.Join(texts, "\n")
	if strings.Contains(page, "supporters:") {
		t.Errorf("empty lists still show a supporters group:\n%s", page)
	}
	if _, err := DecodeSupporters([]byte("{")); err == nil {
		t.Error("a broken file decoded")
	}
}
