//go:build !windows

package ipc

import (
	"context"
	"fmt"
	"net"
	"os"
)

// DialContext attempts to establish an IPC connection, timing out if the
// provided context expires.
func DialContext(context context.Context, path string) (net.Conn, error) {
	// Create a zero-valued dialer, which will have the same dialing behavior as
	// the raw dialing functions.
	dialer := &net.Dialer{}

	// Perform dialing.
	return dialer.DialContext(context, "unix", path)
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
		return nil, fmt.Errorf("unable to set socket permissions: %w", err)
	}

	// Create the listener.
	return listener, nil
}
