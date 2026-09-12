package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestHeldGroupJumpsMatchRowScrolling(t *testing.T) {
	now := time.Unix(100, 0)
	var groups, singles []data.Row
	for i := 0; i < 26; i++ {
		date := time.Date(2024, time.Month(i+1), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		for j := 0; j < 2; j++ {
			title := fmt.Sprintf("%c %d", 'A'+i, j)
			row := data.Row{K: title, Title: title, Date: date, Updated: date}
			groups = append(groups, row)
			if j == 0 {
				singles = append(singles, row)
			}
		}
	}
	for _, mode := range []data.SortMode{data.SortUpdated, data.SortDebut, data.SortAlphabetical} {
		for _, delay := range []int{200, 300, 500} {
			for _, speed := range []string{"20", "30", "60"} {
				for _, direction := range []struct{ group, row platform.Key }{
					{platform.KeyPageUp, platform.KeyUp}, {platform.KeyPageDown, platform.KeyDown},
				} {
					t.Run(fmt.Sprintf("%v/%d/%s/%v", mode, delay, speed, direction.group), func(t *testing.T) {
						cfg := Config{PhysW: 320, PhysH: 240, RememberSort: true, LastSort: mode, HoldDelay: delay, Scroll: speed, Now: func() time.Time { return now }}
						a := New(cfg, data.Ingest(groups, "test", now), nil)
						ref := New(cfg, data.Ingest(singles, "test", now), nil)
						a.MoveToKey("N 0")
						ref.MoveToKey("N 0")
						a.Handle(platform.Event{Key: direction.group, Pressed: true, At: now})
						ref.Handle(platform.Event{Key: direction.row, Pressed: true, At: now})
						first := a.CursorKey()
						deadline := now.Add(time.Duration(delay) * time.Millisecond)
						a.Frame(deadline.Add(-time.Millisecond))
						if first == "N 0" || a.CursorKey() != first || !a.NextTick().Equal(deadline) {
							t.Fatal("first jump or selected hold delay was not respected")
						}
						for frame := 0; frame < 9; frame++ {
							at := deadline.Add(time.Duration(frame) * frameDur)
							a.Frame(at)
							ref.Frame(at)
							wantTop := a.screenLine(a.cursor)
							if a.groupHeaders() {
								wantTop = centeredTop(a.screenLine(a.cursor), a.totalLines(), a.lay.Lines)
							}
							if a.CursorKey() != ref.CursorKey() || a.top != wantTop {
								t.Fatalf("frame %d: group %q / row %q, top %d want %d", frame, a.CursorKey(), ref.CursorKey(), a.top, wantTop)
							}
						}
						last := a.CursorKey()
						if last == first {
							t.Fatal("held shoulder never repeated")
						}
						a.Handle(platform.Event{Key: direction.group, At: deadline.Add(9 * frameDur)})
						a.Frame(deadline.Add(time.Second))
						if a.CursorKey() != last || a.Repeating() {
							t.Fatal("release did not stop jumps")
						}
					})
				}
			}
		}
	}
}

func TestLetterJumpsCenterTheGroupUnderItsHeader(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		for _, groupSize := range []int{2, 30} {
			rows := []data.Row{}
			for _, initial := range []string{"A", "B", "Z"} {
				for i := 0; i < groupSize; i++ {
					title := fmt.Sprintf("%s %02d", initial, i)
					rows = append(rows, data.Row{K: title, Title: title})
				}
			}
			a := New(Config{PhysW: 320, PhysH: 240, Rotation: rot, RememberSort: true, LastSort: data.SortAlphabetical}, data.Ingest(rows, "test", time.Now()), nil)
			if len(a.marks) != 3 || a.markText(0) != "A" || a.markText(1) != "B" || a.markText(2) != "Z" {
				t.Fatalf("rotation=%v size=%d: letter headers %v", rot, groupSize, a.marks)
			}
			jump := func(key platform.Key, want string) {
				t.Helper()
				a.actList(key)
				a.Paint()
				if a.CursorKey() != want || a.top != centeredTop(a.screenLine(a.cursor), a.totalLines(), a.lay.Lines) || a.shortPage || a.notice != "" {
					t.Fatalf("rotation=%v size=%d: %v selected %q at line %d, top %d, notice %q; want %q centered", rot, groupSize, key, a.CursorKey(), a.screenLine(a.cursor), a.top, a.notice, want)
				}
				if a.markAt(a.cursor) && a.top > 0 && a.screenLine(a.cursor)-1 < a.top {
					t.Fatalf("rotation=%v size=%d: the header of %q scrolled off above", rot, groupSize, want)
				}
			}
			jump(platform.KeyPageDown, "B 00")
			jump(platform.KeyPageDown, "Z 00")
			// Row movement keeps the centered scrolling, bounded by the list end.
			a.actList(platform.KeyDown)
			a.Paint()
			if a.top != centeredTop(a.screenLine(a.cursor), a.totalLines(), a.lay.Lines) {
				t.Fatal("walking after a jump did not keep centered scrolling")
			}
			jump(platform.KeyPageUp, "B 00")
			jump(platform.KeyPageUp, "A 00")
			jump(platform.KeyPageDown, "B 00")
			a.actList(platform.KeyEnd)
			if a.top != max(0, a.totalLines()-a.lay.Lines) {
				t.Fatal("End should still fill the final page")
			}
		}
	}
}

