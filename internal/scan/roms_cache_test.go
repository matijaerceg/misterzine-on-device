package scan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// countingCheck is a ROMCheck over card whose real check counts its calls.
func countingCheck(t *testing.T, card, cachePath string) (*ROMCheck, func() int) {
	t.Helper()
	c := NewROMCheck(card)
	c.CachePath = cachePath
	inner := c.run
	var mu sync.Mutex
	calls := 0
	c.run = func(rel string, acc *CoreAccess) ROMResult {
		mu.Lock()
		calls++
		mu.Unlock()
		return inner(rel, acc)
	}
	return c, func() int {
		mu.Lock()
		defer mu.Unlock()
		return calls
	}
}

func runSweep(t *testing.T, c *ROMCheck, paths []string) (n, reused int) {
	t.Helper()
	done := make(chan [2]int, 1)
	c.Sweep(paths, func(n, reused int, _ time.Duration) { done <- [2]int{n, reused} })
	select {
	case r := <-done:
		return r[0], r[1]
	case <-time.After(5 * time.Second):
		t.Fatal("sweep never finished")
		return 0, 0
	}
}

// A sweep keeps its verdicts on the card, and the next run's sweep reuses
// them while the MRA and the zips it names are as they were: a zip that
// appears, an MRA that changes, a verdict that asked the core's beta
// status, or a cache written for another ROM root is checked afresh.
func TestROMSweepReusesVerdicts(t *testing.T) {
	card := filepath.Join(t.TempDir(), "fat")
	mra := func(body string) string { return `<misterromdescription><rbf>core</rbf>` + body + `</misterromdescription>` }
	putFile(t, filepath.Join(card, "_Arcade/ok.mra"), mra(`<rom index="0" zip="ok.zip" md5="None"><part name="a" crc="`+crcOf("a")+`"/></rom>`))
	putFile(t, filepath.Join(card, "_Arcade/bad.mra"), mra(`<rom index="0" zip="gone.zip" md5="None"><part name="a" crc="`+crcOf("a")+`"/></rom>`))
	putFile(t, filepath.Join(card, "_Arcade/key.mra"), mra(`<rom index="0" zip="ok.zip" md5="None"><part name="a" crc="`+crcOf("a")+`"/></rom><rom index="17" zip="jtbeta.zip" md5="None"><part name="beta.bin"/></rom>`))
	putZip(t, filepath.Join(card, "games/mame/ok.zip"), member{name: "a", data: "a"})
	cachePath := filepath.Join(t.TempDir(), "roms.json")
	paths := []string{"_Arcade/ok.mra", "_Arcade/bad.mra", "_Arcade/key.mra", "_Arcade/cores/core.rbf"}

	c, calls := countingCheck(t, card, cachePath)
	if n, reused := runSweep(t, c, paths); n != 3 || reused != 0 || calls() != 3 {
		t.Fatalf("first sweep: %d answered, %d reused, %d checks", n, reused, calls())
	}
	if r, ok := c.Known("_Arcade/bad.mra"); !ok || r.Text != "Missing game ROM: gone.zip" || !r.Block {
		t.Fatalf("bad.mra: %v %+v", ok, r)
	}
	var f romCacheFile
	if b, err := os.ReadFile(cachePath); err != nil || json.Unmarshal(b, &f) != nil {
		t.Fatalf("cache file: %v", err)
	}
	if len(f.MRAs) != 3 || !f.MRAs["_Arcade/key.mra"].Key || f.MRAs["_Arcade/ok.mra"].Key {
		t.Fatalf("cache entries: %+v", f.MRAs)
	}
	if z := f.MRAs["_Arcade/bad.mra"].Zips; len(z) != 1 || z[0].Size != -1 {
		t.Fatalf("an absent zip must be stamped absent: %+v", z)
	}

	// the next run: ok and bad come from the file, key is asked again
	c, calls = countingCheck(t, card, cachePath)
	if n, reused := runSweep(t, c, paths); n != 3 || reused != 2 || calls() != 1 {
		t.Fatalf("second sweep: %d answered, %d reused, %d checks", n, reused, calls())
	}
	if r, ok := c.Known("_Arcade/bad.mra"); !ok || !r.Block {
		t.Fatalf("a reused verdict must be known: %v %+v", ok, r)
	}
	if r, ok := c.Known("_Arcade/ok.mra"); !ok || r != (ROMResult{}) {
		t.Fatalf("ok.mra: %v %+v", ok, r)
	}
	select {
	case <-c.Ready():
	case <-time.After(time.Second):
		t.Fatal("reused verdicts must still signal the list once")
	}

	// the missing zip appears: only bad.mra is read again, and its verdict changes
	putZip(t, filepath.Join(card, "games/mame/gone.zip"), member{name: "a", data: "a"})
	c, calls = countingCheck(t, card, cachePath)
	if n, reused := runSweep(t, c, paths); n != 3 || reused != 1 || calls() != 2 {
		t.Fatalf("after a zip appeared: %d answered, %d reused, %d checks", n, reused, calls())
	}
	if r, ok := c.Known("_Arcade/bad.mra"); !ok || r != (ROMResult{}) {
		t.Fatalf("bad.mra with its zip: %v %+v", ok, r)
	}

	// the MRA changes: read again
	putFile(t, filepath.Join(card, "_Arcade/ok.mra"), mra(`<rom index="0" zip="other.zip" md5="None"><part name="a" crc="`+crcOf("a")+`"/></rom>`))
	c, calls = countingCheck(t, card, cachePath)
	if n, reused := runSweep(t, c, paths); n != 3 || reused != 1 || calls() != 2 {
		t.Fatalf("after an MRA changed: %d answered, %d reused, %d checks", n, reused, calls())
	}
	if r, ok := c.Known("_Arcade/ok.mra"); !ok || r.Text != "Missing game ROM: other.zip" {
		t.Fatalf("changed ok.mra: %v %+v", ok, r)
	}

	// an MRA that left the card drops out of the file
	c, _ = countingCheck(t, card, cachePath)
	runSweep(t, c, paths[1:])
	f = romCacheFile{}
	if b, err := os.ReadFile(cachePath); err != nil || json.Unmarshal(b, &f) != nil {
		t.Fatalf("cache file: %v", err)
	}
	if _, ok := f.MRAs["_Arcade/ok.mra"]; ok || len(f.MRAs) != 2 {
		t.Fatalf("a path no longer swept must drop out: %+v", f.MRAs)
	}

	// a file written for another ROM root says nothing about this one
	f.Root = filepath.Join(card, "elsewhere")
	b, _ := json.Marshal(f)
	if err := os.WriteFile(cachePath, b, 0644); err != nil {
		t.Fatal(err)
	}
	c, calls = countingCheck(t, card, cachePath)
	if n, reused := runSweep(t, c, paths); n != 3 || reused != 0 || calls() != 3 {
		t.Fatalf("another root's file: %d answered, %d reused, %d checks", n, reused, calls())
	}

	// without a path nothing is kept, and a replaced check (no parse) leaves no stamps
	c = NewROMCheck(card)
	c.run = func(string, *CoreAccess) ROMResult { return ROMResult{} }
	if n, reused := runSweep(t, c, paths); n != 3 || reused != 0 {
		t.Fatalf("no cache path: %d answered, %d reused", n, reused)
	}
}

