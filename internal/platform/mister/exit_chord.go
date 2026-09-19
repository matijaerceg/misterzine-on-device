//go:build linux

package mister

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"
	"unsafe"
)

// While a core runs, Main holds every input device exclusively (EVIOCGRAB),
// so nothing else receives its events: not another evdev reader, and not
// the joystick node either, since the kernel's grab is per device. The
// pressed-key bitmap a device keeps is another matter: the kernel updates
// it before it decides who gets the event, and EVIOCGKEY reads it from any
// open descriptor. The exit chord is therefore not read as events but
// polled as state, which is what MiSTer_SAM's controller check amounts to.
// Buttons only: a Select or Start defined on an axis edge is not in the
// bitmap, and such a pad has no chord.

// eviocgkey is EVIOCGKEY(96): the key bitmap for codes 0..767.
const eviocgkey = uintptr(0x80000000) | uintptr(96)<<16 | uintptr(0x45)<<8 | 0x18

// pressedKeys reads the device's current key bitmap.
func pressedKeys(f *os.File) ([96]byte, error) {
	var bits [96]byte
	err := ioctl(f.Fd(), eviocgkey, unsafe.Pointer(&bits[0]))
	return bits, err
}

// exitChordCodes are the button codes the variant needs on this pad, from
// its MiSTer define (or the Linux layout of a pad MiSTer has not defined),
// or nil when the pad lacks one of them or has it on an axis.
func exitChordCodes(m padMapping, variant string) []uint16 {
	slots := ExitChordSlots(variant)
	if slots == nil {
		return nil
	}
	codes := make([]uint16, 0, len(slots))
	for _, s := range slots {
		code, ok := m.Slots[s]
		if !ok || code < 256 || code >= emuBase {
			return nil
		}
		codes = append(codes, code)
	}
	return codes
}

type chordPad struct {
	f     *os.File
	name  string
	codes []uint16
	hold  chordHold
}

func (p *chordPad) allDown() bool {
	bits, err := pressedKeys(p.f)
	if err != nil {
		return false
	}
	for _, c := range p.codes {
		if bits[c/8]&(1<<(c%8)) == 0 {
			return false
		}
	}
	return true
}

// WatchExitChord polls the connected pads for the chord until stop closes
// and calls fire the first time it is held for ExitChordHold while armed
// says the moment is right (a game is on the screen). Pads that come and
// go are picked up every couple of seconds. It returns once stopped.
func WatchExitChord(stop <-chan struct{}, variant string, armed func() bool, fire func(), lg *log.Logger) {
	if ExitChordSlots(variant) == nil {
		return
	}
	pads := map[string]*chordPad{}
	defer func() {
		for _, p := range pads {
			p.f.Close()
		}
	}()
	rescan := func() {
		paths, _ := filepath.Glob("/dev/input/event*")
		sort.Strings(paths)
		seen := map[string]bool{}
		for _, path := range paths {
			seen[path] = true
			if _, have := pads[path]; have {
				continue
			}
			f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOCTTY|syscall.O_CLOEXEC|syscall.O_NONBLOCK, 0)
			if err != nil {
				continue
			}
			if isKeyboard(f) {
				f.Close()
				continue
			}
			codes := exitChordCodes(devicePad(f), variant)
			if codes == nil {
				f.Close()
				continue
			}
			pads[path] = &chordPad{f: f, name: devName(f), codes: codes, hold: chordHold{hold: ExitChordHold}}
			lg.Printf("exit chord: %s on %s (%s): buttons %v", ExitChordName(variant), path, pads[path].name, codes)
		}
		for path, p := range pads {
			if !seen[path] {
				p.f.Close()
				delete(pads, path)
			}
		}
	}
	rescan()
	poll := time.NewTicker(40 * time.Millisecond)
	defer poll.Stop()
	scan := time.NewTicker(2 * time.Second)
	defer scan.Stop()
	fired := false
	for {
		select {
		case <-stop:
			return
		case <-scan.C:
			rescan()
		case now := <-poll.C:
			if fired {
				continue
			}
			for _, p := range pads {
				down := p.allDown()
				if p.hold.update(down, now) && armed() {
					lg.Printf("exit chord: held on %s; leaving the game", p.name)
					fired = true
					fire()
					break
				}
			}
		}
	}
}
