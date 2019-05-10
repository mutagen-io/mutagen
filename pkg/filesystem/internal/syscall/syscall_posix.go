// +build darwin freebsd linux

package syscall

import (
	"golang.org/x/sys/unix"
)

const (
	// AT_REMOVEDIR is the numeric representation of the AT_REMOVEDIR flag used
	// with the unlinkat system call.
	AT_REMOVEDIR = unix.AT_REMOVEDIR
)

// Symlinkat is a Go entry point for the symlinkat system call.
func Symlinkat(target string, directory int, path string) error {
	return unix.Symlinkat(target, directory, path)
}

// Readlinkat is a Go entry point for the readlinkat system call.
func Readlinkat(directory int, path string, buffer []byte) (int, error) {
	return unix.Readlinkat(directory, path, buffer)
}
