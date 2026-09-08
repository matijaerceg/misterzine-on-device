//go:build linux

package mister

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
)

// Never extend a mapping past the memory the framebuffer driver owns.
func framebufferMemorySize(x fbFixScreeninfo, g Geom) (int, error) {
	need := int64(g.Stride) * int64(g.H)
	if g.W <= 0 || g.H <= 0 || g.Stride <= 0 ||
		(g.BPP != 16 && g.BPP != 32) || int64(g.Stride) < int64(g.W)*int64(g.BPP/8) ||
		need <= 0 || need > int64(x.SmemLen) || uint64(x.SmemLen) > uint64(^uint(0)>>1) {
		return 0, fmt.Errorf("invalid framebuffer memory: %s, smem_len=%d", g, x.SmemLen)
	}
	return int(x.SmemLen), nil
}

func physicalFramebufferOffset(x fbFixScreeninfo, mmapErr error) (int64, error) {
	if !errors.Is(mmapErr, syscall.ENODEV) || strings.TrimRight(string(x.ID[:]), "\x00") != "MiSTer_fb" {
		return 0, mmapErr
	}
	// Use FBIOGET_FSCREENINFO, never a hard-coded physical address. The
	// driver excludes its palette/header page from smem_start/smem_len.
	if x.SmemStart == 0 || uint64(x.SmemStart)%uint64(os.Getpagesize()) != 0 ||
		x.SmemLen == 0 || uint64(x.SmemStart)+uint64(x.SmemLen) > 1<<32 {
		return 0, fmt.Errorf("%w; invalid MiSTer_fb physical memory range", mmapErr)
	}
	return int64(x.SmemStart), nil
}
