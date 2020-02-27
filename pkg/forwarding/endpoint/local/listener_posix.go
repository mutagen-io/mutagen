// +build !windows

package local

import (
	"errors"
	"net"
	"syscall"
)

// listenWindowsNamedPipe returns an "unsupported" error on POSIX systems.
func listenWindowsNamedPipe(_ string) (net.Listener, error) {
	return nil, errors.New("Windows named pipes not supported on POSIX systems")
}

// isConflictingSocket returns whether or not a Unix domain socket listening
// error is due to a conflicting socket.
func isConflictingSocket(err error) bool {
	// On POSIX systems, both of these errors are possible depending on the
	// nature of the conflicting on-disk object and (if it's a socket) whether
	// or not a listener is currently bound to it.
	return errors.Is(err, syscall.EEXIST) || errors.Is(err, syscall.EADDRINUSE)
}
