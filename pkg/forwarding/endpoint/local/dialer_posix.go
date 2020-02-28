// +build !windows

package local

import (
	"context"
	"errors"
	"net"
)

// dialWindowsNamedPipe returns an "unsupported" error on POSIX systems.
func dialWindowsNamedPipe(_ context.Context, _ string) (net.Conn, error) {
	return nil, errors.New("Windows named pipes not supported on POSIX systems")
}
