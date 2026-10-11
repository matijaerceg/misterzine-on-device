//go:build linux

package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// orphanCard puts the local game the tests share on h's card: Orphan on the
// Defender core, which no catalogue row names.
func orphanCard(t *testing.T, h *host) {
	t.Helper()
	os.MkdirAll(filepath.Join(h.card, "_Arcade", "cores"), 0755)
	os.WriteFile(filepath.Join(h.card, "_Arcade", "cores", "Defender_20260714.rbf"), []byte("x"), 0644)
	writeMRA(t, h.card, "_Arcade/_Extra/Orphan.mra", "Orphan", "orphan", "defender")
}

// scanOnce runs one complete scan of h's card against the catalogue cat,
// which leaves the local rows it found in h's cache folder.
func scanOnce(t *testing.T, h *host, cat []data.Row, hash string) {
	t.Helper()
	h.a.SetData(data.Ingest(cat, hash, time.Now()), nil)
	h.requestScan()
	finishBackground(t, h)
}

// restart is the start of the next run on the same card and cache: the
// catalogue rows plus the saved local rows, as run() builds its dataset.
func restart(t *testing.T, prev *host, cat []data.Row, hash string, favs map[string]bool) (*host, int, string) {
	t.Helper()
	h := backgroundHost(t)
	h.root, h.card = prev.root, prev.card
	rows, n, note := h.withSavedLocalRows(cat, hash)
	h.localSaved = n > 0
	h.a = app.New(app.Config{PhysW: 320, PhysH: 240, Favorites: favs, FavChanged: func() { h.favDirty = true }},
		data.Ingest(rows, hash, time.Now()), nil)
	return h, n, note
}

// The report on 2026-10-10: a 2 TB card took 16 s to scan, and the list
// showed 308 fewer games until it had. The next start lists them at once.
func TestSavedLocalRowsListedAtStartup(t *testing.T) {
	first := backgroundHost(t)
	orphanCard(t, first)
	scanOnce(t, first, catalogueRows(), "cat")
	if _, err := os.Stat(filepath.Join(first.root, "cache", localRowsName)); err != nil {
		t.Fatalf("a complete scan saved no local rows: %v", err)
	}

	h, n, note := restart(t, first, catalogueRows(), "cat", map[string]bool{"local:orphan": true})
	ds := h.a.Data()
	if n != 1 || note != " + 1 local rows from the last scan" || ds.NCat != 2 || len(ds.Rows) != 3 || ds.Rows[2].K != "local:orphan" {
		t.Fatalf("startup: n=%d note=%q ncat=%d rows=%d", n, note, ds.NCat, len(ds.Rows))
	}
	if ds.Gen != first.a.Data().Gen {
		t.Fatalf("saved rows read back as another generation: %q vs %q", ds.Gen, first.a.Data().Gen)
	}
	if !h.a.FavoriteSet()["local:orphan"] {
		t.Fatal("the star on the saved local row is gone")
	}
	// The scan finds the same card: the saved rows stay, the statuses index
	// into them, and nothing is left waiting for a recheck.
	h.requestScan()
	finishBackground(t, h)
	if h.a.Data() != ds || len(h.status) != 3 || h.status[2] != data.StatusFoundUndated || h.altGen != ds.Gen {
		t.Fatalf("unchanged card: dataset replaced=%v status=%v altGen=%q", h.a.Data() != ds, h.status, h.altGen)
	}
	if h.localSaved || h.scanPending || !h.a.FavoriteSet()["local:orphan"] {
		t.Fatalf("after the scan: saved=%v pending=%v favs=%v", h.localSaved, h.scanPending, h.a.FavoriteSet())
	}
}

func TestNoSavedLocalRows(t *testing.T) {
	first := backgroundHost(t)
	h, n, note := restart(t, first, catalogueRows(), "cat", nil)
	if n != 0 || note != "" || h.localSaved || len(h.a.Data().Rows) != 2 {
		t.Fatalf("no file: n=%d note=%q rows=%d", n, note, len(h.a.Data().Rows))
	}
	// a card without local games saves an empty set, which adds nothing
	scanOnce(t, first, catalogueRows(), "cat")
	if _, err := os.Stat(filepath.Join(first.root, "cache", localRowsName)); err != nil {
		t.Fatalf("no file after a complete scan: %v", err)
	}
	if _, n, note = restart(t, first, catalogueRows(), "cat", nil); n != 0 || note != "" {
		t.Fatalf("empty set: n=%d note=%q", n, note)
	}
}

