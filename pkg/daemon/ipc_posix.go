// +build !windows

package daemon

import (
	"net"
	"os"
	"time"

	"github.com/pkg/errors"
)

const (
	// socketName is the name of the UNIX domain socket used for daemon IPC. It
	// resides within the daemon subdirectory of the Mutagen directory.
	socketName = "daemon.sock"
)

// DialTimeout attempts to establish a daemon IPC connection, timing out after
// the specified duration.
func DialTimeout(timeout time.Duration) (net.Conn, error) {
	// Compute the socket path.
	socketPath, err := subpath(socketName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute socket path")
	}

	// Dial.
	return net.DialTimeout("unix", socketPath, timeout)
}

// NewListener creates a new daemon IPC listener.
func NewListener() (net.Listener, error) {
	// Compute the socket path.
	socketPath, err := subpath(socketName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute socket path")
	}

	// Remove the socket path if it exists. This is safe since the caller should
	// own the daemon lock. In general, the socket path will be cleaned up when
	// the listener is closed, but if there's a crash, we need to wipe it.
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "unable to remove stale socket")
	}

	// Create the listener.
	return net.Listen("unix", socketPath)
}