// SetPaused holds the sweep between two checks and lets it go on; a held
// sweep still gives way to a new one, and a nil checker takes it quietly.
func TestROMSweepPauses(t *testing.T) {
	c := NewROMCheck(t.TempDir())
	var mu sync.Mutex
	checked := 0
	c.run = func(string, *CoreAccess) ROMResult {
		mu.Lock()
		checked++
		mu.Unlock()
		return ROMResult{}
	}
	c.SetPaused(true)
	done := make(chan int, 1)
	c.Sweep([]string{"a.mra", "b.mra"}, func(n, _ int, _ time.Duration) { done <- n })
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	n := checked
	mu.Unlock()
	if n != 0 {
		t.Fatalf("%d checks ran while paused", n)
	}
	select {
	case <-done:
		t.Fatal("the sweep finished while paused")
	default:
	}
	c.SetPaused(false)
	select {
	case n := <-done:
		if n != 2 {
			t.Fatalf("answered %d, want 2", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the sweep never resumed")
	}

	c.SetPaused(true)
	first := make(chan int, 1)
	c.Sweep([]string{"c.mra"}, func(n, _ int, _ time.Duration) { first <- n })
	second := make(chan int, 1)
	c.Sweep([]string{"d.mra"}, func(n, _ int, _ time.Duration) { second <- n })
	c.SetPaused(false)
	select {
	case <-second:
	case <-time.After(5 * time.Second):
		t.Fatal("the second sweep never finished")
	}
	select {
	case <-first:
		t.Fatal("a held sweep ran on after a new one began")
	case <-time.After(200 * time.Millisecond):
	}
	var nilc *ROMCheck
	nilc.SetPaused(true)
	nilc.SetPaused(false)
}
