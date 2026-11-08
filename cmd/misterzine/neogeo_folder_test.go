//go:build linux

package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/store"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/mgl"
)

func putCartFile(t *testing.T, card, p, content string) {
	t.Helper()
	target := filepath.Join(card, p)
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func putPack(t *testing.T, card, name string, files ...string) {
	t.Helper()
	target := filepath.Join(card, name)
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(target)
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	for _, name := range files {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("test fixture")); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func folderGames(h *host) []string {
	var targets []string
	for _, r := range h.a.Data().Rows {
		if r.IsNeoGeoFile() {
			targets = append(targets, r.MRA)
		}
	}
	slices.Sort(targets)
	return targets
}

func TestNeoGeoFolderScanAndLaunch(t *testing.T) {
	h := backgroundHost(t)
	const dir = "games/NEOGEO/"
	const pack = "Packs/Player's \"Arcade\" & friends.zip"
	const bout = "1 World A-Z/3 Count Bout (3countb).neo"
	const slug = "1 World A-Z/mslug.neo"
	const dark = "Darksoft/garou"
	for _, name := range []string{dir + "mslug.neo", dir + "Alternate/mslug.neo", dir + "aof/track01.cue", dir + "unknown.neo", dir + "lasthope.neo", dir + "zintrckb.neo", dir + "neocd.bin", dir + "3countb.chd", dir + "kof98/prom", dir + "kof98/crom0", "games/NeoGeo-CD/garou.neo", "_Console/NeoGeo_20261101.rbf"} {
		putCartFile(t, h.card, name, "fixture")
	}
	putPack(t, h.card, dir+pack, bout, slug, dark+"/prom", dark+"/crom0", "zintrckb.neo", "kof97.chd", "nested.zip", "not-a-game.neo")
	putPack(t, h.card, dir+"mslug2.zip", "prom", "crom0", "srom")
	putPack(t, h.card, dir+"aof.zip", "disc.cue", "track01.bin")
	writeNeoGeoShortcut(t, h.card, "_Favorites/My Slug.mgl", "_Console/NeoGeo")
	putCartFile(t, h.card, dir+"extra/garou.neo", "fixture")
	putCartFile(t, h.card, "_Favorites/Custom.mgl", `<mistergamedescription><rbf>_Console/NeoGeo</rbf><setname>custom</setname><file root="games" type="f" index="1" path="NEOGEO/extra/garou.neo"/></mistergamedescription>`)

	scanOnce(t, h, catalogueRows(), "cat")
	if got := folderGames(h); len(got) != 0 {
		t.Fatalf("new installation imported %v", got)
	}
	if _, err := h.prepareLaunch(dir + pack + "/" + bout); err == nil {
		t.Fatal("opted-out launch")
	}
	h.setScanNeoGeoFolder(true)
	finishBackground(t, h)
	want := []string{dir + "Alternate/mslug.neo", dir + "kof98", dir + "mslug2.zip", dir + pack + "/" + bout, dir + pack + "/" + slug, dir + pack + "/" + dark}
	slices.Sort(want)
	if got := folderGames(h); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	for i, r := range h.a.Data().Rows {
		if !r.IsNeoGeoFile() {
			continue
		}
		if h.status[i] != data.StatusFoundUndated {
			t.Fatalf("status for %s: %v", r.MRA, h.status[i])
		}
		file, err := h.prepareLaunch(r.MRA)
		if err != nil {
			t.Fatalf("%s: %v", r.MRA, err)
		}
		if file != filepath.Join(h.root, "cache", "neogeo-launch.mgl") {
			t.Fatal(file)
		}
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "&#") {
			t.Fatalf("MiSTer cannot decode numeric XML entities: %s", b)
		}
		d, err := mgl.Parse(strings.NewReader(string(b)))
		if err != nil {
			t.Fatal(err)
		}
		if ok, _ := d.Cartridge(); !ok || d.Files[0].Delay != "1" || d.RBF != "_Console/NeoGeo" || d.Files[0].Path != filepath.Join(h.card, r.MRA) {
			t.Fatalf("wrong launch for %s: %+v", r.MRA, d)
		}
	}
	cached, _, _ := h.withSavedLocalRows(catalogueRows(), "cat")
	if !slices.ContainsFunc(cached, func(r data.Row) bool { return r.IsNeoGeoFile() }) {
		t.Fatal("missing opted-in cache")
	}
	off, _, _ := restart(t, h, catalogueRows(), "cat", nil)
	if len(folderGames(off)) != 0 {
		t.Fatal("old cache bypassed opt-in")
	}

	putPack(t, h.card, dir+pack, slug)
	if _, err := h.prepareLaunch(dir + pack + "/" + bout); err == nil {
		t.Fatal("launched removed ZIP member")
	}
	if _, err := h.prepareLaunch(dir + pack + "/" + slug); err != nil {
		t.Fatal(err)
	}
	putCartFile(t, h.card, "_Arcade/NeoGeo_20261101.rbf", "fixture")
	if err := os.Remove(filepath.Join(h.card, "_Console/NeoGeo_20261101.rbf")); err != nil {
		t.Fatal(err)
	}
	h.requestScan()
	finishBackground(t, h)
	if h.index.NeoGeoRBF != "_Arcade/NeoGeo" {
		t.Fatal(h.index.NeoGeoRBF)
	}
	if _, err := h.prepareLaunch(dir + pack + "/" + slug); err != nil {
		t.Fatal(err)
	}
	putCartFile(t, h.card, "_Console/NeoGeo_20261101.rbf", "fixture")
	previous := h.a.Data()
	status := map[string]data.Status{}
	for i, r := range previous.Rows {
		status[r.K] = h.status[i]
	}
	// Existing non-folder versions and ownership come from the completed scan.
	k := previous.Rows[0].K
	h.alts = map[string][]string{k: {"_Arcade/versions/Existing.mra"}}
	h.owners = map[string]string{k: "another-source"}
	h.requestScan() // toggle while the previous scan is in flight
	h.setScanNeoGeoFolder(false)
	if len(folderGames(h)) != 0 {
		t.Fatal("opt-out left games in the list")
	}
	if len(h.status) != len(h.a.Data().Rows) || h.owners[k] != "another-source" || len(h.alternatives(&h.a.Data().Rows[0])) != 1 {
		t.Fatal("opt-out discarded existing games' scan results")
	}
	for i, r := range h.a.Data().Rows {
		if h.status[i] != status[r.K] {
			t.Fatalf("opt-out changed the card status for %s", r.K)
		}
	}
	finishBackground(t, h)
	if len(folderGames(h)) != 0 {
		t.Fatal("old scan resurrected folder games")
	}
	if _, err := h.prepareLaunch(dir + pack + "/" + slug); err == nil {
		t.Fatal("old target launched after opt-out")
	}
	if _, err := h.prepareLaunch("_Favorites/My Slug.mgl"); err != nil {
		t.Fatal(err)
	}
}

func TestNeoGeoFolderZIPErrorIsIsolatedAndOnlyReadOnOptIn(t *testing.T) {
	h := backgroundHost(t)
	putCartFile(t, h.card, "games/NEOGEO/damaged.zip", "not a zip")
	putCartFile(t, h.card, "games/NEOGEO/3countb.neo", "fixture")
	writeNeoGeoShortcut(t, h.card, "_Favorites/Slug.mgl", "_Console/NeoGeo")
	for _, on := range []bool{false, true} {
		result, menuErr, folderErr := h.scanNeoGeo(on)
		if menuErr != nil || (folderErr != nil) != on || len(result.Rows) != map[bool]int{false: 1, true: 2}[on] {
			t.Fatalf("opt-in %v: %v rows, menu %v, folder %v", on, len(result.Rows), menuErr, folderErr)
		}
	}
}

func TestNeoGeoUSBScanLaunchAndRemoval(t *testing.T) {
	h := backgroundHost(t)
	h.card = filepath.Join(t.TempDir(), "fat")
	const sd = "games/NEOGEO/"
	const usb = "../usb0/games/NEOGEO/"
	const member = "pack.zip/World/3countb.neo"
	putCartFile(t, h.card, "_Console/NeoGeo.rbf", "core")
	putCartFile(t, h.card, sd+"mslug.neo", "sd")
	putCartFile(t, h.card, usb+"mslug.neo", "usb")
	putCartFile(t, h.card, usb+"kof98.neo", "USB-only loose cartridge")
	putPack(t, h.card, usb+"pack.zip", "World/3countb.neo")
	writeNeoGeoShortcut(t, h.card, "_Favorites/Slug.mgl", "_Console/NeoGeo")
	// This SD-pinned shortcut must not hide the USB edition of the game.
	putCartFile(t, h.card, "_Favorites/Bout.mgl", `<mistergamedescription><rbf>_Console/NeoGeo</rbf><file type="f" index="1" path="`+filepath.Join(h.card, sd, member)+`"/></mistergamedescription>`)
	h.setScanNeoGeoFolder(true)
	finishBackground(t, h)
	if got := folderGames(h); !slices.Equal(got, []string{usb + "kof98.neo", usb + member}) {
		t.Fatalf("USB scan with relative and pinned shortcuts: %v", got)
	}
	cold := backgroundHost(t)
	cold.card, cold.root = h.card, h.root
	cold.settings.ScanNeoGeoFolder = true
	rows, _, _ := cold.withSavedLocalRows(catalogueRows(), "cat")
	for _, target := range []string{usb + member, usb + "kof98.neo"} {
		if !slices.ContainsFunc(rows, func(r data.Row) bool { return r.IsNeoGeoFile() && r.MRA == target }) {
			t.Fatalf("missing cached USB cartridge %s", target)
		}
		launch, err := cold.prepareLaunch(target)
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(launch)
		if err != nil {
			t.Fatal(err)
		}
		d, err := mgl.Parse(strings.NewReader(string(b)))
		if err != nil || len(d.Files) != 1 || d.Files[0].Path != filepath.Join(h.card, target) {
			t.Fatalf("USB launch file: %+v, %v", d, err)
		}
	}
	if backing, ok := mgl.GameBackingPath(usb + member); !ok || backing != usb+"pack.zip" {
		t.Fatalf("wrong file for USB availability: %q", backing)
	}
	// An empty folder at the USB root wins over USB/games and the SD.
	empty := filepath.Join(h.card, "../usb0/NEOGEO")
	if err := os.Mkdir(empty, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := cold.prepareLaunch(usb + member); err == nil {
		t.Fatal("cached launch ignored changed storage priority")
	}
	h.requestScan()
	finishBackground(t, h)
	if got := folderGames(h); len(got) != 0 {
		t.Fatalf("expected empty USB library, found %v", got)
	}
	putCartFile(t, h.card, sd+"garou.neo", "sd")
	if err := os.Rename(filepath.Join(h.card, "../usb0"), filepath.Join(h.card, "../unplugged")); err != nil {
		t.Fatal(err)
	}
	h.requestScan()
	finishBackground(t, h)
	if got := folderGames(h); !slices.Equal(got, []string{sd + "garou.neo"}) {
		t.Fatalf("expected SD library after rescan, found %v", got)
	}
}

// A returning player can press Start on a cached row before even the first
// core/status result arrives. Use the same core and ZIP member as a later launch.
func TestCachedNeoGeoLaunchBeforeFirstScan(t *testing.T) {
	h := backgroundHost(t)
	h.settings.ScanNeoGeoFolder = true
	const pack = "games/NEOGEO/Collection.zip"
	putPack(t, h.card, pack, "Arcade/3countb.neo")
	const core = "_Console/_SNK/NeoGeo_20261101.rbf"
	putCartFile(t, h.card, core, "fixture")
	scanOnce(t, h, catalogueRows(), "cat")

	cold := backgroundHost(t)
	cold.root, cold.card = h.root, h.card
	cold.settings.ScanNeoGeoFolder = true
	rows, _, _ := cold.withSavedLocalRows(catalogueRows(), "cat")
	pos := slices.IndexFunc(rows, func(r data.Row) bool { return r.IsNeoGeoFile() })
	if pos < 0 || cold.index != nil {
		t.Fatal("expected a cached cartridge before the first core scan")
	}
	target := rows[pos].MRA
	if target != pack+"/Arcade/3countb.neo" {
		t.Fatal(target)
	}
	launch, err := cold.prepareLaunch(target)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(launch)
	if err != nil {
		t.Fatal(err)
	}
	d, err := mgl.Parse(strings.NewReader(string(b)))
	if err != nil || d.RBF != "_Console/_SNK/NeoGeo" || d.Files[0].Path != filepath.Join(h.card, target) {
		t.Fatalf("early launch: %+v, %v", d, err)
	}
	if err := os.Remove(filepath.Join(h.card, core)); err != nil {
		t.Fatal(err)
	}
	if _, err := cold.prepareLaunch(target); err == nil {
		t.Fatal("early launch with missing core")
	}
}

func TestNeoGeoFolderPreferenceAndLinkedLibrary(t *testing.T) {
	h := backgroundHost(t)
	library := t.TempDir()
	putCartFile(t, library, "3countb.neo", "network fixture")
	putCartFile(t, h.card, "games/other.txt", "fixture")
	if err := os.Symlink(library, filepath.Join(h.card, "games", "NEOGEO")); err != nil {
		t.Fatal(err)
	}
	h.a = app.New(app.Config{PhysW: 320, PhysH: 240, NeoGeoSupport: true, Action: func(kind, arg string) {
		if kind == "scan-neogeo-folder" {
			h.setScanNeoGeoFolder(arg == "on")
		}
	}}, data.Ingest(catalogueRows(), "cat", time.Now()), nil)
	now := time.Now()
	tap := func(key platform.Key) {
		now = now.Add(time.Second)
		h.a.Handle(platform.Event{Key: key, Pressed: true, At: now})
		h.a.Handle(platform.Event{Key: key, At: now.Add(time.Millisecond)})
	}
	tap(platform.KeyBack)
	tap(platform.KeyPageDown)
	tap(platform.KeyRight)
	tap(platform.KeyPageDown)
	tap(platform.KeyUp)
	for _, on := range []bool{true, false} {
		if on {
			tap(platform.KeyRight)
		} else {
			tap(platform.KeyLeft)
		}
		finishBackground(t, h)
		h.saveAll(true)
		saved, err := store.LoadSettings(filepath.Join(h.root, "settings.json"))
		if err != nil || saved.ScanNeoGeoFolder != on || (len(folderGames(h)) > 0) != on {
			t.Fatalf("option %v, saved %v, games %v, error %v", on, saved.ScanNeoGeoFolder, folderGames(h), err)
		}
	}
	entries, err := os.ReadDir(library)
	if err != nil || len(entries) != 1 || entries[0].Name() != "3countb.neo" {
		t.Fatalf("library was modified: %v %v", entries, err)
	}
}
