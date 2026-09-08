//go:build linux

package mister

import (
	"errors"
	"fmt"
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
	if _, err := os.Stat(abs); err != nil {
		return "", err
	}
	return abs, nil
}

// findCore looks for NAME_YYYYMMDD.rbf (newest date) in the core folders.
func findCore(card, name string) (string, error) {
	lname := strings.ToLower(name)
	best, bestDate := "", ""
	for _, dir := range []string{"_Console", "_Computer", "_Other", "_Utility", "_Arcade/cores"} {
		entries, err := os.ReadDir(path.Join(card, dir))
		if err != nil {
			continue
		}
		for _, e := range entries {
			n := e.Name()
			ln := strings.ToLower(n)
			if !strings.HasSuffix(ln, ".rbf") {
				continue
			}
			stem := strings.TrimSuffix(ln, ".rbf")
			base, date := stem, ""
			if i := strings.LastIndex(stem, "_"); i > 0 && len(stem)-i-1 >= 8 {
				if d := stem[i+1 : i+9]; isDigits(d) {
					base, date = stem[:i], d
				}
			}
			if base != lname {
				continue
			}
			if best == "" || date > bestDate {
				best, bestDate = path.Join(card, dir, n), date
			}
		}
	}
	if best == "" {
		return "", fmt.Errorf("core %s not found on the card", name)
	}
	return best, nil
}

func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}
