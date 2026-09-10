package support

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestCaptureWindowAndUnmappedButton(t *testing.T) {
	from := time.Unix(100, 0)
	c := NewCapture(Report{}, from, from.Add(6*time.Second))
	c.AddDevice(Device{Node: "event2", Name: "Arcade"})
	c.Record("event2", 1, 28, 1, from.Add(-time.Second)) // opening confirm
	c.Record("event2", 1, 299, 1, from)
	c.Record("event2", 1, 299, 0, from.Add(time.Second))
	c.Record("event2", 1, 315, 1, from.Add(6*time.Second)) // after capture
	c.RecordStart(true, from.Add(6*time.Second))
	r := c.Snapshot()
	if len(r.Signals) != 1 || r.Signals[0].Code != 299 || r.Signals[0].Down != 1 || r.Signals[0].Up != 1 || r.StartDown != 0 {
		t.Fatalf("lost raw button or included navigation outside capture: %+v", r)
	}
	if r.Devices[0].Route() != "Skipped by normal input reader" || r.Conclusion() != "Button received; not recognised as Start." {
		t.Fatalf("wrong unsupported-controller result: %+v", r)
	}
	r.Devices[0].Name = "changed"
	r.Signals[0].Code = 315
	if got := c.Snapshot(); got.Devices[0].Name != "Arcade" || got.Signals[0].Code != 299 {
		t.Fatal("snapshot aliases live capture")
	}
}

func TestCaptureIsBoundedAndKeepsSourcesSeparate(t *testing.T) {
	now := time.Now()
	c := NewCapture(Report{}, now.Add(-time.Second), now.Add(time.Second))
	for i := 0; i < MaxDevices+5; i++ {
		c.AddDevice(Device{Node: fmt.Sprint(i)})
	}
	c.Record("raw", 1, 315, 1, now)
	c.Record("virtual", 1, 28, 1, now)
	for i := 0; i < MaxSignals+5; i++ {
		c.Record("raw", 1, uint16(i), 1, now)
	}
	r := c.Snapshot()
	if len(r.Devices) != MaxDevices || len(r.Signals) != MaxSignals || r.Dropped == 0 || r.Signals[0].Node == r.Signals[1].Node {
		t.Fatalf("unbounded or merged evidence: %+v", r)
	}
}

func TestStartAndMissingReleaseConclusions(t *testing.T) {
	now := time.Now()
	c := NewCapture(Report{}, now.Add(-time.Second), now.Add(time.Second))
	if !strings.HasPrefix(c.Snapshot().Conclusion(), "No button") {
		t.Fatal("invented input")
	}
	c.Record("keyboard", 1, 2, 0, now)
	if !strings.HasPrefix(c.Snapshot().Conclusion(), "Release received") {
		t.Fatal("release mistaken for movement")
	}
	c.RecordStart(true, now)
	if !strings.Contains(c.Snapshot().Conclusion(), "release not seen") {
		t.Fatal("missing release not reported")
	}
	c.RecordStart(false, now)
	if !strings.HasPrefix(c.Snapshot().Conclusion(), "Start recognised") {
		t.Fatal("normal Start not recognised")
	}
}
