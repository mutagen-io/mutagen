//go:build !windows

package ipc

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"
)

// DialContext attempts to establish an IPC connection, timing out if the
// provided context expires.
func DialContext(ctx context.Context, path string) (net.Conn, error) {
	// Create a zero-valued dialer, which will have the same dialing behavior as
	// the raw dialing functions.
	dialer := &net.Dialer{}

	// Perform dialing.
	return dialer.DialContext(ctx, "unix", path)
}

// NewListener creates a new IPC listener.
func NewListener(path string, logger *logging.Logger) (net.Listener, error) {
	// Create the listener.
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	// Explicitly set socket permissions. Unfortunately we can't do this
	// atomically on socket creation, but we can do it quickly.
	if err := os.Chmod(path, 0600); err != nil {
		must.Close(listener, logger)
		return nil, fmt.Errorf("unable to set socket permissions: %w", err)
	}

	// Success.
	return listener, nil
}
