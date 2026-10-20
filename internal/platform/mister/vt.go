package mister

import (
	"strconv"
	"strings"
	"time"
)

// VTWaitTarget reads a /proc/<pid>/syscall line and reports the console a
// process is blocked waiting for: ioctl (54 on ARM) VT_WAITACTIVE (0x5607),
// whose argument is the console number.
func VTWaitTarget(syscallLine string) (int, bool) {
	f := strings.Fields(syscallLine)
	if len(f) < 4 || f[0] != "54" || f[2] != "0x5607" {
		return 0, false
	}
	n, err := strconv.ParseInt(f[3], 0, 32)
	if err != nil || n < 1 || n > 63 {
		return 0, false
	}
	return int(n), true
}

// awaitConsole polls the active console until it is want or timeout has
// passed. A VT_ACTIVATE is dropped by the kernel while the front console is
// in graphics mode with nobody owning VT switching (a frontend drawing on
// tty2), and VT_WAITACTIVE would then sleep for the rest of the boot; a
// bounded poll turns that into a reported failure.
func awaitConsole(want string, active func() string, pause func(time.Duration), timeout time.Duration) bool {
	const step = 50 * time.Millisecond
	for waited := time.Duration(0); ; waited += step {
		if active() == want {
			return true
		}
		if waited >= timeout {
			return false
		}
		pause(step)
	}
}
