//go:build linux

package mister

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

func probeEvent(typ, code uint16, value int32) []byte {
	offset := int(unsafe.Sizeof(syscall.Timeval{}))
	b := make([]byte, offset+8)
	binary.LittleEndian.PutUint16(b[offset:], typ)
	binary.LittleEndian.PutUint16(b[offset+2:], code)
	binary.LittleEndian.PutUint32(b[offset+4:], uint32(value))
	return b
}

type shortProbeReader struct{ io.Reader }

func (r shortProbeReader) Read(b []byte) (int, error) { return r.Reader.Read(b[:min(5, len(b))]) }

func TestProbeReadsNonstandardButtonsAndDroppedBatches(t *testing.T) {
	var raw []byte
	for _, e := range []struct {
		typ, code uint16
		value     int32
	}{
		{1, 299, 1}, {1, 299, 0}, {0, 3, 0}, {1, 315, 1}, {0, 0, 0}, {3, 16, -1},
	} {
		raw = append(raw, probeEvent(e.typ, e.code, e.value)...)
	}
	c := support.NewCapture(support.Report{}, time.Now().Add(-time.Second), time.Now().Add(time.Second))
	err := readProbe(shortProbeReader{bytes.NewReader(raw)}, "arcade", c, make(chan struct{}))
	if err != io.EOF {
		t.Fatal(err)
	}
	r := c.Snapshot()
	if len(r.Signals) != 2 || r.Signals[0].Code != 299 || r.Signals[0].Down != 1 || r.Signals[0].Up != 1 || r.Dropped != 1 || r.Signals[1].Value != -1 {
		t.Fatalf("raw evidence incorrectly filtered/decoded: %+v", r)
	}
}

func TestProbeStopsBeforeAnotherRead(t *testing.T) {
	stop := make(chan struct{})
	close(stop)
	c := support.NewCapture(support.Report{}, time.Time{}, time.Now())
	if err := readProbe(nil, "test", c, stop); err != nil {
		t.Fatal(err)
	}
}

func TestProbeCloseUnblocksIdleReader(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	p := &InputProbe{files: []*os.File{r}, stop: make(chan struct{})}
	c := support.NewCapture(support.Report{}, time.Now(), time.Now().Add(time.Second))
	p.wg.Add(1)
	go func() { defer p.wg.Done(); readProbe(r, "test", c, p.stop) }()
	done := make(chan struct{})
	go func() { p.Close(); p.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("closing probe left an idle reader blocked")
	}
}

func TestProbeHasIndependentDeadline(t *testing.T) {
	p := &InputProbe{stop: make(chan struct{})}
	go p.closeAt(time.Now().Add(10 * time.Millisecond))
	select {
	case <-p.stop:
	case <-time.After(time.Second):
		t.Fatal("observer outlived test deadline")
	}
}
