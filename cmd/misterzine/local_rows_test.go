//go:build linux

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/fetch"
	"github.com/matijaerceg/misterzine-on-device/internal/scan"
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

	// The core alone does not make the game run: the list still shows the
	// catalogue row greyed while its own MRA is missing, so the copy that
	// runs keeps standing in.
	os.WriteFile(filepath.Join(cores, "jttaitox_20260907.rbf"), []byte("x"), 0644)
	h.requestScan()
	finishBackground(t, h)
	if ds = h.a.Data(); len(ds.Rows) != 4 || !ds.Rows[3].Standin {
		t.Fatalf("the standin left while the catalogue row is still not on the card: rows=%d", len(ds.Rows))
	}

	// The beta arrives whole, core and MRA: the catalogue row runs now, so
	// the standin gives way, its file becomes one more version of the game,
	// and the star follows the game to the catalogue.
	writeMRA(t, h.card, "_Arcade/Gigandes.mra", "Gigandes", "gigandes", "jttaitox")
	h.requestScan()
	finishBackground(t, h)
	if ds = h.a.Data(); len(ds.Rows) != 3 || ds.Gen != "cat" {
		t.Fatalf("standin kept after the game arrived: rows=%d gen=%q", len(ds.Rows), ds.Gen)
	}
	if f := h.a.FavoriteSet(); !f["gigandes"] || f["local:gigandes"] || !h.favDirty {
		t.Fatalf("star not handed over: %v dirty=%v", f, h.favDirty)
	}
	if vs := h.alternatives(&ds.Rows[2]); len(vs) != 1 || vs[0] != "_Arcade/_Extra/Gigandes.mra" {
		t.Fatalf("the former standin's file is not a version of the game: %v", vs)
	}
}

// A catalogue game that runs here offers the card's sets of it on another
// core that is on the card, labelled with that core, and drops one the
// moment its core leaves: the first pass of a rescan updates the index.
func TestExtraVersionsFromScan(t *testing.T) {
	h := backgroundHost(t)
	cores := filepath.Join(h.card, "_Arcade", "cores")
	os.MkdirAll(cores, 0755)
	os.WriteFile(filepath.Join(cores, "phoenix_20240601.rbf"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(cores, "pleiads_20240601.rbf"), []byte("x"), 0644)
	writeMRA(t, h.card, "_Arcade/Pleiads (Tehkan).mra", "Pleiads (Tehkan)", "pleiads", "phoenix")
	centuri := "_Arcade/_alternatives/_Pleiads/Pleiads (Centuri).mra"
	writeMRA(t, h.card, centuri, "Pleiads (Centuri)", "pleiadce", "pleiads")
	cat := []data.Row{{K: "pleiads", Title: "Pleiads", Base: "Arcade", Src: "distribution_mister", Core: "phoenix", SN: "pleiads",
		Family: "pleiads", FamilySets: []string{"pleiadce"}, MRA: "_Arcade/Pleiads (Tehkan).mra", Updated: "2026-06-01"}}
	h.a = app.New(app.Config{PhysW: 320, PhysH: 240}, data.Ingest(cat, "cat", time.Now()), nil)
	h.requestScan()
	finishBackground(t, h)
	row := &h.a.Data().Rows[0]
	if vs := h.alternatives(row); len(vs) != 1 || vs[0] != centuri || h.altCores[centuri] != "pleiads" {
		t.Fatalf("versions %v, cores %v", vs, h.altCores)
	}
	if d := h.scanDiag; d == nil || !strings.Contains(strings.Join(d.lines, "\n"), "1 extra versions from 1 files") {
		t.Fatalf("scan line: %+v", d)
	}
	os.Remove(filepath.Join(cores, "pleiads_20240601.rbf"))
	h.index = scan.ScanCores(h.card)
	if vs := h.alternatives(row); len(vs) != 0 {
		t.Fatalf("a version whose core left is still offered: %v", vs)
	}
}

// One game in two catalogue implementations: a standin that gives way when
// one of them arrives hands its star to that one, which offers its file,
// not to whichever row the setname points at on paper.
func TestStandinHandsOverToTheRowThatRuns(t *testing.T) {
	h := backgroundHost(t)
	cores := filepath.Join(h.card, "_Arcade", "cores")
	os.MkdirAll(cores, 0755)
	os.WriteFile(filepath.Join(cores, "otherbh_20260101.rbf"), []byte("x"), 0644)
	writeMRA(t, h.card, "_Arcade/_Extra/Black Heart (other).mra", "Black Heart", "blkheart", "otherbh")
	cat := []data.Row{
		{K: "blkheart", Title: "Black Heart (Coin-Op Collection)", Base: "Arcade", Src: "coinop", Core: "blkheart_mister", SN: "blkheart", MRA: "_Arcade/Black Heart (Coin-Op).mra", Updated: "2026-09-09"},
		{K: "black-heart", Title: "Black Heart", Base: "Arcade", Src: "distribution_mister", Core: "Arcade-NMK16_Gunnail", SN: "blkheart", MRA: "_Arcade/Black Heart.mra", Updated: "2026-09-19"},
	}
	h.a = app.New(app.Config{PhysW: 320, PhysH: 240, Favorites: map[string]bool{"local:blkheart": true}, FavChanged: func() { h.favDirty = true }},
		data.Ingest(cat, "cat", time.Now()), nil)
	h.requestScan()
	finishBackground(t, h)
	if ds := h.a.Data(); len(ds.Rows) != 3 || !ds.Rows[2].Standin {
		t.Fatalf("no standin while neither implementation is here: %d rows", len(ds.Rows))
	}
	// Coin-Op's release arrives; the NMK one, later in the catalogue and
	// the setname's owner on paper, does not
	os.WriteFile(filepath.Join(cores, "blkheart_mister_20260909.rbf"), []byte("x"), 0644)
	writeMRA(t, h.card, "_Arcade/Black Heart (Coin-Op).mra", "Black Heart", "blkheart", "blkheart_mister")
	h.requestScan()
	finishBackground(t, h)
	if f := h.a.FavoriteSet(); !f["blkheart"] || f["black-heart"] || f["local:blkheart"] {
		t.Fatalf("the star went to %v, not the implementation that runs", f)
	}
	if vs := h.alternatives(&h.a.Data().Rows[0]); len(vs) != 1 || vs[0] != "_Arcade/_Extra/Black Heart (other).mra" {
		t.Fatalf("Coin-Op's row does not offer the former standin: %v", vs)
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
