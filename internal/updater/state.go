// Package updater supervises Update All independently of the framebuffer UI.
package updater

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const TailLines = 160

var Stages = []string{"Prepare", "Check", "Update", "Extras", "Finish"}

type State struct {
	ID              string    `json:"id"`
	PID             int       `json:"pid"`
	Boot            string    `json:"boot"`
	Status          string    `json:"status"`
	Stage           int       `json:"stage"`
	Label           string    `json:"label"`
	Message         string    `json:"message,omitempty"`
	Started         time.Time `json:"started"`
	Heartbeat       time.Time `json:"heartbeat"`
	LastOutput      time.Time `json:"last_output"`
	Finished        time.Time `json:"finished,omitempty"`
	Elapsed         int       `json:"elapsed"`
	Protected       bool      `json:"protected"`
	CancelRequested bool      `json:"cancel_requested"`
	Reboot          bool      `json:"reboot"`
	SawSuccess      bool      `json:"saw_success"`
	HadErrors       bool      `json:"had_errors"`
	Lines           []string  `json:"lines"`
}

func (s State) Active() bool {
	return s.Status == "starting" || s.Status == "running" || s.Status == "cancelling"
}

func stateDir(root string) string  { return filepath.Join(root, "update-all") }
func StatePath(root string) string { return filepath.Join(stateDir(root), "state.json") }
func LogPath(root string) string   { return filepath.Join(stateDir(root), "output.log") }
func livePath(root string) string {
	h := fnv.New64a()
	h.Write([]byte(root))
	return filepath.Join(os.TempDir(), fmt.Sprintf("misterzine-update-%x.json", h.Sum64()))
}

func save(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err = os.WriteFile(path+".tmp", b, 0600); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}

// Read never treats a vanished/rebooted worker as a successful update.
func Read(root string) (State, error) {
	var s State
	b, err := os.ReadFile(livePath(root))
	if os.IsNotExist(err) {
		b, err = os.ReadFile(StatePath(root))
	}
	if err != nil {
		return s, err
	}
	if err = json.Unmarshal(b, &s); err != nil {
		return s, err
	}
	if s.Active() && !workerAlive(s) {
		s.Status = "interrupted"
		s.Message = "Updater stopped or system restarted. Check the log."
		s.Protected = false
		if restarted(s) {
			s.Reboot = false
			if s.SawSuccess && !s.HadErrors {
				s.Status = "restarted"
				s.Message = "Update All reported success before restart."
			} else if s.HadErrors {
				s.Message = "Restarted after update errors. Review the log."
			}
		}
	}
	return s, nil
}

var ansi = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)
var urlSecret = regexp.MustCompile(`(?i)([?&](?:token|key|password|secret|auth|access_token|api_key)=)[^ &]+`)
var urlLogin = regexp.MustCompile(`(https?://)[^ /@]+:[^ /@]+@`)

func cleanLine(line string) string {
	line = ansi.ReplaceAllString(line, "")
	line = strings.Map(func(r rune) rune {
		if r == '\t' {
			return ' '
		}
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, line)
	line = strings.TrimSpace(line)
	lower := strings.ToLower(line)
	if strings.Contains(lower, "authorization:") || strings.Contains(lower, "bearer ") ||
		strings.Contains(lower, "password") || strings.Contains(lower, "patreonkey") ||
		strings.Contains(lower, "access_token") || strings.Contains(lower, "api_key") {
		return "[private configuration omitted]"
	}
	line = urlSecret.ReplaceAllString(line, "${1}[hidden]")
	line = urlLogin.ReplaceAllString(line, "${1}[hidden]@")
	if runes := []rune(line); len(runes) > 512 {
		line = string(runes[:512])
	}
	return line
}

func (s *State) output(line string, now time.Time) string {
	line = cleanLine(line)
	if line == "" {
		return ""
	}
	s.LastOutput = now
	s.Lines = append(s.Lines, line)
	if len(s.Lines) > TailLines {
		s.Lines = append([]string(nil), s.Lines[len(s.Lines)-TailLines:]...)
	}
	s.observe(line)
	return line
}

func (s *State) observe(line string) {
	l := strings.ToLower(line)
	advance := func(stage int, label string) {
		if stage >= s.Stage {
			s.Stage, s.Label = stage, label
		}
	}
	switch {
	case strings.Contains(l, "there were some errors in the updaters"):
		s.HadErrors = true
		advance(4, "Finished with errors")
	case strings.Contains(l, "linux will be updated"), strings.Contains(l, "fetching the new linux image"), strings.Contains(l, "hold your breath"), strings.Contains(l, "installing analogue pocket firmware"):
		s.Protected = true
		if strings.Contains(l, "linux") || strings.Contains(l, "hold your breath") {
			s.Reboot = true
		}
		advance(2, "Updating system files")
	case strings.Contains(l, "linux has been updated"), strings.Contains(l, "your pocket firmware is on"), strings.Contains(l, "your pocket firmware could not"):
		s.Protected = false
		s.Reboot = strings.Contains(l, "linux has been updated") || s.Reboot
	case strings.Contains(l, "rebooting "), strings.Contains(l, "you should reboot"):
		s.Reboot = true
		advance(4, "Restart required")
	case strings.HasPrefix(l, "success! log"), strings.HasPrefix(l, "success! more details at:"):
		s.SawSuccess = true
		advance(4, "Finishing")
	case strings.Contains(l, "running arcade organizer"), strings.Contains(l, "backing up analogue pocket"):
		advance(3, "Running extras")
	case strings.Contains(l, "running mister downloader"), strings.HasPrefix(l, "reading sections from"):
		advance(1, "Checking for updates")
	case s.Stage >= 1 && (strings.HasPrefix(l, "downloading ") || strings.HasPrefix(l, "fetching ") || strings.HasPrefix(l, "installing ") || strings.HasPrefix(l, "removing ") || strings.HasPrefix(l, "updates found")):
		advance(2, "Updating files")
	}
}

func (s State) Summary() string {
	if s.Message != "" {
		return s.Message
	}
	if s.CancelRequested && s.Protected {
		return "Cancel requested; finishing system update"
	}
	if s.Protected {
		return "System update: cancellation waits until safe"
	}
	if s.Reboot {
		return "Update All may restart the system"
	}
	if s.Status == "completed" {
		return "Update All finished successfully"
	}
	return "Updating with your saved Update All settings"
}
