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

// NewListener creates a new IPC listener.
func NewListener(path string) (net.Listener, error) {
	// Create the listener.
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	// Explicitly set socket permissions.
	if err := os.Chmod(path, 0600); err != nil {
		listener.Close()
		return nil, errors.Wrap(err, "unable to set socket permissions")
	}

	// Create the listener.
	return listener, nil
}
