package stream

import (
	"io"
	"net"

	"github.com/inconshreveable/muxado"
)

type Multiplexer interface {
	Open() (net.Conn, error)
	Accept() (net.Conn, error)
	Close() error
}

func Multiplex(connection io.ReadWriteCloser, server bool) Multiplexer {
	if server {
		return muxado.Server(connection, nil)
	} else {
		return muxado.Client(connection, nil)
	}
}
