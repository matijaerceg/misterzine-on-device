//go:build linux

package mister

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

// InputProbe is a short-lived, read-only evdev observer. In particular it does
// not use EVIOCGRAB and does not feed raw events into normal app navigation.
// It opens nodes even when they lack KEY_ENTER, KEY_UP or BTN_START.
type InputProbe struct {
	files []*os.File
	wg    sync.WaitGroup
	once  sync.Once
	stop  chan struct{}
}

func OpenInputProbe(c *support.Capture, until time.Time) *InputProbe {
	p := &InputProbe{stop: make(chan struct{})}
	paths, _ := filepath.Glob("/dev/input/event*")
	for _, path := range paths {
		f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOCTTY|syscall.O_CLOEXEC|syscall.O_NONBLOCK, 0)
		d := support.Device{Node: filepath.Base(path)}
		if err != nil {
			d.Error = err.Error()
			c.AddDevice(d)
			continue
		}
		d.Name = devName(f)
		if d.Name == "misterzine launcher" {
			f.Close()
			continue
		}
		d.Keyboard, d.StandardStart = isKeyboard(f), isPad(f)
		d.Virtual = d.Name == "MiSTer virtual input"
		if !d.Virtual {
			m := deviceStart(f)
			d.StartCode, d.StartMapping, d.StartNote = m.Code, m.Source, m.Note
		}
		var id [4]uint16
		if err := ioctl(f.Fd(), 0x80084502, unsafe.Pointer(&id[0])); err == nil { // EVIOCGID
			d.Bus, d.Vendor, d.Product, d.Version = id[0], id[1], id[2], id[3]
		}
		if !c.AddDevice(d) {
			f.Close()
			continue
		}
		p.files = append(p.files, f)
		p.wg.Add(1)
		go func(f *os.File, node string) {
			defer p.wg.Done()
			if err := readProbe(f, node, c, p.stop); err != nil {
				c.DeviceError(node, err.Error())
			}
		}(f, d.Node)
	}
	// Bound the observer lifetime even if an external updater changes the UI
	// screen before its normal completion tick can run.
	go p.closeAt(until)
	return p
}

func (p *InputProbe) closeAt(until time.Time) {
	t := time.NewTimer(time.Until(until))
	defer t.Stop()
	select {
	case <-p.stop:
	case <-t.C:
		p.Close()
	}
}

func (p *InputProbe) Close() {
	p.once.Do(func() {
		close(p.stop)
		for _, f := range p.files {
			f.Close()
		}
		p.wg.Wait()
	})
}

// Native Linux uses 16-byte events on MiSTer's 32-bit ARM, 24 on 64-bit hosts.
// Use the native timeval size so the diagnostic also has real Linux host tests.
func readProbe(r io.Reader, node string, c *support.Capture, stop <-chan struct{}) error {
	offset := int(unsafe.Sizeof(syscall.Timeval{}))
	size := offset + 8
	buf := make([]byte, size*64)
	pending := 0
	dropped := false
	for {
		select {
		case <-stop:
			return nil
		default:
		}
		n, err := r.Read(buf[pending:])
		if n > 0 {
			n += pending
			used := 0
			for used+size <= n {
				e := buf[used+offset : used+size]
				typ, code := binary.LittleEndian.Uint16(e), binary.LittleEndian.Uint16(e[2:])
				value := int32(binary.LittleEndian.Uint32(e[4:]))
				used += size
				if typ == 0 && code == 3 {
					c.Record(node, typ, code, value, time.Now())
					dropped = true
					continue
				}
				if dropped {
					if typ == 0 && code == 0 {
						dropped = false
					}
					continue
				}
				c.Record(node, typ, code, value, time.Now())
			}
			pending = copy(buf, buf[used:n])
		}
		if err != nil {
			select {
			case <-stop:
				return nil
			default:
			}
			if errors.Is(err, syscall.EAGAIN) {
				time.Sleep(5 * time.Millisecond)
				continue
			}
			return err
		}
		if n == 0 {
			return io.EOF
		}
	}
}
