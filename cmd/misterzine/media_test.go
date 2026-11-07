package main

import (
	"slices"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/images"
)

func TestPrefetchIncludesFlyersButNeverVideos(t *testing.T) {
	ds := data.Ingest([]data.Row{{Base: "Arcade", K: "demo", MediaKey: "demo", MediaSlots: []string{"flyer", "video"}}}, "", time.Now())
	pics := picsFor(ds)
	if len(pics) != 1 || pics[0].Slot != "flyer" || pics[0].Key != "demo" {
		t.Fatalf("unexpected background media downloads: %+v", pics)
	}
}

func TestPrefetchLocalArtwork(t *testing.T) {
	row := data.Row{Base: "Arcade", K: "local:neogeo:mslug:fixture", Img: "mslug", ImgSlots: []string{"ltitle", "lsnap"}}
	pics := picsFor(data.Ingest([]data.Row{row}, "", time.Now()))
	want := []images.Pic{{Key: "mslug", Slot: "lsnap"}, {Key: "mslug", Slot: "ltitle"}}
	if !slices.Equal(pics, want) {
		t.Fatalf("prefetch order: got %+v, want %+v", pics, want)
	}
}
