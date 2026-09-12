//go:build linux

package mister

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// The menu launcher needs two things Main only offers through the keyboard:
// F9 opens the framebuffer console, and the console has to be on tty2 for
// Main to ignore OSD keys while a program draws (menu.cpp: "prevent OSD
// control while script is executing on framebuffer"). A virtual keyboard
// through /dev/uinput provides the key; VT ioctls do the switching.

const (
	uiSetEvbit   = 0x40045564
	uiSetKeybit  = 0x40045565
	uiDevCreate  = 0x5501
	uiDevDestroy = 0x5502
	vtActivate   = 0x5606
	evSyn        = 0
	evKeyType    = 1

	// KeyF9 and KeyF12 are Main's console open and OSD/console close keys.
	KeyF9  = 67
	KeyF12 = 88
)

// VKeyboard is a uinput keyboard Main will read like any other.
type VKeyboard struct {
	f *os.File
}

// NewVKeyboard creates the device. Main opens it on hotplug; give it a
// moment before the first key.
func NewVKeyboard(name string) (*VKeyboard, error) {
	f, err := os.OpenFile("/dev/uinput", os.O_WRONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open /dev/uinput: %w", err)
	}
	set := func(req uintptr, v int) error {
		_, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), req, uintptr(v))
		if e != 0 {
			return e
		}
		return nil
	}
	if err := set(uiSetEvbit, evKeyType); err != nil {
		f.Close()
		return nil, fmt.Errorf("UI_SET_EVBIT: %w", err)
	}
	if err := set(uiSetEvbit, evSyn); err != nil {
		f.Close()
		return nil, fmt.Errorf("UI_SET_EVBIT syn: %w", err)
	}
	for _, k := range []int{KeyF9, KeyF12, keyEnter, keyEsc, keyUp, keyDown, keyLeft, keyRight} {
		if err := set(uiSetKeybit, k); err != nil {
			f.Close()
			return nil, fmt.Errorf("UI_SET_KEYBIT %d: %w", k, err)
		}
	}
	// legacy struct uinput_user_dev: name[80], input_id{4 x u16}, ff_effects_max u32, 4 x [64]s32
	var dev [1116]byte
	copy(dev[:79], name)
	binary.LittleEndian.PutUint16(dev[80:], 0x03) // BUS_USB
	binary.LittleEndian.PutUint16(dev[82:], 0x1209)
	binary.LittleEndian.PutUint16(dev[84:], 0x5a1e)
	binary.LittleEndian.PutUint16(dev[86:], 1)
	if _, err := f.Write(dev[:]); err != nil {
		f.Close()
		return nil, fmt.Errorf("uinput setup: %w", err)
	}
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uiDevCreate, 0); e != 0 {
		f.Close()
		return nil, fmt.Errorf("UI_DEV_CREATE: %v", e)
	}
	return &VKeyboard{f: f}, nil
}

func (k *VKeyboard) emit(typ, code uint16, value int32) error {
	var ev [16]byte
	now := time.Now()
	binary.LittleEndian.PutUint32(ev[0:], uint32(now.Unix()))
	binary.LittleEndian.PutUint32(ev[4:], uint32(now.Nanosecond()/1000))
	binary.LittleEndian.PutUint16(ev[8:], typ)
	binary.LittleEndian.PutUint16(ev[10:], code)
	binary.LittleEndian.PutUint32(ev[12:], uint32(value))
	_, err := k.f.Write(ev[:])
	return err
}

// Press taps a key: down, sync, up, sync.
func (k *VKeyboard) Press(code uint16) error {
	for _, step := range []struct {
		t, c uint16
		v    int32
	}{{evKeyType, code, 1}, {evSyn, 0, 0}, {evKeyType, code, 0}, {evSyn, 0, 0}} {
		if err := k.emit(step.t, step.c, step.v); err != nil {
			return err
		}
		time.Sleep(20 * time.Millisecond)
	}
	return nil
}

// Close destroys the device.
func (k *VKeyboard) Close() error {
	syscall.Syscall(syscall.SYS_IOCTL, k.f.Fd(), uiDevDestroy, 0)
	return k.f.Close()
}

// ActiveTTY reports the active virtual console, e.g. "tty1".
func ActiveTTY() string {
	b, err := os.ReadFile("/sys/class/tty/tty0/active")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// Chvt switches to console n and waits up to two seconds for the switch.
// The kernel drops the request while the front console is in graphics mode
// with nobody owning VT switching, which is how a frontend such as Degauss
// leaves tty2 while it draws; VT_WAITACTIVE would then never return, so
// the wait is a bounded poll of the active console and the failure is
// reported instead.
func Chvt(n int) error {
	f, err := os.OpenFile("/dev/tty0", os.O_RDWR|syscall.O_NOCTTY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), vtActivate, uintptr(n)); e != 0 {
		return fmt.Errorf("VT_ACTIVATE %d: %v", n, e)
	}
	want := fmt.Sprintf("tty%d", n)
	if !awaitConsole(want, ActiveTTY, time.Sleep, 2*time.Second) {
		return fmt.Errorf("console did not switch to %s in 2s (still %s: another program holds the screen?)", want, ActiveTTY())
	}
	return nil
}

var _ = unsafe.Pointer(nil)
