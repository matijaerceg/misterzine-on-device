//go:build linux

package main

import (
	"bytes"
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
)

// The menu launcher. Main's core browser only lists cores and MGL files, and
// sorgelig will not add script entries to it (Main issue #664), so the app
// gets there the way he suggested for TapTo: a resident helper. The database
// ships MisterZine.mgl at the card root (the menu shows the file name);
// picking it reloads the menu core with the setname "misterzine", which
// Main writes to /tmp/CORENAME. The
// watcher sees that, opens the framebuffer console the way Main's own
// Scripts entry does (F9 from a virtual keyboard, then tty2, where Main
// ignores OSD keys), runs the app there, and reloads the plain menu when it
// exits.

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
	scriptEntry   = "/media/fat/Scripts/misterzine.sh"
)

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
		ensureMGL()
		fmt.Printf("enabled=%v running=%v mgl=%v\n", launcherEnabled(), watcherPID() > 0, fileExists(mglPath))
		return 0
	case "enable":
		if err := launcherEnable(); err != nil {
			fmt.Println("enable:", err)
			return 1
		}
		return launcherStart()
	case "disable":
		launcherDisable()
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
	return err == nil && strings.Contains(string(b), startupLine)
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
	return os.WriteFile(startupScript, []byte(s), 0755)
}

// launcherDisable removes the boot block.
func launcherDisable() {
	b, err := os.ReadFile(startupScript)
	if err != nil {
		return
	}
	var out []string
	for _, ln := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(ln) == startupMark || strings.TrimSpace(ln) == startupLine {
			continue
		}
		out = append(out, ln)
	}
	os.WriteFile(startupScript, []byte(strings.Join(out, "\n")), 0755)
}

func watcherPID() int {
	b, err := os.ReadFile(pidFile)
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

func launcherStop() {
	if pid := watcherPID(); pid > 0 {
		syscall.Kill(pid, syscall.SIGTERM)
	}
	os.Remove(pidFile)
}

// watch is the resident loop.
func watch() int {
	lg := log.New(os.Stdout, "", log.Ltime)
	os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
	lg.Printf("watch: started, pid %d", os.Getpid())
	warmCache()
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
		if err != nil || strings.TrimSpace(string(b)) != "misterzine" {
			continue
		}
		handled = st.ModTime()
		if out, _ := exec.Command("pidof", "misterzine").Output(); len(strings.Fields(string(out))) > 1 {
			lg.Printf("watch: CORENAME says misterzine and the app is already running; leaving it")
			continue
		}
		lg.Printf("watch: misterzine selected in the menu")
		if kbd == nil {
			if kbd, err = mister.NewVKeyboard("misterzine launcher"); err != nil {
				lg.Printf("watch: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}
			time.Sleep(500 * time.Millisecond) // let Main open the new device
		}
		os.Remove(launchedFile)
		if err := runFromMenu(lg, kbd); err != nil {
			lg.Printf("watch: %v", err)
		}
		ensureMGL()
		if b, err := os.ReadFile(launchedFile); err == nil {
			// the app itself loaded a core: leave it alone
			os.Remove(launchedFile)
			lg.Printf("watch: the app launched %s; not touching the menu", strings.TrimSpace(string(b)))
		} else if f, err := os.OpenFile("/dev/MiSTer_cmd", os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			// back to a plain menu: resets CORENAME so this does not retrigger
			f.WriteString("load_core /media/fat/menu.rbf\n")
			f.Close()
		}
		// wait for CORENAME to change before watching again
		for i := 0; i < 40; i++ {
			time.Sleep(250 * time.Millisecond)
			b, _ := os.ReadFile(corenameFile)
			if strings.TrimSpace(string(b)) != "misterzine" {
				break
			}
		}
	}
}

// runFromMenu opens the console and runs the Scripts entry on tty2, like
// Main does for its own Scripts menu, and returns when the app exits.
func runFromMenu(lg *log.Logger, kbd *mister.VKeyboard) error {
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
	launcher := "#!/bin/bash\nexport LC_ALL=en_US.UTF-8\nexport HOME=/root\ncd " + filepath.Dir(scriptEntry) + "\n" + scriptEntry + "\n"
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
