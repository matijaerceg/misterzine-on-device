//go:build linux

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform/mister"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

// Main lists cores and MGL files. The resident launcher watches for our MGL's
// setname, opens the script console, and restores Menu when the app exits.

const (
	startupScript = "/media/fat/linux/user-startup.sh"
	startupMark   = "# misterzine"
	startupLine   = "[[ -e /media/fat/misterzine/misterzine ]] && /media/fat/misterzine/misterzine launcher start"
	mglPath       = "/media/fat/MisterZine.mgl"
	mglBody       = "<mistergamedescription>\n\t<rbf>menu</rbf>\n\t<setname>misterzine</setname>\n</mistergamedescription>\n"
	pidFile       = "/media/fat/misterzine/watch.pid"
	watchLog      = "/media/fat/misterzine/watch.log"
	corenameFile  = "/tmp/CORENAME"
	launchedFile  = "/media/fat/misterzine/launched" // written by the app right before it loads a core
	scriptEntry   = "/media/fat/misterzine/launch.sh"
	bootMark      = "/tmp/misterzine-boot" // /tmp empties at boot: the first watcher of a boot creates it
	menuCore      = "MENU"                 // what Main writes to CORENAME for its own menu
)

// watchSettings reads the Options the watcher carries out (Open at boot,
// Return after game); the app saves settings.json on every change.
func watchSettings() store.Settings {
	s, _ := store.LoadSettings(filepath.Join(filepath.Dir(pidFile), "settings.json"))
	return s
}

func coreName() string {
	b, _ := os.ReadFile(corenameFile)
	return strings.TrimSpace(string(b))
}

// sendMainCmd writes one line to Main's command FIFO. Main recreates the
// FIFO each time it restarts (every core load) and opens it a moment after
// it has written CORENAME, so right after a core change the write can find
// no FIFO yet (ENOENT) or no reader (ENXIO); those are retried for 10 s.
func sendMainCmd(line string) error {
	var err error
	for i := 0; i < 40; i++ {
		var f *os.File
		if f, err = os.OpenFile("/dev/MiSTer_cmd", os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_, err = f.WriteString(line + "\n")
			f.Close()
			return err
		}
		if !errors.Is(err, syscall.ENXIO) && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		time.Sleep(250 * time.Millisecond)
	}
	return err
}

// firstWatcherSinceBoot is true once per boot: the mark lives in /tmp, which
// is empty after power-on, and the watcher a Downloader update restarts or a
// Setup run hours later finds it (or fails the uptime bound).
func firstWatcherSinceBoot() bool {
	if fileExists(bootMark) {
		return false
	}
	os.WriteFile(bootMark, nil, 0644)
	b, _ := os.ReadFile("/proc/uptime")
	if f := strings.Fields(string(b)); len(f) > 0 {
		if up, err := strconv.ParseFloat(f[0], 64); err == nil {
			return up < 300
		}
	}
	return false
}

// awaitMenuAtBoot polls until Main has written a core name after boot and
// reports whether it is the menu: a bootcore from the INI comes up instead
// and is left alone, as is a Main that never appears before the deadline.
func awaitMenuAtBoot(probe func() string, pause func(time.Duration), deadline time.Duration) bool {
	for waited := time.Duration(0); waited < deadline; waited += 250 * time.Millisecond {
		switch probe() {
		case menuCore:
			return true
		case "":
			pause(250 * time.Millisecond)
		default:
			return false
		}
	}
	return false
}

// awaitGameExit follows the core the app just launched: true once the game
// has come up and later given way to Main's menu (the user left it through
// the OSD). False when the game never appeared before loadDeadline, or when
// another core replaced it without passing through the menu (something else
// took over). An empty name is Main between two writes; it is waited out.
func awaitGameExit(probe func() string, pause func(time.Duration), loadDeadline time.Duration) bool {
	game := ""
	for waited := time.Duration(0); game == ""; waited += 250 * time.Millisecond {
		if waited >= loadDeadline {
			return false
		}
		pause(250 * time.Millisecond)
		if core := probe(); core != "" && core != menuCore && core != "misterzine" {
			game = core
		}
	}
	for {
		pause(250 * time.Millisecond)
		switch probe() {
		case game, "":
		case menuCore:
			return true
		default:
			return false
		}
	}
}