// The run that fetched a new catalogue ended before its rescan did: the
// saved rows are the old catalogue's. A game the new one lists leaves the
// list at once and the first complete scan moves its star; the rest stay.
func TestSavedLocalRowsOfAnotherCatalogue(t *testing.T) {
	first := backgroundHost(t)
	orphanCard(t, first)
	writeMRA(t, first.card, "_Arcade/_Extra/Soon.mra", "Soon", "soon", "defender")
	scanOnce(t, first, catalogueRows(), "cat")
	if n := len(first.a.Data().Rows); n != 4 {
		t.Fatalf("first scan: %d rows", n)
	}

	cat := append(catalogueRows(), data.Row{K: "soon", Title: "Soon", Base: "Arcade", Src: "coinop", Core: "defender", SN: "soon", MRA: "_Arcade/_Extra/Soon.mra"})
	h, n, note := restart(t, first, cat, "new", map[string]bool{"local:soon": true})
	ds := h.a.Data()
	if n != 1 || note != " + 1 local rows from the last scan (of catalogue cat)" || ds.NCat != 3 || len(ds.Rows) != 4 || ds.Rows[3].K != "local:orphan" {
		t.Fatalf("startup: n=%d note=%q ncat=%d rows=%d", n, note, ds.NCat, len(ds.Rows))
	}
	h.requestScan()
	finishBackground(t, h)
	if f := h.a.FavoriteSet(); !f["soon"] || f["local:soon"] {
		t.Fatalf("the star stayed on the local row the catalogue took over: %v", f)
	}
	if ds = h.a.Data(); len(ds.Rows) != 4 || ds.Rows[3].K != "local:orphan" || h.localSaved {
		t.Fatalf("after the scan: rows=%d saved=%v", len(ds.Rows), h.localSaved)
	}
	// and the file now holds this catalogue's rows
	b, _ := os.ReadFile(filepath.Join(h.root, "cache", localRowsName))
	if hash, rows, err := data.DecodeLocalRows(b); err != nil || hash != "new" || len(rows) != 1 || rows[0].K != "local:orphan" {
		t.Fatalf("saved after the scan: %q %v %v", hash, rows, err)
	}
}

// The card changed while the app was closed: the scan replaces the saved
// rows, a removed game leaving and a retitled one taking its new name,
// although the retitled one alone would not change the generation.
func TestScanReplacesSavedLocalRows(t *testing.T) {
	first := backgroundHost(t)
	orphanCard(t, first)
	writeMRA(t, first.card, "_Arcade/_Extra/Gone.mra", "Gone", "gone", "defender")
	scanOnce(t, first, catalogueRows(), "cat")

	writeMRA(t, first.card, "_Arcade/_Extra/Orphan.mra", "Orphan (World)", "orphan", "defender")
	os.Remove(filepath.Join(first.card, "_Arcade", "_Extra", "Gone.mra"))
	dir := filepath.Join(first.card, "_Arcade", "_Extra")
	st, _ := os.Stat(dir)
	os.Chtimes(dir, st.ModTime().Add(2e9), st.ModTime().Add(2e9))
	h, n, _ := restart(t, first, catalogueRows(), "cat", nil)
	if n != 2 {
		t.Fatalf("startup: %d saved local rows", n)
	}
	h.requestScan()
	finishBackground(t, h)
	ds := h.a.Data()
	if len(ds.Rows) != 3 || ds.Rows[2].K != "local:orphan" || ds.Rows[2].Title != "Orphan (World)" || len(h.status) != 3 || h.localSaved {
		t.Fatalf("after the scan: rows=%d %+v status=%v saved=%v", len(ds.Rows), ds.Rows[2:], h.status, h.localSaved)
	}

	// The retitle alone: same generation, and still the scan's row wins.
	writeMRA(t, h.card, "_Arcade/_Extra/Orphan.mra", "Orphan (Japan)", "orphan", "defender")
	st, _ = os.Stat(dir)
	os.Chtimes(dir, st.ModTime().Add(2e9), st.ModTime().Add(2e9))
	h2, _, _ := restart(t, h, catalogueRows(), "cat", nil)
	gen := h2.a.Data().Gen
	h2.requestScan()
	finishBackground(t, h2)
	if ds = h2.a.Data(); ds.Gen != gen || ds.Rows[2].Title != "Orphan (Japan)" || len(h2.status) != 3 || h2.scanPending {
		t.Fatalf("retitle: gen %q -> %q title=%q status=%v pending=%v", gen, ds.Gen, ds.Rows[2].Title, h2.status, h2.scanPending)
	}
}

// A file this build cannot read is ignored with one log line, and the next
// complete scan writes a good one over it.
func TestUnreadableSavedLocalRowsIgnored(t *testing.T) {
	for _, body := range []string{`{"v":1,"hash":"cat","rows":[{"k":"local:orphan"`, `{"v":99,"hash":"cat","rows":[]}`} {
		first := backgroundHost(t)
		orphanCard(t, first)
		path := filepath.Join(first.root, "cache", localRowsName)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte(body), 0644)

		h := backgroundHost(t)
		h.root, h.card = first.root, first.card
		var buf bytes.Buffer
		h.lg = log.New(&buf, "", 0)
		rows, n, note := h.withSavedLocalRows(catalogueRows(), "cat")
		if n != 0 || note != "" || len(rows) != 2 {
			t.Fatalf("%s: n=%d note=%q rows=%d", body, n, note, len(rows))
		}
		if lines := strings.Split(strings.TrimSpace(buf.String()), "\n"); len(lines) != 1 || !strings.Contains(lines[0], localRowsName+" ignored") {
			t.Fatalf("%s: log %q", body, buf.String())
		}
		scanOnce(t, h, catalogueRows(), "cat")
		b, _ := os.ReadFile(path)
		if _, rows, err := data.DecodeLocalRows(b); err != nil || len(rows) != 1 {
			t.Fatalf("%s: rewritten as %v %v", body, rows, err)
		}
	}
}
