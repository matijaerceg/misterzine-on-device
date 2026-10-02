//go:build !arcade && linux

package main

import "github.com/matijaerceg/misterzine-on-device/internal/app"

func (h *host) configureControls(cfg *app.Config, root string) {}