// launcherCmd handles: launcher start | stop | status | enable | disable
func launcherCmd(args []string) int {
	if len(args) == 0 {
		fmt.Println("usage: misterzine launcher start|stop|status|enable|disable")
		return 2
	}
	switch args[0] {
	case "start":
		return launcherStart()
	case "stop":
		launcherStop()
		return 0
	case "status":
		fmt.Printf("enabled=%v running=%v mgl=%v\n", launcherEnabled(), watcherPID() > 0, fileExists(mglPath))
		return 0
	case "enable":
		if err := launcherEnable(); err != nil {
			fmt.Println("enable:", err)
			return 1
		}
		return launcherStart()
	case "disable":
		if err := launcherDisable(); err != nil {
			fmt.Println("disable:", err)
			return 1
		}
		launcherStop()
		return 0
	case "watch":
		return watch()
	}
	fmt.Println("unknown launcher command", args[0])
	return 2
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

// warmCache reads the binary and the data cache once so the first start
// after a cold boot is not slower than the ones after it.
func warmCache() {
	exe, _ := os.Executable()
	for _, p := range []string{exe, "/media/fat/misterzine/cache/data.json", "/media/fat/misterzine/cache/meta.json", "/media/fat/misterzine/cache/alts.json"} {
		if p == "" {
			continue
		}
		if f, err := os.Open(p); err == nil {
			io.Copy(io.Discard, f)
			f.Close()
		}
	}
}

// launcherEnabled reports whether the boot script starts the watcher.
func launcherEnabled() bool {
	b, err := os.ReadFile(startupScript)
	return err == nil && hasStartupHook(string(b))
}

func isStartupHook(line string) bool {
	code, _, _ := strings.Cut(strings.TrimSpace(line), "#")
	return strings.TrimSpace(code) == startupLine
}

func hasStartupHook(script string) bool {
	for _, line := range strings.Split(script, "\n") {
		if isStartupHook(line) {
			return true
		}
	}
	return false
}

// ensureMGL makes sure the MGL exists under its proper name. The card is
// case-insensitive, so an older lowercase file answers to the new name too
// and would keep the menu entry lowercase; it is renamed. Downloader may
// also have removed the old path after installing the new one (same file on
// this filesystem), in which case the MGL is written back.
func ensureMGL() error {
	dir, base := filepath.Split(mglPath)
	if ents, err := os.ReadDir(dir); err == nil {
		exact := false
		for _, e := range ents {
			if e.Name() == base {
				exact = true
			}
		}
		if !exact {
			for _, e := range ents {
				if !e.IsDir() && strings.EqualFold(e.Name(), base) {
					// a case-only rename is a no-op on exFAT: go through a temp name
					tmp := mglPath + ".tmp"
					if err := os.Rename(filepath.Join(dir, e.Name()), tmp); err == nil {
						if err := os.Rename(tmp, mglPath); err == nil {
							return nil
						}
					}
				}
			}
		}
		if exact {
			return nil
		}
	}
	return os.WriteFile(mglPath, []byte(mglBody), 0644)
}

// launcherEnable adds the boot block and makes sure the MGL exists.
func launcherEnable() error {
	if err := ensureMGL(); err != nil {
		return err
	}
	if launcherEnabled() {
		return nil
	}
	b, err := os.ReadFile(startupScript)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	s := string(b)
	if s == "" {
		s = "#!/bin/sh\n"
	}
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	s += "\n" + startupMark + "\n" + startupLine + "\n"
	return writeStartup(startupScript, []byte(s))
}

// writeStartup preserves the existing script and permissions until the
// replacement is written and synced. Other apps share this boot script.
func writeStartup(path string, b []byte) error {
	mode := os.FileMode(0755)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".misterzine-startup-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := f.Chmod(mode); err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// launcherDisable removes both the boot block and the main-menu entry.
func launcherDisable() error { return disableLauncherFiles(startupScript, mglPath) }

func disableLauncherFiles(startup, mgl string) error {
	b, err := os.ReadFile(startup)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var out []string
	for _, ln := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(ln) == startupMark || isStartupHook(ln) {
			continue
		}
		out = append(out, ln)
	}
	if next := []byte(strings.Join(out, "\n")); !bytes.Equal(next, b) {
		if err := writeStartup(startup, next); err != nil {
			return err
		}
	}
	if err := os.Remove(mgl); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func watcherPID() int { return watcherPIDFrom(pidFile) }

func watcherPIDFrom(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	if pid <= 0 {
		return 0
	}
	if err := syscall.Kill(pid, 0); err != nil {
		return 0
	}
	cmd, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil || !bytes.Contains(cmd, []byte("watch")) {
		return 0
	}
	return pid
}

// launcherStart spawns the watcher detached, once.
func launcherStart() int {
	if launcherEnabled() {
		ensureMGL()
	}
	if watcherPID() > 0 {
		return 0
	}
	exe, err := os.Executable()
	if err != nil {
		fmt.Println("start:", err)
		return 1
	}
	lg, _ := os.OpenFile(watchLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	cmd := exec.Command(exe, "launcher", "watch")
	cmd.Stdout, cmd.Stderr = lg, lg
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		fmt.Println("start:", err)
		return 1
	}
	os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)
	return 0
}

// processAncestor follows the actual parent chain, including the login shell
// and Scripts wrapper between the watcher and the on-screen app.
func processAncestor(ancestor, pid int) bool {
	for depth := 0; pid > 1 && depth < 64; depth++ {
		if pid == ancestor {
			return true
		}
		b, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
		if err != nil {
			return false
		}
		parent := 0
		for _, line := range strings.Split(string(b), "\n") {
			if fields := strings.Fields(line); len(fields) == 2 && fields[0] == "PPid:" {
				parent, _ = strconv.Atoi(fields[1])
				break
			}
		}
		if parent == pid {
			return false
		}
		pid = parent
	}
	return false
}

func launcherStop() { stopWatcher(pidFile) }

func stopWatcher(path string) {
	if pid := watcherPIDFrom(path); pid > 0 {
		if processAncestor(pid, os.Getpid()) {
			// This watcher owns our console session. Leave it alive to restore
			// Menu when we exit; watch then stops if the setting is still off.
			return
		}
		syscall.Kill(pid, syscall.SIGTERM)
	}
	os.Remove(path)
}

func removeWatcherPID(path string, pid int) {
	if b, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(b)) == strconv.Itoa(pid) {
		os.Remove(path)
	}
}

