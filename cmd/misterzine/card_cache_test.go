//go:build linux

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/fetch"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/scan"
)

func TestRepeatedCardFailureStaysQuiet(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		for _, message := range []string{"Card scan failed", "Card scan incomplete"} {
			t.Run(rot.String()+"/"+message, func(t *testing.T) {
				h := backgroundHost(t)
				now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
				ds := data.Ingest(catalogueRows(), "saved", now)
				h.status = []data.Status{data.StatusCurrent, data.StatusCurrent}
				h.savedCard = &cardInventory{}
				h.a = app.New(app.Config{PhysW: 320, PhysH: 240, SafeInsetX: 15, SafeInsetY: 15, Rotation: rot,
					Status: h.cardStatus, TimerNow: func() time.Time { return now }, Now: func() time.Time { return now }}, ds, nil)
				paint := func() []byte {
					h.a.Paint()
					return bytes.Clone(h.a.Logical().Pix)
				}
				fail := func() {
					h.receiveScan(scanResult{gen: ds.Gen, final: true, notice: message})
				}
				fail()
				first := paint()
				h.a.Notice(message+". Using saved list.", 8*time.Second)
				if !bytes.Equal(first, paint()) {
					t.Fatal("failure did not explain that the saved list was retained")
				}
				now = now.Add(9 * time.Second)
				h.a.Tick(now)
				quiet := paint()
				if bytes.Equal(first, quiet) {
					t.Fatal("failure notice did not expire")
				}
				fail()
				if !bytes.Equal(quiet, paint()) {
					t.Fatal("an identical automatic failure covered the status bar again")
				}
				h.manualScan = true
				fail()
				if !bytes.Equal(first, paint()) {
					t.Fatal("a manual retry did not report its failure")
				}
			})
		}
	}
}

func TestBrowsePreviousCatalogueThroughRefresh(t *testing.T) {
	h := backgroundHost(t)
	cat := catalogueRows()
	for _, row := range cat {
		putCartFile(t, h.card, "_Arcade/cores/"+row.Core+".rbf", "core")
		writeMRA(t, h.card, row.MRA, row.Title, row.SN, row.Core)
	}
	scanOnce(t, h, cat, "old")
	h, _, _ = restart(t, h, catalogueRows(), "old", nil)
	old := h.a.Data()
	h.restoreCard(old)
	h.a = app.New(app.Config{PhysW: 320, PhysH: 240, Status: h.cardStatus}, old, nil)
	h.a.SetFilters(data.Filters{Install: data.InstallFound})
	if h.savedCard == nil || h.a.Visible() != 2 {
		t.Fatal("first list did not contain yesterday's games")
	}
	h.a.MoveToKey(cat[0].K)
	added := data.Row{K: "added", Title: "New game", Base: "Arcade", Core: cat[0].Core, SN: "added", MRA: "_Arcade/New.mra"}
	next := []data.Row{added, cat[1], cat[0]}
	writeMRA(t, h.card, added.MRA, added.Title, added.SN, added.Core)
	if err := os.Remove(filepath.Join(h.card, cat[0].MRA)); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(next)
	if err != nil {
		t.Fatal(err)
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(raw))
	fr := fetch.Fresh{Rows: next, Meta: data.Meta{Hash: hash}, RawData: raw, RawMeta: []byte(fmt.Sprintf(`{"hash":%q}`, hash))}
	scan.Motion.SetPaused(true)
	defer scan.Motion.SetPaused(false)
	h.requestScan() // the ordinary startup scan is still checking yesterday's catalogue
	h.swap(fr)
	if h.a.Data() != old || h.a.Visible() != 2 || h.a.CursorKey() != cat[0].K {
		t.Fatal("download cleared or replaced the visible list")
	}
	if _, err := os.Stat(filepath.Join(h.root, "cache", "data.json")); !os.IsNotExist(err) {
		t.Fatal("unscanned download replaced the catalogue on disk")
	}
	// The player can switch games and use the highlighted game's old
	// launch target before the replacement inventory has arrived.
	h.a.MoveToKey(cat[1].K)
	if _, err := h.prepareLaunch(cat[1].MRA); err != nil {
		t.Fatal(err)
	}
	scan.Motion.SetPaused(false)
	for {
		select {
		case r := <-h.scanCh:
			h.receiveScan(r)
			if r.catalogue != nil || h.savedCard == nil || h.a.Data() != old || h.a.Visible() != 2 || h.a.CursorKey() != cat[1].K {
				t.Fatal("an outstanding scan displaced the saved library during the update")
			}
			if !r.final {
				continue
			}
		case <-time.After(5 * time.Second):
			t.Fatal("old startup scan did not finish")
		}
		break
	}
	finishBackground(t, h)
	if h.a.Data().Hash != hash || h.pendingCatalogue.Load() != nil || h.a.Visible() != 2 || h.a.CursorKey() != cat[1].K ||
		!h.status[0].Found() || !h.status[1].Found() || h.status[2].Found() {
		t.Fatalf("reconciled list: %s, cursor %q, statuses %v", h.a.Data().Hash, h.a.CursorKey(), h.status)
	}
	// A restart with the newly downloaded catalogue can restore the
	// matching complete inventory as well.
	rows, meta, _ := h.loadData()
	reopened, _, _ := restart(t, h, rows, meta.Hash, nil)
	reopened.restoreCard(reopened.a.Data())
	if reopened.savedCard == nil || reopened.savedCard.statusOnly || !slices.Equal(h.status, reopened.status) {
		t.Fatal("completed refresh left the next start without matching card data")
	}
}

