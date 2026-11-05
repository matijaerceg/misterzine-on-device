package app

import (
	"fmt"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// SetSourceFiles receives the same scan that supplies the Sources filter.
// A nil snapshot means that a scan has not reported yet.
func (a *App) SetSourceFiles(files []data.SourceFile) {
	a.sourceFiles = files
	if a.screen == ScreenSourceFiles {
		a.buildPanel()
		a.all = true
	}
}

func (a *App) sourceFileEntries() []panelEntry {
	// Match paintPanel's frame, text padding and scroll-arrow gutter. Wrap
	// paths too, so a long drop-in filename remains readable at narrow insets.
	cols := a.sm.Cols(a.lay.Body.Dx() - 8 - a.sm.W)
	var entries []panelEntry
	add := func(text, key string, heading bool) {
		for i, line := range gfx.Wrap(data.ASCII(text), cols, 1000) {
			entries = append(entries, panelEntry{text: line, kind: "source-file-line", value: fmt.Sprintf("%s/%d", key, i), header: heading})
		}
	}
	add("Paths start at the card root.", "intro", false)
	add("Refresh with Rescan card.", "refresh", false)
	if a.sourceFiles == nil {
		add("Waiting for the card scan.", "waiting", false)
		return entries
	}
	// Put files that actually supply databases first; keep all other checks
	// visible below them, including the alternative main INI and empty globs.
	for _, used := range []bool{true, false} {
		for _, f := range a.sourceFiles {
			if (f.State == data.SourceFileUsed) != used {
				continue
			}
			entries = append(entries, panelEntry{header: true, info: true})
			add(f.State+": "+f.Path, f.Path, true)
			if f.Note != "" {
				add(f.Note, f.Path+"/note", false)
			}
			for i, s := range f.Sections {
				key := fmt.Sprintf("%s/section/%d", f.Path, i)
				section := "[" + s.ID + "]"
				if s.Default {
					section = "Downloader default:"
				}
				add(section, key, false)
				for _, src := range s.Sources {
					add("> "+data.SrcFull(src), key+"/"+src, false)
				}
				if len(s.Sources) == 0 {
					add("No matching catalogue source", key+"/unknown", false)
				}
			}
		}
	}
	return entries
}

func (a *App) openSourceFiles() {
	a.screen = ScreenSourceFiles
	a.panel.entries = nil
	a.panel.cursor, a.panel.top = 0, 0
	a.buildPanel()
	a.all = true
}

func (a *App) closeSourceFiles() {
	a.openOptions()
	for i, e := range a.panel.entries {
		if e.kind == "source-files" {
			a.panel.cursor = i
			break
		}
	}
	a.all = true
}
