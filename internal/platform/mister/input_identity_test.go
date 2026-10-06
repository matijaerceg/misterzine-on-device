//go:build linux

package mister

import "testing"

func TestControllerInstanceIdentity(t *testing.T) {
	a, _ := controllerIdentity(1, 2, "", "usb-1.1/input0")
	b, _ := controllerIdentity(1, 2, "", "usb-1.2/input0")
	if a == b {
		t.Fatal("different ports merged")
	}
	c, _ := controllerIdentity(1, 2, "", "usb-1.1/input1")
	if a != c {
		t.Fatal("receiver interfaces split")
	}
	a, _ = controllerIdentity(1, 2, "serial", "usb-1.1/input0")
	b, _ = controllerIdentity(1, 2, "serial", "usb-1.2/input0")
	if a != b {
		t.Fatal("serial identity changed with port")
	}
}
