package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/report"
)

func TestSourceFilesPageAndReportFollowScan(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest([]data.Row{{K: "a", Title: "A", Base: "Arcade"}}, "", time.Now()), nil)
	a.openOptions()
	expandOptionsForTest(a)
	row := -1
	for i, e := range a.panel.entries {
		if e.kind == "source-files" {
			row = i
			if !e.child || !e.opensPage || a.panel.entries[i-1].kind != "sources" {
				t.Fatalf("Source files must be a child of Sources: %+v", e)
			}
		}
	}
	if row < 0 || a.InstalledOnly() {
		t.Fatal("Source files must be available with Sources: all")
	}
	a.panel.cursor = row
	a.actPanel(platform.KeyEnter)
	if a.screen != ScreenSourceFiles || !strings.Contains(sourcePageText(a), "Waiting for the card scan") {
		t.Fatalf("before scan: %s", sourcePageText(a))
	}
	files := []data.SourceFile{
		{Path: "Scripts/downloader.ini", State: data.SourceFileSkipped, Note: "Using downloader.ini"},
		{Path: "downloader.ini", State: data.SourceFileUsed, Sections: []data.SourceSection{{ID: "jtcores", Sources: []string{"jtbindb"}}}},
	}
	a.SetSourceFiles(files)
	text := sourcePageText(a)
	if strings.Index(text, "used: downloader.ini") > strings.Index(text, "skipped: Scripts/downloader.ini") || !strings.Contains(text, "[jtcores]") {
		t.Fatalf("live scan did not rebuild in used-first order: %s", text)
	}
	saved := a.ReportPart()
	text = string(report.Build(report.Input{App: saved}))
	if !strings.Contains(text, "used: downloader.ini") || !strings.Contains(text, "[jtcores] -> jtbindb") {
		t.Fatalf("report lost provenance: %s", text)
	}
	a.actPanel(platform.KeyEnd)
	for _, key := range []platform.Key{platform.KeyEnter, platform.KeySpace, platform.KeyLeft, platform.KeyRight} {
		if a.actPanel(key) {
			t.Fatalf("read-only page acted on %v", key)
		}
	}
	a.SetSourceFiles([]data.SourceFile{{Path: "downloader.ini", State: data.SourceFileMissing}})
	if strings.Contains(sourcePageText(a), "jtcores") || strings.Contains(string(report.Build(report.Input{App: a.ReportPart()})), "jtcores") {
		t.Fatal("new scan retained old databases")
	}
	// Real event dispatch must route Back to Options and restore its child row.
	now := time.Now()
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: now})
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: false, At: now.Add(time.Millisecond)})
	if a.screen != ScreenOptions || a.panel.entries[a.panel.cursor].kind != "source-files" {
		t.Fatalf("Back lost the Options position: %v", a.screen)
	}
}

func sourcePageText(a *App) string {
	var lines []string
	for _, e := range a.panel.entries {
		lines = append(lines, e.text)
	}
	return strings.Join(lines, "\n")
}

func TestSourceFilesLongPathsRemainReadable(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240, Rotation: gfx.RotLeft, SafeInsetX: 40, SafeInsetY: 40}, data.Ingest(nil, "", time.Now()), nil)
	path := "Scripts/downloader/a-very-long-custom-database-file-name.ini"
	a.SetSourceFiles([]data.SourceFile{{Path: path, State: data.SourceFileUsed}})
	a.openSourceFiles()
	var joined string
	for _, e := range a.panel.entries {
		if e.header && !e.info {
			joined += e.text
		}
	}
	if !strings.Contains(strings.ReplaceAll(joined, " ", ""), path) {
		t.Fatalf("long path was truncated: %q", joined)
	}
}
