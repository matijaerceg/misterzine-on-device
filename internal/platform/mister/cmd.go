//go:build linux

package mister

import (
	"errors"
	"fmt"
	"github.com/matijaerceg/misterzine-on-device/internal/scan"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
)

const cmdPath = "/dev/MiSTer_cmd"

// Cmd writes commands to Main's FIFO. Main parses one command per read, so
// each command is one write, and callers wait for its effect before sending
// another (never chain a framebuffer restore and a load_core).
type Cmd struct {
	mu  sync.Mutex
	log *log.Logger
}

// NewCmd checks that the FIFO exists.
func NewCmd(lg *log.Logger) (*Cmd, error) {
	if _, err := os.Stat(cmdPath); err != nil {
		return nil, fmt.Errorf("%s: %w", cmdPath, err)
	}
	return &Cmd{log: lg}, nil
}

// Available checks for a FIFO reader without sending a command. The reader
// may still disappear later, so Send must continue handling its own errors.
func (c *Cmd) Available() error {
	if c == nil {
		return errors.New("MiSTer command interface unavailable")
	}
	f, err := os.OpenFile(cmdPath, os.O_WRONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	return f.Close()
}

// Send writes one command line.
func (c *Cmd) Send(line string) error {
	if strings.ContainsAny(line, "\n\x00") || len(line) > 900 || line == "" {
		return errors.New("bad command")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	f, err := os.OpenFile(cmdPath, os.O_WRONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line + "\n")
	c.log.Printf("MiSTer_cmd: %q err=%v", line, err)
	return err
}

// LaunchPath turns the app's launch target into an absolute path for
// load_core: a card-relative .mra/.rbf/.mgl path, or "core:NAME" which
// resolves to the newest NAME_*.rbf under the usual core folders.
func LaunchPath(card, target string) (string, error) {
	if strings.ContainsAny(target, "\r\n\x00") {
		return "", errors.New("bad launch target")
	}
	if strings.HasPrefix(target, "core:") {
		return findCore(card, strings.TrimPrefix(target, "core:"))
	}
	if strings.Contains(target, "..") {
		return "", errors.New("bad path")
	}
	ext := strings.ToLower(path.Ext(target))
	if ext != ".mra" && ext != ".rbf" && ext != ".mgl" {
		return "", fmt.Errorf("not launchable: %s", target)
	}
	abs := target
	if !strings.HasPrefix(target, "/") {
		abs = path.Join(card, target)
	}
	if len(abs)+len("load_core ") > 900 {
		return "", errors.New("launch path too long")
	}
	if info, err := os.Stat(abs); err != nil {
		return "", err
	} else if !info.Mode().IsRegular() {
		return "", errors.New("launch target is not a file")
	}
	return abs, nil
}

// findCore uses the same filename and alias rules as the card-status index.
func findCore(card, name string) (string, error) {
	if core, ok := scan.ScanCores(card).Lookup(name); ok {
		return LaunchPath(card, core.Path)
	}
	return "", fmt.Errorf("core %s not found on the card", name)
}
