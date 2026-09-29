package app

// MisterZine Arcade, the members' build, compiles extra files into this
// package from a private repository with the arcade build tag. They plug
// into the hooks that members_free.go stubs out for the free build, where
// every hook leaves things exactly as they are.

// MembersSettings is Config.Members for the host to save: the members'
// extras' settings, which the free build carries through untouched.
func (a *App) MembersSettings() map[string]string {
	if len(a.cfg.Members) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range a.cfg.Members {
		out[k] = v
	}
	return out
}
