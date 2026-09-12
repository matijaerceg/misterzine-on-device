//go:build linux

package mister

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	ttyPath    = "/dev/tty0"
	kdgetmode  = 0x4B3B
	kdsetmode  = 0x4B3A
	kdText     = 0
	kdGraphics = 1
	tcgets     = 0x5401
	tcsets     = 0x5402
	tcflsh     = 0x540B
	tciflush   = 0
)

// ErrNotConsole means we are not on a virtual console (run over SSH).
var ErrNotConsole = errors.New("not on a virtual console: open MisterZine from the MiSTer main menu (run MisterZine-Setup in Scripts once if its entry is missing)")

// Console puts the virtual console into graphics mode with a raw, silent
// stdin, and restores both. Restore is idempotent and must run on every
// exit path: a console left in graphics mode shows a black screen until
// reboot, and a raw tty makes the wrapper shell misbehave.
type Console struct {
	tty     *os.File
	mode    uint32
	termios syscall.Termios
	haveTIO bool
	mu      sync.Mutex
	done    bool
	log     *log.Logger
}

// OnVirtualConsole reports whether stdin is a virtual console (/dev/ttyN),
// which is what a Scripts launch gives us. Over SSH stdin is a pty, and
// /dev/tty0 would still open, so this check is what keeps a remote shell
// from taking over the screen.
func OnVirtualConsole() bool {
	target, err := os.Readlink("/proc/self/fd/0")
	if err != nil {
		return false
	}
	if !strings.HasPrefix(target, "/dev/tty") {
		return false
	}
	rest := strings.TrimPrefix(target, "/dev/tty")
	if rest == "" {
		return false
	}
	for _, c := range rest {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// AcquireConsole switches to graphics mode.
func AcquireConsole(lg *log.Logger) (*Console, error) {
	if !OnVirtualConsole() {
		return nil, ErrNotConsole
	}
	f, err := os.OpenFile(ttyPath, os.O_RDWR|syscall.O_NOCTTY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("%w (%v)", ErrNotConsole, err)
	}
	c := &Console{tty: f, log: lg}
	if err := ioctl(f.Fd(), kdgetmode, unsafe.Pointer(&c.mode)); err != nil {
		f.Close()
		return nil, fmt.Errorf("%w (KDGETMODE: %v)", ErrNotConsole, err)
	}
	if err := ioctl(os.Stdin.Fd(), tcgets, unsafe.Pointer(&c.termios)); err == nil {
		raw := c.termios
		raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
		raw.Cc[syscall.VMIN] = 0
		raw.Cc[syscall.VTIME] = 0
		if err := ioctl(os.Stdin.Fd(), tcsets, unsafe.Pointer(&raw)); err == nil {
			c.haveTIO = true
		}
	}
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), kdsetmode, kdGraphics)
	f.WriteString("\x1b[?25l")
	lg.Printf("console: graphics mode (was %d), stdin raw=%v", c.mode, c.haveTIO)
	return c, nil
}

// Restore puts everything back. Safe to call more than once.
func (c *Console) Restore() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.done {
		return
	}
	c.done = true
	syscall.Syscall(syscall.SYS_IOCTL, c.tty.Fd(), kdsetmode, uintptr(c.mode))
	c.tty.WriteString("\x1b[?25h")
	if c.haveTIO {
		ioctl(os.Stdin.Fd(), tcsets, unsafe.Pointer(&c.termios))
	}
	syscall.Syscall(syscall.SYS_IOCTL, os.Stdin.Fd(), tcflsh, tciflush)
	c.tty.Close()
	c.log.Printf("console: restored")
}

// ConsoleGraphics reports whether console p (such as /dev/tty2) is in
// graphics mode: what a program drawing on it sets, and keeps set until it
// exits and restores text mode. False when the console cannot be read.
func ConsoleGraphics(p string) bool {
	f, err := os.OpenFile(p, os.O_RDWR|syscall.O_NOCTTY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return false
	}
	defer f.Close()
	var mode uint32
	if err := ioctl(f.Fd(), kdgetmode, unsafe.Pointer(&mode)); err != nil {
		return false
	}
	return mode == kdGraphics
}

// RestoreAll is the `console-restore` subcommand: text mode and a visible
// cursor on the consoles a script can touch, for the wrapper's exit trap and
// the SSH recovery recipe. Errors are ignored on purpose.
func RestoreAll() {
	for _, p := range []string{"/dev/tty0", "/dev/tty1", "/dev/tty2"} {
		f, err := os.OpenFile(p, os.O_RDWR|syscall.O_NOCTTY|syscall.O_CLOEXEC, 0)
		if err != nil {
			continue
		}
		syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), kdsetmode, kdText)
		f.WriteString("\x1b[?25h")
		syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), tcflsh, tciflush)
		f.Close()
	}
}
