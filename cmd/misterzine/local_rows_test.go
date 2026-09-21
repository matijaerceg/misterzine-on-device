//go:build linux

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/fetch"
)

func writeMRA(t *testing.T, card, rel, name, setname, rbf string) {
	t.Helper()
	p := filepath.Join(card, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(p), 0755)
	body := `<misterromdescription><name>` + name + `</name><setname>` + setname + `</setname><rbf>` + rbf + `</rbf>` +
		`<rotation>horizontal</rotation><rom index="0" zip="` + setname + `.zip"><part name="x"/></rom></misterromdescription>`
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func catalogueRows() []data.Row {
	return []data.Row{
		{K: "colony7", Title: "Colony 7", Base: "Arcade", Src: "distribution_mister", Core: "defender", SN: "colony7", MRA: "_Arcade/Colony 7 (Set 1).mra", Updated: "2026-07-14"},
		{K: "1942", Title: "1942", Base: "Arcade", Src: "distribution_mister", Core: "jt1942", SN: "1942", MRA: "_Arcade/1942.mra", Updated: "2026-06-01"},
	}
}

func TestReceiveScanInstallsLocalRows(t *testing.T) {
	h := backgroundHost(t)
	os.MkdirAll(filepath.Join(h.card, "_Arcade", "cores"), 0755)
	os.WriteFile(filepath.Join(h.card, "_Arcade", "cores", "Defender_20260714.rbf"), []byte("x"), 0644)
	writeMRA(t, h.card, "_Arcade/Colony 7 (Set 1).mra", "Colony 7", "colony7", "defender")
	writeMRA(t, h.card, "_Arcade/_Extra/Orphan.mra", "Orphan", "orphan", "defender")
	writeMRA(t, h.card, "_Arcade/_Extra/Orphan (set 2).mra", "Orphan", "orphan", "defender")
	writeMRA(t, h.card, "_Arcade/_Organized/_O/Orphan.mra", "Orphan", "orphan", "defender")
	h.a.SetData(data.Ingest(catalogueRows(), "cat", time.Now()), nil)
	h.a.MoveToKey("1942") // kept by key through the swap
	h.requestScan()
	finishBackground(t, h)

	ds := h.a.Data()
	if ds.Hash != "cat" || ds.NCat != 2 || len(ds.Rows) != 3 || ds.Gen == ds.Hash {
		t.Fatalf("dataset after scan: hash=%q ncat=%d rows=%d gen=%q", ds.Hash, ds.NCat, len(ds.Rows), ds.Gen)
	}
	local := ds.Rows[2]
	if local.K != "local:orphan" || local.MRA != "_Arcade/_Extra/Orphan.mra" || local.Core != "defender" || !local.IsLocal() {
		t.Fatalf("local row: %+v", local)
	}
	// A local row has no shipped build to compare with: "on card, date unknown".
	if len(h.status) != 3 || h.status[2] != data.StatusFoundUndated || h.status[0] != data.StatusCurrent || h.status[1] != data.StatusNotFound {
		t.Fatalf("statuses: %v", h.status)
	}
	if h.altGen != ds.Gen || len(h.alts["local:orphan"]) != 1 || h.alts["local:orphan"][0] != "_Arcade/_Extra/Orphan (set 2).mra" {
		t.Fatalf("alternatives: gen=%q alts=%v", h.altGen, h.alts)
	}
	if h.a.CursorKey() != "1942" {
		t.Fatalf("cursor moved to %q", h.a.CursorKey())
	}
	if h.scanPending {
		t.Fatal("a settled local set must not queue another scan")
	}

	// A second scan with an unchanged card reports the same generation and
	// installs nothing new.
	h.requestScan()
	finishBackground(t, h)
	if h.a.Data() != ds || h.altGen != ds.Gen || len(h.status) != 3 {
		t.Fatal("unchanged card replaced the dataset")
	}

	// Removing the file drops the row on the next scan.
	os.Remove(filepath.Join(h.card, "_Arcade", "_Extra", "Orphan.mra"))
	os.Remove(filepath.Join(h.card, "_Arcade", "_Extra", "Orphan (set 2).mra"))
	dir := filepath.Join(h.card, "_Arcade", "_Extra")
	st, _ := os.Stat(dir)
	os.Chtimes(dir, st.ModTime().Add(2e9), st.ModTime().Add(2e9))
	h.requestScan()
	finishBackground(t, h)
	if ds = h.a.Data(); len(ds.Rows) != 2 || ds.Gen != "cat" || len(h.status) != 2 {
		t.Fatalf("removed file kept its row: rows=%d gen=%q", len(ds.Rows), ds.Gen)
	}
}

// A game the catalogue lists only for a core this card has not got: the file
// that does play it becomes a standin row beside the greyed catalogue one,
// and hands its star over once the catalogue's own core arrives.
func TestStandinRowFromScan(t *testing.T) {
	h := backgroundHost(t)
	cores := filepath.Join(h.card, "_Arcade", "cores")
	os.MkdirAll(cores, 0755)
	os.WriteFile(filepath.Join(cores, "taitox_20260904.rbf"), []byte("x"), 0644)
	writeMRA(t, h.card, "_Arcade/_Extra/Gigandes.mra", "Gigandes", "gigandes", "taitox")
	cat := append(catalogueRows(), data.Row{K: "gigandes", Title: "Gigandes", Base: "Arcade", Src: "jtbindb",
		Core: "jttaitox", SN: "gigandes", MRA: "_Arcade/Gigandes.mra", Beta: true, Updated: "2026-09-07"})
	favs := map[string]bool{"local:gigandes": true} // starred while it was the only copy that runs
	h.a = app.New(app.Config{PhysW: 320, PhysH: 240,
		Favorites:  favs,
		FavChanged: func() { h.favDirty = true },
	}, data.Ingest(cat, "cat", time.Now()), nil)
	h.favDirty = false
	h.requestScan()
	finishBackground(t, h)

	ds := h.a.Data()
	if ds.NCat != 3 || len(ds.Rows) != 4 {
		t.Fatalf("dataset after scan: ncat=%d rows=%d", ds.NCat, len(ds.Rows))
	}
	if r := ds.Rows[3]; r.K != "local:gigandes" || !r.Standin || r.Core != "taitox" || r.MRA != "_Arcade/_Extra/Gigandes.mra" {
		t.Fatalf("standin row: %+v", ds.Rows[3])
	}
	// The catalogue's own row stays greyed: its core really is absent.
	if len(h.status) != 4 || h.status[2] != data.StatusNotFound || h.status[3] != data.StatusFoundUndated {
		t.Fatalf("statuses: %v", h.status)
	}

	// The beta core arrives: the catalogue row can run it now, so the standin
	// gives way and the star follows the game to the catalogue.
	os.WriteFile(filepath.Join(cores, "jttaitox_20260907.rbf"), []byte("x"), 0644)
	h.requestScan()
	finishBackground(t, h)
	if ds = h.a.Data(); len(ds.Rows) != 3 || ds.Gen != "cat" {
		t.Fatalf("standin kept after the core arrived: rows=%d gen=%q", len(ds.Rows), ds.Gen)
	}
	if f := h.a.FavoriteSet(); !f["gigandes"] || f["local:gigandes"] || !h.favDirty {
		t.Fatalf("star not handed over: %v dirty=%v", f, h.favDirty)
	}
}

// A catalogue refresh alone says nothing about this card's cores, so a
// standin row keeps its key until a scan finds the game runnable.
func TestSwapKeepsStandinKey(t *testing.T) {
	h := backgroundHost(t)
	stand := data.Row{K: "local:gigandes", Title: "Gigandes", Base: "Arcade", Src: data.SrcLocal, SN: "gigandes",
		Core: "taitox", MRA: "_Arcade/_Extra/Gigandes.mra", Standin: true}
	h.a = app.New(app.Config{PhysW: 320, PhysH: 240,
		Favorites:  map[string]bool{"local:gigandes": true},
		FavChanged: func() { h.favDirty = true },
	}, data.Ingest(data.MergeLocal(catalogueRows(), []data.Row{stand}), "cat", time.Now()), nil)
	h.favDirty = false
	rows := append(catalogueRows(), data.Row{K: "gigandes", Title: "Gigandes", Base: "Arcade", Src: "jtbindb",
		Core: "jttaitox", SN: "gigandes", MRA: "_Arcade/Gigandes.mra"})
	h.swap(fetch.Fresh{Changed: true, Rows: rows, Meta: data.Meta{Hash: "new", Updated: "2026-09-17T00:00Z"}})
	ds := h.a.Data()
	if ds.NCat != 3 || len(ds.Rows) != 4 || ds.Rows[3].K != "local:gigandes" {
		t.Fatalf("swap dropped the standin: ncat=%d rows=%d", ds.NCat, len(ds.Rows))
	}
	if f := h.a.FavoriteSet(); !f["local:gigandes"] || f["gigandes"] {
		t.Fatalf("star moved to a row the card cannot run: %v", f)
	}
	finishBackground(t, h)
	// The rescan of an empty card drops it: the file is not there either.
	if ds = h.a.Data(); len(ds.Rows) != 3 {
		t.Fatalf("rescan: rows=%d", len(ds.Rows))
	}
}

func TestReceiveScanStaleGenIgnored(t *testing.T) {
	h := backgroundHost(t)
	h.a.SetData(data.Ingest(catalogueRows(), "cat", time.Now()), nil)
	stale := data.MergeLocal(catalogueRows(), []data.Row{{K: "local:x", Title: "x", Base: "Arcade", Src: data.SrcLocal, Core: "c", MRA: "_Arcade/x.mra"}})
	h.receiveScan(scanResult{gen: "other", final: true, rows: stale, nextGen: data.Generation("cat", stale[2:]),
		status: []data.Status{data.StatusCurrent, data.StatusCurrent, data.StatusCurrent}, alts: map[string][]string{"local:x": {"y"}}})
	// Nothing installed; a scan of the current generation is queued or
	// already under way (a final result starts the pending scan itself).
	if len(h.a.Data().Rows) != 2 || len(h.status) != 0 || len(h.alts) != 0 || !(h.scanPending || h.scanRunning) {
		t.Fatalf("a result for another generation was installed: rows=%d status=%v alts=%v pending=%v running=%v",
			len(h.a.Data().Rows), h.status, h.alts, h.scanPending, h.scanRunning)
	}
	finishBackground(t, h)
	if len(h.status) != 2 || h.altGen != "cat" {
		t.Fatalf("rescan of the current generation: status=%v altGen=%q", h.status, h.altGen)
	}
}

func TestSwapKeepsLocals(t *testing.T) {
	h := backgroundHost(t)
	local := []data.Row{
		{K: "local:orphan", Title: "Orphan", Base: "Arcade", Src: data.SrcLocal, SN: "orphan", Core: "defender", MRA: "_Arcade/_Extra/Orphan.mra"},
		{K: "local:soon", Title: "Soon", Base: "Arcade", Src: data.SrcLocal, SN: "soon", Core: "defender", MRA: "_Arcade/_Extra/Soon.mra"},
	}
	// The local "soon" is starred, has a remembered version and a launch.
	h.a = app.New(app.Config{PhysW: 320, PhysH: 240,
		Favorites:      map[string]bool{"local:soon": true},
		Versions:       map[string]string{"local:soon": "_Arcade/_Extra/Soon (set 2).mra"},
		RecentLaunches: []data.Recent{{K: "local:soon", At: "2026-09-17T10:00:00Z"}},
		FavChanged:     func() { h.favDirty = true },
		VersionChanged: func() { h.dirty = true },
		RecentsChanged: func() { h.dirty = true },
	}, data.Ingest(data.MergeLocal(catalogueRows(), local), "cat", time.Now()), nil)
	h.favDirty, h.dirty = false, false
	// The new catalogue lists "soon": that local row must go; the other stays.
	rows := append(catalogueRows(), data.Row{K: "soon", Title: "Soon", Base: "Arcade", Src: "coinop", Core: "defender", SN: "soon", MRA: "_Arcade/_Coin-Op/Soon.mra"})
	h.swap(fetch.Fresh{Changed: true, Rows: rows, Meta: data.Meta{Hash: "new", Updated: "2026-09-17T00:00Z"}})
	ds := h.a.Data()
	if ds.Hash != "new" || ds.NCat != 3 || len(ds.Rows) != 4 || ds.Rows[3].K != "local:orphan" {
		t.Fatalf("swap: hash=%q ncat=%d rows=%d", ds.Hash, ds.NCat, len(ds.Rows))
	}
	// ...and its star, version and launch record follow it to the new row.
	if f := h.a.FavoriteSet(); !f["soon"] || f["local:soon"] || !h.favDirty {
		t.Fatalf("favorite not moved: %v dirty=%v", f, h.favDirty)
	}
	if v := h.a.Versions(); v["soon"] != "_Arcade/_Extra/Soon (set 2).mra" || v["local:soon"] != "" || !h.dirty {
		t.Fatalf("version not moved: %v dirty=%v", v, h.dirty)
	}
	if r := h.a.Recents(); len(r) != 1 || r[0].K != "soon" {
		t.Fatalf("recent not moved: %v", r)
	}
	if !h.scanRunning {
		t.Fatal("swap must rescan the card against the new catalogue")
	}
	finishBackground(t, h)
	// An empty card: the rescan drops the stale local row.
	if ds = h.a.Data(); len(ds.Rows) != 3 || ds.Gen != "new" {
		t.Fatalf("rescan: rows=%d gen=%q", len(ds.Rows), ds.Gen)
	}
}
