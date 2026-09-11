package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"reflect"
)

func (a *App) onlyFacet(kind string) map[string]int {
	switch kind {
	case "base":
		return a.ds.Facets.Base
	case "src":
		return a.ds.Facets.Src
	case "rot":
		return a.ds.Facets.Rot
	case "plr":
		return a.ds.Facets.Plr
	case "res":
		return a.ds.Facets.Res
	case "genre":
		return a.ds.Facets.Genre
	case "directions":
		return a.ds.Facets.Directions
	case "buttons":
		return a.ds.Facets.Buttons
	}
	return nil
}

func (a *App) canOnlyFilter() bool {
	if a.screen != ScreenFilter || a.panel.cursor >= len(a.panel.entries) {
		return false
	}
	e := a.panel.entries[a.panel.cursor]
	return !e.header && !e.info && (a.onlyFacet(e.kind) != nil || e.kind == "install" || e.kind == "fav" || e.kind == "since")
}

func (a *App) onlyFilter() bool {
	if !a.canOnlyFilter() {
		return false
	}
	e := a.panel.entries[a.panel.cursor]
	f := a.filters
	off := map[string]bool{}
	for value := range a.onlyFacet(e.kind) {
		if value != e.value {
			off[value] = true
		}
	}
	switch e.kind {
	case "base":
		f.BaseOff = off
	case "src":
		f.SrcOff = off
	case "rot":
		f.RotOff = off
	case "plr":
		f.PlrOff = off
	case "res":
		f.ResOff = off
	case "genre":
		f.GenreOff = off
	case "directions":
		f.DirectionsOff = off
	case "buttons":
		f.ButtonsOff = off
	case "install":
		f.Install = e.value
	case "fav":
		f.FavOnly = true
	case "since":
		f.Since = true
	}
	if sameSection(e.kind, a.filters, f) {
		copySection(e.kind, &f, data.Filters{})
	}
	a.SetFilters(f)
	a.buildPanel()
	return true
}

// Compare/copy only one section so enabling all never changes other sections.
func copySection(kind string, to *data.Filters, from data.Filters) {
	switch kind {
	case "base":
		to.BaseOff = from.BaseOff
	case "src":
		to.SrcOff = from.SrcOff
	case "rot":
		to.RotOff = from.RotOff
	case "plr":
		to.PlrOff = from.PlrOff
	case "res":
		to.ResOff = from.ResOff
	case "genre":
		to.GenreOff = from.GenreOff
	case "directions":
		to.DirectionsOff = from.DirectionsOff
	case "buttons":
		to.ButtonsOff = from.ButtonsOff
	case "install":
		to.Install = from.Install
	case "fav":
		to.FavOnly = from.FavOnly
	case "since":
		to.Since = from.Since
	}
}

func sameSection(kind string, a, b data.Filters) bool {
	var left, right data.Filters
	copySection(kind, &left, a)
	copySection(kind, &right, b)
	return reflect.DeepEqual(left, right)
}

func (a *App) noChangesLabel() string {
	if len(a.view) > data.SplitScan {
		return "No changes in top " + itoa(data.SplitScan)
	}
	return "No changes in this view"
}
