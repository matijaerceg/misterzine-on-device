//go:build linux

package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/scan"
)

func TestFoundOnCardReturnAndRefresh(t *testing.T) {
	h := backgroundHost(t)
	cat := catalogueRows()
	orphanCard(t, h)
	writeMRA(t, h.card, cat[0].MRA, cat[0].Title, cat[0].SN, cat[0].Core)
	scanOnce(t, h, cat, "catalogue")
	p := filepath.Join(h.root, "cache", cardCacheName)
	stamp := time.Unix(1000, 0)
	if err := os.Chtimes(p, stamp, stamp); err != nil {
		t.Fatal(err)
	}

	const kept = "local:orphan"
	reopen := func() *host {
		t.Helper()
		warm, _, _ := restart(t, h, catalogueRows(), "catalogue", nil)
		ds := warm.a.Data()
		warm.restoreCard(ds)
		warm.a = app.New(app.Config{PhysW: 320, PhysH: 240, Status: warm.cardStatus}, ds, nil)
		warm.a.SetFilters(data.Filters{Install: data.InstallFound})
		if warm.index != nil || warm.a.Visible() != 2 || !warm.a.ResumeAt(data.Recent{K: kept}) || warm.a.CursorKey() != kept {
			t.Fatal("could not return to the previous game before starting a scan")
		}
		return warm
	}
	warm := reopen()
	ds := warm.a.Data()
	// A partial result must not empty a previously complete list.
	warm.receiveScan(scanResult{gen: ds.Gen, index: &scan.Index{}, status: make([]data.Status, len(ds.Rows))})
	warm.receiveScan(scanResult{gen: ds.Gen, index: &scan.Index{}, final: true, notice: "Card scan incomplete"})
	if warm.index != nil || warm.savedCard == nil || warm.a.Visible() != 2 || warm.a.CursorKey() != kept {
		t.Fatal("an incomplete scan replaced the saved list")
	}
	warm.requestScan()
	finishBackground(t, warm)
	info, err := os.Stat(p)
	if err != nil || !info.ModTime().Equal(stamp) || warm.savedCard != nil || warm.a.CursorKey() != kept {
		t.Fatalf("unchanged scan rewrote the cache or lost the cursor: %v", err)
	}

	if err := os.Remove(filepath.Join(warm.card, cat[0].MRA)); err != nil {
		t.Fatal(err)
	}
	writeMRA(t, warm.card, cat[1].MRA, cat[1].Title, cat[1].SN, cat[1].Core)
	putCartFile(t, warm.card, "_Arcade/cores/jt1942_20261001.rbf", "core")
	warm = reopen()
	if _, err := warm.prepareLaunch(cat[0].MRA); err == nil {
		t.Fatal("a removed game could launch from its cached status")
	}
	warm.requestScan()
	finishBackground(t, warm)
	next, _, _ := restart(t, warm, catalogueRows(), "catalogue", nil)
	next.restoreCard(next.a.Data())
	if warm.a.Visible() != 2 || warm.a.CursorKey() != kept || next.savedCard == nil || !slices.Equal(warm.status, next.status) ||
		next.status[0].Found() || !next.status[1].Found() {
		t.Fatal("added/removed games or the selection did not survive refresh and restart")
	}
}

func TestCachedSharedLauncherChangesOwner(t *testing.T) {
	h := backgroundHost(t)
	rows := catalogueRows()
	rows[1].MRA = rows[0].MRA
	for _, row := range rows {
		putCartFile(t, h.card, "_Arcade/cores/"+row.Core+".rbf", "core")
	}
	writeMRA(t, h.card, rows[0].MRA, rows[0].Title, rows[0].SN, rows[0].Core)
	scanOnce(t, h, rows, "cat")
	next, _, _ := restart(t, h, rows, "cat", nil)
	next.restoreCard(next.a.Data())
	if next.owners[rows[1].K] != rows[0].K {
		t.Fatal("lost the saved owner of a shared file")
	}
	if _, err := next.prepareLaunch(rows[0].MRA); err != nil {
		t.Fatalf("unchanged shared file: %v", err)
	}
	writeMRA(t, h.card, rows[1].MRA, rows[1].Title, rows[1].SN, rows[1].Core)
	if _, err := next.prepareLaunch(rows[0].MRA); err == nil {
		t.Fatal("cached game would launch another game's core")
	}
}

func TestCardCacheFallback(t *testing.T) {
	h := backgroundHost(t)
	cat := catalogueRows()
	scanOnce(t, h, cat, "catalogue")
	ds := h.a.Data()
	p := filepath.Join(h.root, "cache", cardCacheName)
	for _, tc := range []struct {
		name string
		ds   *data.Dataset
	}{
		{"new catalogue", data.Ingest(cat, "new", time.Now())},
		{"different game keys", data.Ingest([]data.Row{{K: "other"}, cat[1]}, ds.Hash, time.Now())},
		{"different scan option", ds},
		{"damaged file", ds},
	} {
		t.Run(tc.name, func(t *testing.T) {
			next := backgroundHost(t)
			next.root = h.root
			next.settings.ScanNeoGeoFolder = tc.name == "different scan option"
			if tc.name == "damaged file" {
				if err := os.WriteFile(p, []byte("{"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			next.restoreCard(tc.ds)
			if next.savedCard != nil || len(next.status) > 0 {
				t.Fatal("unusable cache applied")
			}
		})
	}
}

func TestCachedCollectionAndRememberedVersionAtStart(t *testing.T) {
	h := backgroundHost(t)
	row := catalogueRows()[0]
	row.Family, row.FamilySets = row.SN, []string{"colony7a"}
	primary := "_Ongo/" + row.MRA
	alt := "_Ongo/_Arcade/_alternatives/_Colony 7/Set 2.mra"
	core := "_Ongo/cores/Defender_20260801.rbf"
	writeMRA(t, h.card, primary, row.Title, row.SN, row.Core)
	writeMRA(t, h.card, alt, row.Title, "colony7a", row.Core)
	putCartFile(t, h.card, core, "core")
	scanOnce(t, h, []data.Row{row}, "cat")
	next, _, _ := restart(t, h, []data.Row{row}, "cat", nil)
	ds := next.a.Data()
	next.restoreCard(ds)
	var launched string
	next.a = app.New(app.Config{PhysW: 320, PhysH: 240, Status: next.cardStatus,
		Alternatives: next.alternatives, VersionCore: next.versionCore, Versions: map[string]string{row.K: alt},
		Launch: func(p string) {
			var err error
			launched, err = next.prepareLaunch(p)
			if err != nil {
				t.Fatal(err)
			}
		}}, ds, nil)
	next.a.SetFilters(data.Filters{Install: data.InstallFound})
	next.a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: time.Now()})
	if next.index != nil || ds.Rows[0].MRA != primary || ds.Rows[0].CatalogueMRA != row.MRA || launched != filepath.Join(next.card, alt) {
		t.Fatalf("warm collection launch: primary %s, launched %s", ds.Rows[0].MRA, launched)
	}
	if err := os.Remove(filepath.Join(next.card, core)); err != nil {
		t.Fatal(err)
	}
	if _, err := next.prepareLaunch(alt); err == nil || !strings.Contains(err.Error(), "_Ongo/cores") {
		t.Fatalf("removed collection core: %v", err)
	}
}
