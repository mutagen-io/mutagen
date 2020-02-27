// +build !windows

package local

import (
	"context"
	"errors"
	"net"
)

// dialNamedPipe returns an "unsupported" error on POSIX systems.
func dialNamedPipe(_ context.Context, _ string) (net.Conn, error) {
	return nil, errors.New("Windows named pipes not supported on POSIX systems")
}
