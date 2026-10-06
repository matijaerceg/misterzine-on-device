//go:build linux

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform/mister"
)

const (
	zaparooMain         = "MiSTer_Zaparoo"
	zaparooMenu         = "zaparoo/menu_zaparoo.rbf"
	zaparooConsoleState = "/tmp/zaparoo_console_state"
	menuLeaseEnv        = "MISTERZINE_MENU_LEASE"
)

// The configured Main decides the NEXT menu load, including before Main has
// started at boot. Merely having a frontend's files installed is not enough.
func configuredMenuCore() string {
	ini, _ := mister.ActiveIni("/media/fat")
	return frontendMenuCore(mister.ReadIni(ini).Main, fileExists)
}

func frontendMenuCore(main string, exists func(string) bool) string {
	var core string
	switch {
	case strings.EqualFold(filepath.Base(main), zaparooMain):
		core = zaparooMenu
	case strings.EqualFold(filepath.Base(main), "MiSTer_Degauss"):
		core = "degauss/menu.rbf"
	}
	if core != "" && exists("/media/fat/"+core) {
		return core
	}
	return ""
}

func menuEntryBody(core string) string {
	if core != "" {
		// These frontends replace stock menu.rbf and lose the MGL's setname
		// before our watcher can see it. Load the matching menu directly.
		return strings.Replace(mglBody, "<rbf>menu</rbf>", "<rbf>"+strings.TrimSuffix(core, ".rbf")+"</rbf>", 1)
	}
	return mglBody
}

func restoreMenuCmd() string {
	if core := configuredMenuCore(); core != "" {
		return "load_core /media/fat/" + core
	}
	return menuCoreCmd(consoleModeRunning())
}

type consoleState struct {
	pid           int
	status, nonce string
}

func parseConsoleState(b []byte) consoleState {
	f := strings.Fields(string(b))
	if len(f) != 4 || f[0] != "1" {
		return consoleState{}
	}
	pid, err := strconv.Atoi(f[1])
	if err != nil || pid <= 0 {
		return consoleState{}
	}
	return consoleState{pid, f[2], f[3]}
}

func readConsoleState() consoleState {
	b, _ := os.ReadFile(zaparooConsoleState)
	s := parseConsoleState(b)
	exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", s.pid))
	if err != nil || filepath.Base(strings.TrimSuffix(exe, " (deleted)")) != zaparooMain {
		return consoleState{}
	}
	return s
}

// An acquisition error must not fall through to a menu load: another client
// may own the display. The watcher leaves that client's session alone.
var errZaparooConsole = errors.New("Zaparoo console handoff")

type consoleLease struct {
	pid   int
	nonce string
	read  func() consoleState
	send  func(string) error
	pause func(time.Duration)
}

func (l consoleLease) acquire() error {
	if err := l.send("zaparoo_console acquire " + l.nonce + " 2"); err != nil {
		return err
	}
	for i := 0; i < 50; i++ {
		s := l.read()
		if s.pid != l.pid {
			return fmt.Errorf("Main changed during console handoff")
		}
		if s.nonce == l.nonce {
			switch s.status {
			case "acquired":
				return nil
			case "busy", "failed":
				return fmt.Errorf("console %s", s.status)
			}
		}
		l.pause(100 * time.Millisecond)
	}
	// A late acknowledgement must not leave the screen leased forever.
	if l.read().pid == l.pid {
		_ = l.send("zaparoo_console release " + l.nonce)
	}
	return fmt.Errorf("console handoff timed out")
}

func (l consoleLease) release() error {
	s := l.read()
	// A game load re-execs Main. Another client's rejected request can
	// overwrite the published state while we still own the console, so
	// release OUR nonce; Main itself rejects it if ownership has changed.
	if s.pid != l.pid || (s.nonce == l.nonce && s.status == "released") {
		return nil
	}
	if err := l.send("zaparoo_console release " + l.nonce); err != nil {
		return err
	}
	for i := 0; i < 50; i++ {
		s = l.read()
		if s.pid != l.pid || (s.nonce == l.nonce && s.status == "released") {
			return nil
		}
		if s.nonce == l.nonce && s.status == "failed" {
			return fmt.Errorf("console lease no longer owned")
		}
		l.pause(100 * time.Millisecond)
	}
	return fmt.Errorf("console release timed out")
}

// A live protocol announcement means the frontend was started by this Main.
// Disabled/absent frontends do not announce it and keep the normal F9 path.
func borrowZaparooConsole(lg *log.Logger) (*consoleLease, error) {
	pids := pidsOf(zaparooMain)
	if len(pids) != 1 {
		return nil, nil
	}
	pid := pids[0]
	for i := 0; i < 30; i++ {
		s := readConsoleState()
		if s.pid == pid {
			l := &consoleLease{pid, fmt.Sprintf("misterzine-%d-%d", os.Getpid(), time.Now().UnixNano()), readConsoleState, sendMainCmd, time.Sleep}
			if err := l.acquire(); err != nil {
				return nil, fmt.Errorf("%w: %v", errZaparooConsole, err)
			}
			lg.Printf("watch: Zaparoo console acquired on tty2")
			return l, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if len(pidsOf("frontend")) > 0 {
		return nil, fmt.Errorf("%w: frontend is running without a current console protocol", errZaparooConsole)
	}
	// Disabling the frontend can leave its tty7 in graphics mode and VT
	// switching locked across Main's re-exec. No frontend is left to undo
	// that state. Repair only its abandoned console before the F9 path.
	if mister.ActiveTTY() == "tty7" {
		lg.Printf("watch: Zaparoo frontend stopped; restoring its abandoned console")
		if mister.ConsoleGraphics("/dev/tty0") {
			if err := mister.ConsoleText("/dev/tty0"); err != nil {
				return nil, err
			}
		}
		if err := mister.UnlockVTSwitch(); err != nil {
			return nil, err
		}
	}
	return nil, nil
}
