//go:build !linux

package mgl

import "os"

const packReadFlags = os.O_RDONLY
