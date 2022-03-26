package filesystem

import (
	"golang.org/x/sys/unix"
)

// extraOpenFlags specifies platform specific flags to include in open calls.
const extraOpenFlags = unix.O_LARGEFILE