func TestLetterJumpUsesVisibleCollationGroups(t *testing.T) {
	rows := []data.Row{}
	for _, title := range []string{"!Game", "2 Game", "10 Game", "Alpha", "alpha 2", "Beta", "beta 2", "Éclair", "Elephant", "Zoo"} {
		rows = append(rows, data.Row{K: title, Title: title})
	}
	a := New(Config{PhysW: 320, PhysH: 240, RememberSort: true, LastSort: data.SortAlphabetical}, data.Ingest(rows, "test", time.Now()), nil)
	step := func(key platform.Key, want string) {
		t.Helper()
		a.actList(key)
		if a.CursorKey() != want {
			t.Fatalf("%v selected %q; want %q", key, a.CursorKey(), want)
		}
	}
	step(platform.KeyPageDown, "Alpha") // numbers and symbols share #
	step(platform.KeyDown, "alpha 2")
	step(platform.KeyPageDown, "Beta")
	step(platform.KeyDown, "beta 2")
	step(platform.KeyPageUp, "Alpha") // first item of previous group
	step(platform.KeyPageUp, "!Game")
	step(platform.KeyPageUp, "!Game")  // no wrapping
	step(platform.KeyEnd, "Zoo")       // keyboard end remains end
	step(platform.KeyPageUp, "Éclair") // accent uses E, like sorting
	step(platform.KeyPageDown, "Zoo")
	step(platform.KeyPageDown, "Zoo")
	step(platform.KeyHome, "!Game")
	if len(a.marks) != 5 || a.markText(0) != "0-9 and symbols" || a.markText(1) != "A" || a.markText(2) != "B" || a.markText(3) != "E" || a.markText(4) != "Z" {
		t.Fatalf("letter headers: %v", a.marks)
	}

	a.cfg.Favorites = map[string]bool{"Alpha": true, "Éclair": true, "Zoo": true}
	a.SetFilters(data.Filters{FavOnly: true})
	step(platform.KeyHome, "Alpha")
	step(platform.KeyPageDown, "Éclair") // B filtered out
	a.setSearch("o")
	step(platform.KeyPageUp, "Zoo") // search restricts the same view
	a.setSearch("no matches")
	step(platform.KeyPageDown, "")
	step(platform.KeyPageUp, "")
}

func TestMonthJumpsUseSortDateVisibleGroupsAndTopAlignment(t *testing.T) {
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	rows := []data.Row{
		{K: "new", Title: "Newest", Updated: "2026-09-10", Date: "2024-02-10"},
		{K: "same-month", Title: "Same update month", Updated: "2026-09-01", Date: "2024-01-01"},
		{K: "july", Title: "July 2026", Updated: "2026-07-15", Date: "2023-12-15"},
		{K: "older-july", Title: "July 2025", Updated: "2025-07-20", Date: "2023-01-20"},
		{K: "undated", Title: "Undated"},
	}
	base := map[string]string{}
	for _, row := range rows {
		base[row.K] = row.Updated
	}
	base["new"] = "2026-08-01"
	for _, mode := range []data.SortMode{data.SortUpdated, data.SortDebut} {
		a := New(Config{PhysW: 320, PhysH: 240, RememberSort: true, LastSort: mode, Now: func() time.Time { return now }, ClockTrusted: true},
			data.Ingest(rows, "test", now), &data.SeenRecord{T: now.Format(time.RFC3339), Base: base})
		jump := func(key platform.Key, want, label string) {
			t.Helper()
			a.actList(key)
			a.Paint()
			if a.CursorKey() != want || a.top != a.screenLine(a.cursor) || a.notice != label {
				t.Fatalf("%v %v: got %q at top %d (row %d), notice %q; want %q / %q", mode, key, a.CursorKey(), a.top, a.screenLine(a.cursor), a.notice, want, label)
			}
		}
		if mode == data.SortUpdated {
			if a.split < 0 {
				t.Fatal("fixture must exercise the last-look divider")
			}
			jump(platform.KeyPageDown, "july", "July 2026")
			jump(platform.KeyPageDown, "older-july", "July 2025")
			jump(platform.KeyPageDown, "undated", "Date unknown")
			jump(platform.KeyPageDown, "undated", "Date unknown")
			jump(platform.KeyPageUp, "older-july", "July 2025")
			jump(platform.KeyPageUp, "july", "July 2026")
			jump(platform.KeyPageUp, "new", "September 2026")
		} else {
			jump(platform.KeyPageDown, "same-month", "January 2024")
			jump(platform.KeyPageDown, "july", "December 2023")
			jump(platform.KeyPageDown, "older-july", "January 2023")
			jump(platform.KeyPageUp, "july", "December 2023")
			jump(platform.KeyPageUp, "same-month", "January 2024")
			jump(platform.KeyPageUp, "new", "February 2024")
		}
		a.cfg.Favorites = map[string]bool{"new": true, "older-july": true, "undated": true}
		a.SetFilters(data.Filters{FavOnly: true})
		a.actList(platform.KeyHome)
		label := "July 2025"
		if mode == data.SortDebut {
			label = "January 2023"
		}
		jump(platform.KeyPageDown, "older-july", label)
		a.setSearch("July")
		a.actList(platform.KeyPageDown)
		if a.CursorKey() != "older-july" {
			t.Fatal("jump escaped search results")
		}
		a.setSearch("no matches")
		a.actList(platform.KeyPageUp)
		a.actList(platform.KeyPageDown)
		if a.CursorKey() != "" {
			t.Fatal("empty results acquired a selection")
		}
	}
}
