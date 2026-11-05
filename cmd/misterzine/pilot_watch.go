//go:build linux

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform/mister"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
)

const pilotRoot = "/media/fat/misterzine-neogeo-test"

// One detached helper follows one launched game. It exits when the game is
// replaced or when the returned UI closes. Each subsequent launch starts its
// own helper; no boot script or standard-installation watcher is changed.
func startPilotWatch() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	log, err := os.OpenFile(filepath.Join(pilotRoot, "watch.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer log.Close()
	cmd := exec.Command(exe, "pilot-watch")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout, cmd.Stderr = log, log
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

func pilotWatch() int {
	lg := openLog(filepath.Join(pilotRoot, "watch.log"))
	settings, err := store.LoadSettings(filepath.Join(pilotRoot, "settings.json"))
	if err != nil {
		lg.Printf("test watcher: settings: %v", err)
		return 1
	}
	lg.Printf("test watcher: following game, chord=%q return=%v", settings.ExitChord, settings.ReturnAfterGame)
	if !followGame(lg, settings.ExitChord) || !settings.ReturnAfterGame {
		return 0
	}
	if coreName() != menuCore || launcherUpdateActive() {
		return 0
	}
	marker := filepath.Join(pilotRoot, "launched")
	os.Remove(marker)
	kbd, err := mister.NewVKeyboard("MisterZine Neo Geo test return")
	if err != nil {
		lg.Printf("test watcher: keyboard: %v", err)
		return 1
	}
	defer kbd.Close()
	time.Sleep(500 * time.Millisecond)
	lg.Print("test watcher: reopening test app")
	if err := runScriptFromMenu(lg, kbd, true, time.Now(), filepath.Join(pilotRoot, "launch.sh")); err != nil {
		lg.Printf("test watcher: return: %v", err)
		return 1
	}
	// A new game owns the screen after a launch. On ordinary Quit, finish
	// the script session on Menu, as the normal resident launcher does.
	if !fileExists(marker) && coreName() == menuCore {
		sendMainCmd(restoreMenuCmd())
	}
	return 0
}
