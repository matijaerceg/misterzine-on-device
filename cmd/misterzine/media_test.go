package main

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

func TestPrefetchIncludesFlyersButNeverVideos(t *testing.T) {
	ds := data.Ingest([]data.Row{{Base: "Arcade", K: "demo", MediaKey: "demo", MediaSlots: []string{"flyer", "video"}}}, "", time.Now())
	pics := picsFor(ds)
	if len(pics) != 1 || pics[0].Slot != "flyer" || pics[0].Key != "demo" {
		t.Fatalf("unexpected background media downloads: %+v", pics)
	}
}