func TestMismatchedStartupInventoryUsesGameKeys(t *testing.T) {
	previous := backgroundHost(t)
	cat := catalogueRows()
	putCartFile(t, previous.card, "_Arcade/cores/"+cat[0].Core+".rbf", "core")
	writeMRA(t, previous.card, cat[0].MRA, cat[0].Title, cat[0].SN, cat[0].Core)
	scanOnce(t, previous, cat, "yesterday")
	// A 1.2.32 download may already be on disk with yesterday's scan.
	h, _, _ := restart(t, previous, []data.Row{{K: "brand-new"}, cat[1], cat[0]}, "today", nil)
	ds := h.a.Data()
	h.restoreCard(ds)
	if h.savedCard == nil || !h.savedCard.statusOnly || h.cardStatus(0) != data.StatusUnknown ||
		h.cardStatus(1) != data.StatusNotFound || !h.cardStatus(2).Found() || h.altGen != "" || h.owners != nil {
		t.Fatalf("restored outdated positions or launch data: %v", h.status)
	}
	if msg := h.checkProvisionalLaunch(&ds.Rows[2], cat[0].MRA); msg != "" {
		t.Fatal("verified primary could not launch:", msg)
	}
	h.state.Versions = map[string]string{cat[0].K: "_Arcade/_alternatives/Another.mra"}
	if msg := h.checkProvisionalLaunch(&ds.Rows[2], cat[0].MRA); msg == "" {
		t.Fatal("would silently launch the main file for a remembered version")
	}
	h.requestScan()
	finishBackground(t, h)
	if h.savedCard != nil {
		t.Fatal("live scan did not replace provisional statuses")
	}
}

