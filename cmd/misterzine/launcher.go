//go:build linux

package main

import (
	"bytes"
	"fmt"
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
// ships misterzine.mgl at the card root; picking it reloads the menu core
// with the setname "misterzine", which Main writes to /tmp/CORENAME. The
// watcher sees that, opens the framebuffer console the way Main's own
// Scripts entry does (F9 from a virtual keyboard, then tty2, where Main
// ignores OSD keys), runs the app there, and reloads the plain menu when it
// exits.

const (
	startupScript = "/media/fat/linux/user-startup.sh"
	startupMark   = "# misterzine"
	startupLine   = "[[ -e /media/fat/misterzine/misterzine ]] && /media/fat/misterzine/misterzine launcher start"
	mglPath       = "/media/fat/misterzine.mgl"
	mglBody       = "<mistergamedescription>\n\t<rbf>menu</rbf>\n\t<setname>misterzine</setname>\n</mistergamedescription>\n"
	pidFile       = "/media/fat/misterzine/watch.pid"
	watchLog      = "/media/fat/misterzine/watch.log"
	corenameFile  = "/tmp/CORENAME"
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

// launcherEnabled reports whether the boot script starts the watcher.
func launcherEnabled() bool {
	b, err := os.ReadFile(startupScript)
	return err == nil && strings.Contains(string(b), startupLine)
}

// launcherEnable adds the boot block and makes sure the MGL exists.
func launcherEnable() error {
	if !fileExists(mglPath) {
		if err := os.WriteFile(mglPath, []byte(mglBody), 0644); err != nil {
			return err
		}
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
	var kbd *mister.VKeyboard
	defer func() {
		if kbd != nil {
			kbd.Close()
		}
	}()
	for {
		time.Sleep(250 * time.Millisecond)
		b, err := os.ReadFile(corenameFile)
		if err != nil || strings.TrimSpace(string(b)) != "misterzine" {
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
		if err := runFromMenu(lg, kbd); err != nil {
			lg.Printf("watch: %v", err)
		}
		// back to a plain menu: resets CORENAME so this does not retrigger
		if f, err := os.OpenFile("/dev/MiSTer_cmd", os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
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
	if p := exec.Command("pidof", "misterzine"); p != nil {
		if out, _ := p.Output(); len(strings.Fields(string(out))) > 1 {
			return fmt.Errorf("the app is already running")
		}
	}
	// Remote's trick: park on tty3, press F9 until Main switches to tty1
	mister.Chvt(3)
	opened := false
	for i := 0; i < 20; i++ {
		kbd.Press(mister.KeyF9)
		time.Sleep(60 * time.Millisecond)
		if mister.ActiveTTY() == "tty1" {
			opened = true
			break
		}
	}
	if !opened {
		return fmt.Errorf("could not open the console (F9)")
	}
	if err := mister.Chvt(2); err != nil {
		return err
	}
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
