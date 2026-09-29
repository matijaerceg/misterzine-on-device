//go:build !arcade

package app

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// The free build's side of the members' hooks (members.go): none of them
// changes anything.

// membersStart applies the members' settings once the app is built.
func (a *App) membersStart() {}

// membersOptions is the Options rows with the members' own rows added.
func (a *App) membersOptions(e []panelEntry) []panelEntry { return e }

// membersStep applies choice i to a members' Options row of kind.
func (a *App) membersStep(kind string, i int) {}

// membersPress acts on A over a members' Options row of kind, reporting
// whether anything happened.
func (a *App) membersPress(kind string) bool { return false }

// headerNote is what the right end of a group header says for the group
// that view position pos belongs to.
func (a *App) headerNote(pos int) string { return "" }

// membersPaint draws the members' page (ScreenMembers).
func (a *App) membersPaint(c *gfx.Canvas) {}

// membersAct performs a key on the members' page, reporting whether the
// screen changed.
func (a *App) membersAct(k platform.Key) bool { return false }

// membersRepeat is how fast a held key repeats on the members' page; 0
// means it does not.
func (a *App) membersRepeat(k platform.Key) time.Duration { return 0 }
