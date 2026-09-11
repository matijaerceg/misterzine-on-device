package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

func TestOpenFiltersFollowNewDataset(t *testing.T) {
	now := time.Now()
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest([]data.Row{
		{K: "one", Base: "Arcade", Genre: "Puzzle"}, {K: "two", Base: "Arcade", Genre: "Sports"},
	}, "old", now), nil)
	openExpandedFilters(a)
	for i, e := range a.panel.entries {
		if e.kind == "genre" && e.value == "Puzzle" {
			a.panel.cursor = i
		}
	}
	a.SetData(data.Ingest([]data.Row{
		{K: "one", Base: "Arcade", Genre: "Puzzle"}, {K: "three", Base: "Arcade", Genre: "Puzzle"}, {K: "four", Base: "Arcade", Genre: "Action"},
	}, "new", now), nil)
	counts := map[string]int{}
	for _, e := range a.panel.entries {
		if e.kind == "genre" && !e.header {
			counts[e.value] = e.count
		}
	}
	if counts["Puzzle"] != 2 || counts["Action"] != 1 || counts["Sports"] != 0 {
		t.Fatalf("stale visible filters: %v", counts)
	}
	selected := a.panel.entries[a.panel.cursor]
	if selected.kind != "genre" || selected.value != "Puzzle" {
		t.Fatalf("selected filter moved: %+v", selected)
	}
}

func TestShrinkingListKeepsLastPageFilled(t *testing.T) {
	for _, rotation := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
		for _, count := range []int{0, 3, 24} {
			t.Run(fmt.Sprintf("%v/%d", rotation, count), func(t *testing.T) {
				now := time.Now()
				rows := make([]data.Row, 100)
				for i := range rows {
					rows[i] = data.Row{K: fmt.Sprint(i), Title: fmt.Sprintf("Game %03d", i)}
				}
				a := New(Config{PhysW: 320, PhysH: 240, Rotation: rotation}, data.Ingest(rows, "old", now), nil)
				a.Paint()
				a.MoveToKey(a.ds.Rows[a.view[len(a.view)-1]].K)
				a.SetData(data.Ingest(rows[:count], "new", now), nil)
				want := max(0, a.totalLines()-a.lay.Lines)
				if a.top != want {
					t.Fatalf("last page starts at %d, want %d (%d lines, %d visible)", a.top, want, a.totalLines(), a.lay.Lines)
				}
				if count > 0 {
					line := a.screenLine(a.cursor)
					if line < a.top || line >= a.top+a.lay.Lines {
						t.Fatal("cursor off-screen")
					}
				}
			})
		}
	}
}
