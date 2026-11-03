package app

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestNeoGeoFilterRemainsSeparateWithOneGame(t *testing.T) {
	neo := data.Row{K: "local:neogeo:mslug:one", Title: "Metal Slug", Src: data.SrcLocal, Base: "Arcade", Core: "NeoGeo", MRA: "_@Favorites/Metal Slug.mgl"}
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest([]data.Row{neo}, "", time.Now()), nil)
	openExpandedFilters(a)
	find := func(kind, value string) int {
		t.Helper()
		for i, e := range a.panel.entries {
			if e.kind == kind && e.value == value && !e.header {
				return i
			}
		}
		t.Fatalf("missing %s %s", kind, value)
		return -1
	}
	a.panel.cursor = find("src", data.SrcLocal)
	a.actPanel(platform.KeyRight)
	a.panel.cursor = find("core", "local/NeoGeo")
	if a.panel.entries[a.panel.cursor].text != "  Neo Geo" {
		t.Fatal("missing readable Neo Geo label")
	}
	a.actPanel(platform.KeyEnter)
	if len(a.view) != 0 || !a.filters.CoreOff["local/NeoGeo"] || a.filters.SrcOff[data.SrcLocal] {
		t.Fatalf("must hide just Neo Geo: %+v", a.filters)
	}
	b, _ := json.Marshal(a.filters)
	var restored data.Filters
	if err := json.Unmarshal(b, &restored); err != nil {
		t.Fatal(err)
	}
	rows := []data.Row{neo,
		{K: "local:neogeo:aof:two", Title: "Art of Fighting", Src: data.SrcLocal, Base: "Arcade", Core: "NeoGeo_LLAPI_20260101"},
		{K: "local:other", Title: "Other local arcade", Src: data.SrcLocal, Base: "Arcade", Core: "Other"},
	}
	reopened := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(rows, "", time.Now()), nil)
	reopened.SetFilters(restored)
	if len(reopened.view) != 1 || reopened.ds.Rows[reopened.view[0]].K != "local:other" {
		t.Fatal("persisted filter must hide variants and newly discovered Neo Geo games only")
	}
	reopened.SetFilters(data.Filters{})
	if len(reopened.view) != 3 {
		t.Fatal("clearing filters must restore games")
	}
}

func TestSeparateGameReturnDoesNotEnableBootLauncher(t *testing.T) {
	for _, supported := range []bool{false, true} {
		a := New(Config{PhysW: 320, PhysH: 240, GameReturn: supported, Launcher: func() bool { return false }}, data.Ingest(nil, "", time.Now()), nil)
		found := 0
		for _, e := range a.optionsEntries() {
			if e.kind == "return-after-game" || e.kind == "exit-chord" {
				found++
				if e.disabled == supported {
					t.Fatalf("%s availability with separate return=%v", e.kind, supported)
				}
			}
			if e.kind == "open-at-boot" && !e.disabled {
				t.Fatal("separate return must not enable boot launcher")
			}
		}
		if found != 2 {
			t.Fatal("missing return options")
		}
	}
}
