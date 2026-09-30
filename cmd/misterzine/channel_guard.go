//go:build linux

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/beta"
	"github.com/matijaerceg/misterzine-on-device/internal/buildinfo"
	"github.com/matijaerceg/misterzine-on-device/internal/scan"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

// Two guards against a card sliding between the free MisterZine and
// MisterZine Arcade without the member choosing it. Both builds install as
// the one Downloader entry, [misterzine], and only its db_url says which
// build the next Update All brings; MiSTer Companion's Install Center
// rewrites that entry to the free database, and so does copying the
// free release's drop-in again.
//
//   - The beta, unlocked, points its entry back at the beta when it finds
//     it following the free database (or gone), and says so. A locked beta
//     leaves it alone: that member may be on the way out.
//   - The free build, on a card that holds a beta unlock and whose entry
//     follows the free database, asks once whether to go back
//     (app.Config.ArcadeBack), through the member's own installer.

// followsFree reports an entry that fetches the free releases.
func followsFree(urls []string) bool {
	for _, u := range urls {
		if u == strings.ToLower(buildinfo.FreeDBURL) {
			return true
		}
	}
	return false
}

// arcadeBackOffered says whether the free build asks, at start, to go back
// to MisterZine Arcade: once, on a card that had the beta, whose entry now
// follows the free releases and whose member's installer can take it back.
func arcadeBackOffered(root, card string, asked bool) bool {
	if beta.On() || asked || !beta.Unlocked(root) {
		return false
	}
	urls, found := scan.MisterZineEntries(card)
	return found && followsFree(urls) && updater.CanSwitchToBeta(card)
}

// betaEntryNeeded says whether an unlocked beta should point its entry
// back at the beta database: it follows the free releases, or the card's
// downloader.ini has no misterzine entry at all.
func betaEntryNeeded(root, card string) bool {
	if !beta.On() || beta.Check(root) != nil {
		return false
	}
	urls, found := scan.MisterZineEntries(card)
	return found && (len(urls) == 0 || followsFree(urls))
}

// Notices of the beta's guard, short enough for the status bar in tate.
const (
	betaEntryKept   = "Update All keeps Beta now"
	betaEntryFailed = "Run Install-Beta to keep Beta"
)

// keepBetaEntry runs the beta's guard off the UI goroutine: channel.py,
// which the beta installs, rewrites the entry without running Downloader,
// and the answer shows as a notice.
func (h *host) keepBetaEntry() {
	if !betaEntryNeeded(h.root, h.card) {
		return
	}
	script := filepath.Join(h.card, "misterzine", "channel.py")
	if _, err := os.Stat(script); err != nil {
		h.lg.Printf("beta entry: follows the free releases, and %v", err)
		return
	}
	go func() {
		out, err := exec.Command("python3", script, "beta", "--point-only", "--card", h.card).CombinedOutput()
		h.lg.Printf("beta entry: %s (%v)", strings.TrimSpace(string(out)), err)
		notice := betaEntryKept
		if err != nil {
			notice = betaEntryFailed
		}
		h.runOnUI(func() { h.a.Notice(notice, 12*time.Second) })
	}()
}
