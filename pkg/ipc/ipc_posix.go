// +build !windows

package ipc

import (
	"net"
	"os"
	"time"

	"github.com/pkg/errors"
)

// DialTimeout attempts to establish an IPC connection, timing out after the
// specified duration.
func DialTimeout(path string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", path, timeout)
}

// NewListener creates a new IPC listener. It will remove any existing endpoint,
// so an external mechanism should be used to coordinate the establishment of
// listeners.
func NewListener(path string) (net.Listener, error) {
	// Remove the socket path if it exists. In general, the socket path will be
	// cleaned up when a listener is closed, but if there's a crash, we need to
	// wipe it.
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "unable to remove stale socket")
	}

	// Create the listener.
	return net.Listen("unix", path)
}
