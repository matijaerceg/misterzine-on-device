package mgl

import "syscall"

// os.Root.Open uses openat without O_LARGEFILE on 32-bit Linux, including
// armv7. A collection can exceed 2 GiB even when each cartridge is small.
const packReadFlags = syscall.O_RDONLY | syscall.O_LARGEFILE | syscall.O_NONBLOCK
