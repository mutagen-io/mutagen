package daemon

import (
	"net"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/ipc"
)

// DialTimeout attempts to establish a connection to the daemon IPC endpoint.
func DialTimeout(timeout time.Duration) (net.Conn, error) {
	// Compute the path to the daemon IPC endpoint.
	endpoint, err := ipcEndpointPath()
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute endpoint path")
	}

	// Attempt to dial.
	return ipc.DialTimeout(endpoint, timeout)
}

// NewListener attempts to create a daemon IPC listener. It must only be called
// by a process that holds the daemon lock, because it will attempt to remove
// stale IPC listeners.
func NewListener() (net.Listener, error) {
	// Compute the path to the daemon IPC endpoint.
	endpoint, err := ipcEndpointPath()
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute endpoint path")
	}

	// Attempt to create an IPC listener. If this fails due to the endpoint
	// already existing, then attempt to remove the endpoint since we hold the
	// daemon lock and thus the endpoint is (or should be) stale.
	listener, err := ipc.NewListener(endpoint)
	if err != nil && os.IsExist(err) && os.Remove(endpoint) == nil {
		listener, err = ipc.NewListener(endpoint)
	}
	return listener, err
}
