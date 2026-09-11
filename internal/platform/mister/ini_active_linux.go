//go:build linux

package mister

import (
	"os"
	"syscall"
)

// activeAltcfg reads Main's alternative-INI marker from physical memory.
// Anything short of a readable, well-formed marker means the primary INI.
func activeAltcfg() int {
	f, err := os.OpenFile("/dev/mem", os.O_RDONLY|syscall.O_SYNC|syscall.O_CLOEXEC, 0)
	if err != nil {
		return 0
	}
	defer f.Close()
	m, err := syscall.Mmap(int(f.Fd()), altcfgPage, os.Getpagesize(), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return 0
	}
	defer syscall.Munmap(m)
	if altcfgOffset+4 > len(m) {
		return 0
	}
	b := make([]byte, 4)
	copy(b, m[altcfgOffset:altcfgOffset+4])
	return altcfgIndex(b)
}
