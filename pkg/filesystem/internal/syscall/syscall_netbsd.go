package syscall

import (
	"golang.org/x/sys/unix"
)

const (
	// AT_REMOVEDIR is the numeric representation of the AT_REMOVEDIR flag used
	// with the unlinkat system call.
	AT_REMOVEDIR = 0x800
)

// Symlinkat is a Go entry point for the symlinkat system call.
func Symlinkat(target string, directory int, path string) error {
	return unix.Symlinkat(target, directory, path)
}

// Readlinkat is a Go entry point for the readlinkat system call.
func Readlinkat(directory int, path string, buffer []byte) (int, error) {
	return unix.Readlinkat(directory, path, buffer)
}
