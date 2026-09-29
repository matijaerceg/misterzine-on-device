//go:build !arcade

package app

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
