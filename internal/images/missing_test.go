package images

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMissingTTL(t *testing.T) {
	dir := t.TempDir()
	lg := log.New(io.Discard, "", 0)
	// Version 1: a flat list. Every entry is dated now on load.
	os.WriteFile(filepath.Join(dir, "missing.json"), []byte(`["snap/old","lsnap/orphan"]`), 0644)
	s := New(dir, nil, lg, 1<<20)
	if !s.isMissing(Pic{Key: "old", Slot: "snap"}) || !s.isMissing(Pic{Key: "orphan", Slot: "lsnap"}) {
		t.Fatal("v1 entries not loaded")
	}
	if _, err := os.Stat(filepath.Join(dir, "lsnap")); err != nil {
		t.Fatal("lsnap folder not created")
	}
	s.Close()
	b, err := os.ReadFile(filepath.Join(dir, "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	var mf missingFile
	if json.Unmarshal(b, &mf) != nil || mf.V != 2 || len(mf.Missing) != 2 {
		t.Fatalf("v2 not written: %s", b)
	}

	// Version 2 with one stale and one fresh entry: the stale one is retried.
	now := time.Now()
	mf = missingFile{V: 2, Missing: map[string]int64{
		"lsnap/stale": now.Add(-missingTTL - time.Hour).Unix(),
		"lsnap/fresh": now.Add(-time.Hour).Unix(),
	}}
	b, _ = json.Marshal(mf)
	os.WriteFile(filepath.Join(dir, "missing.json"), b, 0644)
	s = New(dir, nil, lg, 1<<20)
	if s.isMissing(Pic{Key: "stale", Slot: "lsnap"}) || !s.isMissing(Pic{Key: "fresh", Slot: "lsnap"}) {
		t.Fatal("expiry not applied")
	}
	s.Close()
	b, _ = os.ReadFile(filepath.Join(dir, "missing.json"))
	mf = missingFile{} // Unmarshal merges into an existing map
	if json.Unmarshal(b, &mf) != nil || len(mf.Missing) != 1 || mf.Missing["lsnap/fresh"] != now.Add(-time.Hour).Unix() {
		t.Fatalf("stale entry kept or fresh stamp changed: %s", b)
	}

	// Nothing missing: the file goes away rather than lingering empty.
	s = New(dir, nil, lg, 1<<20)
	s.ClearCache()
	s.Close()
	if _, err := os.Stat(filepath.Join(dir, "missing.json")); !os.IsNotExist(err) {
		t.Fatal("empty missing.json left behind")
	}

	// Unreadable content is ignored, not fatal.
	os.WriteFile(filepath.Join(dir, "missing.json"), []byte(`{"v":9,"x":1}`), 0644)
	s = New(dir, nil, lg, 1<<20)
	if len(s.missing) != 0 {
		t.Fatal("garbage loaded")
	}
	s.Close()
}
