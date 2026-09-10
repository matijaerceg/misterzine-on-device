//go:build linux

package mister

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unsafe"
)

type startMapping struct {
	Code         uint16
	Source, Note string
}

// Main forwards navigation through its virtual keyboard, but omits Start.
// Read the ordinary global v3 controller map for that missing button only.
// Do not guess at per-core, hashed per-device or alternate-mode map files.
func deviceStart(f *os.File) startMapping {
	var bits [96]byte
	if ioctl(f.Fd(), eviocgbitKey(len(bits)), unsafe.Pointer(&bits[0])) != nil {
		return startMapping{}
	}
	var id [4]uint16
	if ioctl(f.Fd(), 0x80084502, unsafe.Pointer(&id[0])) != nil { // EVIOCGID
		return defaultStart(bits)
	}
	return loadStartMapping("/media/fat/config", fmt.Sprintf("%04x_%04x", id[1], id[2]), bits)
}

func hasButton(bits [96]byte, code uint32) bool {
	return code > 0 && code < uint32(len(bits)*8) && bits[code/8]&(1<<uint(code%8)) != 0
}

func defaultStart(bits [96]byte) startMapping {
	if hasButton(bits, btnStart) {
		return startMapping{Code: btnStart, Source: "Linux default"}
	}
	return startMapping{}
}

func loadStartMapping(configDir, id string, bits [96]byte) startMapping {
	name := "input_" + id + "_v3.map"
	for _, dir := range []string{filepath.Join(configDir, "inputs"), configDir} {
		path := filepath.Join(dir, name)
		f, err := os.Open(path)
		if os.IsNotExist(err) {
			continue
		}
		m := startMapping{Source: path}
		if err != nil {
			m.Note = "Cannot read MiSTer Start mapping"
			return m
		}
		b, err := io.ReadAll(io.LimitReader(f, 129))
		f.Close()
		if err != nil || len(b) != 128 { // NUMBUTTONS=32, uint32_t entries
			m.Note = "Invalid MiSTer controller mapping"
			return m
		}
		code := binary.LittleEndian.Uint32(b[11*4:]) // SYS_BTN_START
		if code == 0 {
			m.Note = "Start is unassigned in MiSTer"
		} else if code < 256 || !hasButton(bits, code) {
			m.Note = "Mapped Start is unavailable on this input device"
		} else {
			m.Code = uint16(code)
		}
		// A saved assignment replaces the default, including unassigned or
		// unsupported entries. Never launch from somebody's reassigned Select.
		return m
	}
	return defaultStart(bits)
}
