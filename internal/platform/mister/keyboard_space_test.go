//go:build linux

package mister

import (
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// A keyboard's Space carries text and Main's virtual input re-types it
// without any, as it does a pad's Y; its letters are KeyOther there. The
// nodes of one USB receiver (keyboard, Consumer Control) share an identity,
// so a profile on one applies to the other.
func TestKeyboardSpaceTextAndSharedReceiverIdentity(t *testing.T) {
	kbd := &device{name: "Logitech USB Receiver", held: map[uint16]platform.Key{}}
	virtual := &device{name: "MiSTer virtual input", held: map[uint16]platform.Key{}}
	if kbd.keyboardText(keySpace) != ' ' || kbd.keyboardText(30) != 'a' {
		t.Fatalf("keyboard text: space %q a %q", kbd.keyboardText(keySpace), kbd.keyboardText(30))
	}
	if virtual.keyboardText(keySpace) != 0 || virtual.keyboardText(30) != 0 {
		t.Fatal("the virtual input types")
	}
	if k, _ := virtual.inputKey(keySpace); k != platform.KeySpace {
		t.Fatalf("virtual Space is %v", k)
	}
	if k, _ := virtual.inputKey(30); k != platform.KeyOther {
		t.Fatalf("virtual letter is %v", k)
	}
	keyboard, _ := controllerIdentity(0x046d, 0xc548, "", "usb-ffb40000.usb-1.2/input0")
	consumer, _ := controllerIdentity(0x046d, 0xc548, "", "usb-ffb40000.usb-1.2/input1")
	if keyboard == "" || keyboard != consumer {
		t.Fatalf("receiver nodes: keyboard %q consumer control %q", keyboard, consumer)
	}
}
