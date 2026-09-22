// Package report builds the plain-text card report that Options ->
// Troubleshooting -> Send a report uploads and keeps on the card, so a
// developer can see why a player's list looks the way it does: the app and
// its settings, the list's filters, every local game, every game file that
// is not a row of its own and why, the card scan's totals and the recent
// log. It holds no passwords, Wi-Fi details or Downloader database addresses:
// the host never passes them in, and Scrub strips query strings and keys
// from the log lines that do go in.
package report

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Magic is the report's first line; the service refuses anything else.
const Magic = "MisterZine report v1"

// Limits keep a report small enough to send from a MiSTer and to read.
const (
	MaxBytes    = 256 << 10
	MaxFiles    = 500
	MaxLogLines = 300
)

// AppPart is what only the UI knows: the list as the player sees it.
type AppPart struct {
	View      string // the list order
	Search    string
	Rotation  string // the screen's rotation
	Rows      int    // every row, catalogue and local
	Catalogue int
	Shown     int      // rows in the list right now
	Effective string   // the filters in force, JSON
	Saved     string   // the filters the player chose, JSON
	Hidden    []string // sources hidden by Installed only
	Local     []LocalGame
}

// LocalGame is one local row: a game read from the card's own MRA.
type LocalGame struct {
	K, Title, Core, File string
	StandsFor            string // the catalogue row a standin sorts under
	Hidden               string // the rule hiding it from the list, if any
}

// File is one card file or folder that is no row of its own.
type File struct{ Path, Reason string }

// Outcome is what sending a report came to, in the words the
// Troubleshooting screen shows: the code the service filed it under, where
// the card copy was saved, and why sending or saving failed.
type Outcome struct {
	Code    string // "" when the report was not sent
	Saved   string // the card copy's path, "" when it could not be written
	Problem string // why it was not sent, for the player
	SaveErr string // why the card copy could not be written
}

// DisplayCode is a code as the player reads it out: K7Q2 as it is, and an
// eight-character one in two halves, 7K2Q-9XMB.
func DisplayCode(code string) string {
	if len(code) != 8 {
		return code
	}
	return code[:4] + "-" + code[4:]
}

// Input is the whole report before formatting.
type Input struct {
	Version  string // the app's version line
	Created  time.Time
	System   []string // kernel, board, display, INI summary
	App      AppPart
	Scan     []string // the card scan's totals
	Layout   []string // the card's folders (scan.CardLayout)
	Files    []File   // game files and folders not in the list
	Settings string   // settings.json, JSON
	Log      []string // the latest log lines, oldest first
}

// Build formats the report, within MaxBytes: the log is cut from its oldest
// end first, then the file list.
func Build(in Input) []byte {
	logLines := in.Log
	if len(logLines) > MaxLogLines {
		logLines = logLines[len(logLines)-MaxLogLines:]
	}
	files := in.Files
	for {
		b := format(in, files, logLines)
		if len(b) <= MaxBytes {
			return b
		}
		switch {
		case len(logLines) > 20:
			logLines = logLines[len(logLines)/2:]
		case len(files) > 50:
			files = files[:len(files)/2]
		default:
			return b[:MaxBytes]
		}
	}
}

func format(in Input, files []File, logLines []string) []byte {
	var b bytes.Buffer
	line := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	head := func(title string) { line("\n== %s", title) }
	line("%s", Magic)
	line("App: %s", in.Version)
	line("Created: %s", in.Created.UTC().Format(time.RFC3339))

	head("SYSTEM")
	for _, s := range in.System {
		line("%s", Scrub(s))
	}

	a := in.App
	head("LIST")
	view := a.View
	if a.Search != "" {
		view += fmt.Sprintf(" (search %q)", a.Search)
	}
	line("View: %s; screen rotation %s", view, a.Rotation)
	line("Rows: %d (catalogue %d, local %d); in the list now: %d", a.Rows, a.Catalogue, len(a.Local), a.Shown)
	line("Filters in force: %s", orNone(a.Effective))
	line("Filters chosen: %s", orNone(a.Saved))
	if len(a.Hidden) > 0 {
		line("Sources hidden by Installed only: %s", strings.Join(a.Hidden, ", "))
	}

	head(fmt.Sprintf("LOCAL GAMES (%d)", len(a.Local)))
	if len(a.Local) == 0 {
		line("none")
	}
	for _, g := range a.Local {
		s := fmt.Sprintf("%s | %s | core %s | %s", g.K, g.Title, orNone(g.Core), g.File)
		if g.StandsFor != "" {
			s += " | stands in for " + g.StandsFor
		}
		if g.Hidden != "" {
			s += " | HIDDEN: " + g.Hidden
		}
		line("%s", s)
	}

	head(fmt.Sprintf("GAME FILES NOT IN THE LIST (%d)", len(in.Files)))
	if len(in.Files) == 0 {
		line("none")
	}
	shown := files
	if len(shown) > MaxFiles {
		shown = shown[:MaxFiles]
	}
	for _, f := range shown {
		line("%s | %s", f.Path, f.Reason)
	}
	if more := len(in.Files) - len(shown); more > 0 {
		line("... and %d more", more)
	}

	head("CARD SCAN")
	for _, s := range in.Scan {
		line("%s", Scrub(s))
	}

	head("CARD LAYOUT")
	for _, s := range in.Layout {
		line("%s", s)
	}
	if len(in.Layout) == 0 {
		line("none")
	}

	head("SETTINGS")
	line("%s", orNone(in.Settings))

	head(fmt.Sprintf("LOG (last %d lines)", len(logLines)))
	for _, s := range logLines {
		line("%s", Scrub(s))
	}
	return b.Bytes()
}

func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "none"
	}
	return s
}

var (
	urlQuery = regexp.MustCompile(`(https?://[^\s"'?#]+)[?#][^\s"']*`)
	secretKV = regexp.MustCompile(`(?i)\b(token|key|password|passwd|secret|auth|authorization)(\s*[=:]\s*)("[^"]*"|[^\s,;&]+)`)
)

// Scrub drops what a log line must never carry off the card: the query
// string of any address, and the value of anything named like a key.
func Scrub(s string) string {
	s = urlQuery.ReplaceAllString(s, "$1?...")
	return secretKV.ReplaceAllString(s, "$1$2...")
}