func TestPendingCatalogueFailureAndRetry(t *testing.T) {
	h := backgroundHost(t)
	old := catalogueRows()
	orphanCard(t, h)
	scanOnce(t, h, old, "previous")
	original := h.a.Data()
	h.card = filepath.Join(h.card, "unreadable")
	h.swap(fetch.Fresh{Rows: old, Meta: data.Meta{Hash: "superseded"}})
	h.swap(fetch.Fresh{Rows: []data.Row{old[1]}, Meta: data.Meta{Hash: "latest"}})
	finishBackground(t, h)
	if h.a.Data() != original || h.pendingCatalogue.Load() == nil || h.pendingCatalogue.Load().Meta.Hash != "latest" || h.cardCheckMessage() != "Card scan failed" {
		t.Fatal("failed or superseded scan displaced the last usable library")
	}
	if _, err := os.Stat(filepath.Join(h.root, "cache", "data.json")); !os.IsNotExist(err) {
		t.Fatal("failed refresh displaced the known catalogue on disk")
	}
	// The card can recover with the network still offline. Automatic retries
	// must back off, keep the list, and never require a successful site check.
	h.client = nil
	selected := h.a.CursorKey()
	for _, delay := range []time.Duration{30 * time.Second, time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 10 * time.Minute, 10 * time.Minute} {
		if remaining := time.Until(h.nextScanRetry); remaining < delay-time.Second || remaining > delay {
			t.Fatalf("retry due in %v, want %v", remaining, delay)
		}
		h.retryScan(h.nextScanRetry.Add(-time.Millisecond))
		if h.scanRunning || h.scanPending {
			t.Fatal("retried before the delay elapsed")
		}
		h.retryScan(h.nextScanRetry)
		if !h.scanRunning || !h.nextScanRetry.IsZero() {
			t.Fatal("due retry did not start")
		}
		h.retryScan(time.Now().Add(time.Hour))
		if h.scanPending {
			t.Fatal("timer queued an extra scan while retrying")
		}
		finishBackground(t, h)
		if h.a.Data() != original || h.a.CursorKey() != selected {
			t.Fatal("repeated failure displaced the list or selection")
		}
	}
	// Update All owns the card until it finishes. A due retry must wait
	// without queuing an extra scan after the updater's own rescan.
	h.updatePending = true
	h.retryScan(h.nextScanRetry)
	h.updatePending, h.updateRunning = false, true
	h.retryScan(h.nextScanRetry)
	if h.scanRunning || h.scanPending || h.nextScanRetry.IsZero() {
		t.Fatal("retry interfered with Update All or was lost")
	}
	h.updateRunning = false
	h.card = filepath.Dir(h.card)
	h.retryScan(h.nextScanRetry)
	finishBackground(t, h)
	if h.a.Data().Hash != "latest" || h.pendingCatalogue.Load() != nil || h.cardCheckMessage() != "" ||
		!h.nextScanRetry.IsZero() || h.scanRetryDelay != 0 {
		t.Fatal("retry did not adopt the latest catalogue and clear the error")
	}
	// A later, unrelated failure starts with the short delay again. An
	// explicit Rescan card can recover before that delay has elapsed.
	h.card = filepath.Join(h.card, "unreadable")
	h.requestScan()
	finishBackground(t, h)
	if remaining := time.Until(h.nextScanRetry); remaining < 29*time.Second || remaining > 30*time.Second {
		t.Fatal("successful recovery did not reset the retry delay")
	}
	h.card = filepath.Dir(h.card)
	h.requestScan()
	finishBackground(t, h)
	if !h.nextScanRetry.IsZero() || h.cardCheckMessage() != "" {
		t.Fatal("explicit rescan did not recover immediately")
	}
}

func TestCatalogueOnFirstFailedCardScan(t *testing.T) {
	h := backgroundHost(t)
	h.card = filepath.Join(h.card, "unreadable")
	h.swap(fetch.Fresh{Rows: catalogueRows(), Meta: data.Meta{Hash: "download"}})
	finishBackground(t, h)
	if h.a.Data().Hash != "download" || h.pendingCatalogue.Load() != nil || h.status != nil || h.cardCheckMessage() != "Card scan failed" {
		t.Fatal("an unreadable card blocked the first catalogue or produced install statuses")
	}
	h.card = filepath.Dir(h.card)
	h.retryScan(h.nextScanRetry)
	finishBackground(t, h)
	if h.index == nil || h.cardCheckMessage() != "" || !h.nextScanRetry.IsZero() {
		t.Fatal("a first scan without saved results did not recover automatically")
	}
}

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