// watch is the resident loop.
func watch() int {
	lg := log.New(os.Stdout, "", log.Ltime)
	os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
	defer removeWatcherPID(pidFile, os.Getpid())
	lg.Printf("watch: started, pid %d", os.Getpid())
	exe, _ := os.Executable()
	runningBinary, _ := os.Stat("/proc/self/exe")
	nextBinaryCheck := time.Now()
	warmCache()
	if first := firstWatcherSinceBoot(); first && launcherEnabled() && watchSettings().OpenAtBoot {
		lg.Printf("watch: open at boot: waiting for the menu")
		if awaitMenuAtBoot(coreName, time.Sleep, 2*time.Minute) {
			ensureMGL()
			if err := sendMainCmd("load_core " + mglPath); err != nil {
				lg.Printf("watch: open at boot: %v", err)
			}
		} else {
			lg.Printf("watch: open at boot: the menu did not come up (bootcore?); leaving it")
		}
	}
	resume := false // the next app run was a return after a game
	var kbd *mister.VKeyboard
	defer func() {
		if kbd != nil {
			kbd.Close()
		}
	}()
	// Main rewrites /tmp/CORENAME on every core load, so a fresh write
	// saying "misterzine" is a selection: act on each one once, keyed by
	// the file's mtime. A selection made before the watcher was up (a quick
	// press after a cold boot) counts too: /tmp is empty at boot, so the
	// file cannot be stale.
	var handled time.Time
	for {
		time.Sleep(100 * time.Millisecond)
		st, err := os.Stat(corenameFile)
		if err != nil || st.ModTime().Equal(handled) {
			continue
		}
		b, err := os.ReadFile(corenameFile)
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(b)) != "misterzine" {
			if time.Now().After(nextBinaryCheck) {
				nextBinaryCheck = time.Now().Add(2 * time.Second)
				if binaryReplaced(exe, runningBinary) && !launcherUpdateActive() {
					lg.Printf("watch: binary replaced; restarting launcher")
					if kbd != nil {
						kbd.Close()
						kbd = nil
					}
					if err := syscall.Exec(exe, []string{exe, "launcher", "watch"}, os.Environ()); err != nil {
						lg.Printf("watch: replacement could not start: %v; keeping current launcher", err)
						nextBinaryCheck = time.Now().Add(30 * time.Second)
					}
				}
			}
			continue
		}
		handled = st.ModTime()
		if out, _ := exec.Command("pidof", "misterzine").Output(); len(strings.Fields(string(out))) > 1 {
			lg.Printf("watch: CORENAME says misterzine and the app is already running; leaving it")
			continue
		}
		lg.Printf("watch: misterzine selected in the menu")
		if s, _ := updater.Read(filepath.Dir(pidFile)); !s.Active() && updater.OtherScript() {
			lg.Printf("watch: another updater is running; finish it before opening MisterZine")
			continue // leave its script console and /tmp/script untouched
		}
		if kbd == nil {
			if kbd, err = mister.NewVKeyboard("misterzine launcher"); err != nil {
				lg.Printf("watch: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}
			time.Sleep(500 * time.Millisecond) // let Main open the new device
		}
		os.Remove(launchedFile)
		if err := runFromMenu(lg, kbd, resume); err != nil {
			lg.Printf("watch: %v", err)
		}
		resume = false
		enabled := launcherEnabled()
		if enabled {
			ensureMGL()
		}
		if b, err := os.ReadFile(launchedFile); err == nil {
			// the app itself loaded a core: leave it alone
			os.Remove(launchedFile)
			lg.Printf("watch: the app launched %s; not touching the menu", strings.TrimSpace(string(b)))
			if enabled && watchSettings().ReturnAfterGame {
				lg.Printf("watch: return after game: waiting for the game to exit to the menu")
				if awaitGameExit(coreName, time.Sleep, 20*time.Second) && !launcherUpdateActive() {
					lg.Printf("watch: game exited; reopening MisterZine")
					resume = true
					if err := sendMainCmd("load_core " + mglPath); err != nil {
						lg.Printf("watch: return after game: %v", err)
						resume = false
					}
					continue // the selection arrives as a fresh CORENAME write
				}
				lg.Printf("watch: not returning: the game did not load, or another core took over")
			}
		} else if waitForMenuRestore(lg, func() (string, bool) {
			b, _ := os.ReadFile(corenameFile)
			return strings.TrimSpace(string(b)), launcherUpdateActive()
		}, time.Sleep) {
			// back to a plain menu: resets CORENAME so this does not retrigger
			sendMainCmd("load_core /media/fat/menu.rbf")
		}
		// wait for CORENAME to change before watching again
		for i := 0; i < 40; i++ {
			time.Sleep(250 * time.Millisecond)
			b, _ := os.ReadFile(corenameFile)
			if strings.TrimSpace(string(b)) != "misterzine" {
				break
			}
		}
		if !enabled {
			lg.Printf("watch: launcher disabled; finished the app session and stopping")
			return 0
		}
	}
}

// runFromMenu opens the console and runs the app wrapper on tty2, like
// Main does for its own Scripts menu, and returns when the app exits.
// resume tells the app it is reopening after a game.
func runFromMenu(lg *log.Logger, kbd *mister.VKeyboard, resume bool) error {
	time.Sleep(1200 * time.Millisecond) // Main has just re-executed itself
	// Remote's trick: park on tty3, press F9 until Main switches to tty1
	lg.Printf("watch: console: active %s, fb mode %q before", mister.ActiveTTY(), mister.SysfsMode())
	mister.Chvt(3)
	opened := false
	presses := 0
	for i := 0; i < 20; i++ {
		kbd.Press(mister.KeyF9)
		presses++
		time.Sleep(60 * time.Millisecond)
		if mister.ActiveTTY() == "tty1" {
			opened = true
			break
		}
	}
	if !opened {
		return fmt.Errorf("could not open the console (F9 pressed %d times, active %s)", presses, mister.ActiveTTY())
	}
	if err := mister.Chvt(2); err != nil {
		return err
	}
	lg.Printf("watch: console open after %d F9 press(es): active %s, fb mode %q", presses, mister.ActiveTTY(), mister.SysfsMode())
	entry := scriptEntry
	if !fileExists(entry) {
		// Allow an older installation to finish an update before its new
		// wrapper arrives. Downloader removes this old Scripts entry.
		entry = "/media/fat/Scripts/misterzine.sh"
	}
	args := ""
	if resume {
		args = " --resume"
	}
	launcher := "#!/bin/bash\nexport LC_ALL=en_US.UTF-8\nexport HOME=/root\ncd " + filepath.Dir(entry) + "\n" + entry + args + "\n"
	if err := os.WriteFile("/tmp/script", []byte(launcher), 0700); err != nil {
		return err
	}
	lg.Printf("watch: running the app on tty2")
	t0 := time.Now()
	cmd := exec.Command("/sbin/agetty", "-a", "root", "-l", "/tmp/script", "--nohostname", "-L", "tty2", "linux")
	err := cmd.Run()
	lg.Printf("watch: app finished after %v (err=%v)", time.Since(t0).Round(time.Second), err)
	return nil
}

// The idle launcher may be an old inode after Downloader replaced the file.
// Missing or non-regular replacements leave the working process alone.
func binaryReplaced(path string, running os.FileInfo) bool {
	if running == nil || path == "" {
		return false
	}
	current, err := os.Stat(path)
	return err == nil && current.Mode().IsRegular() && !os.SameFile(running, current)
}

func launcherUpdateActive() bool {
	s, _ := updater.Read(filepath.Dir(pidFile))
	return s.Active() || updater.OtherScript()
}

// Keep the updater's console/core intact after the UI exits. If another core
// has already been selected, it owns the screen and must not be replaced.
func waitForMenuRestore(lg *log.Logger, probe func() (string, bool), pause func(time.Duration)) bool {
	announced := false
	for {
		core, active := probe()
		if core != "misterzine" {
			return false
		}
		if !active {
			return true
		}
		if !announced {
			lg.Printf("watch: waiting for updater before restoring Menu")
			announced = true
		}
		pause(500 * time.Millisecond)
	}
}
