package stream

import (
	"io"
	"net"

	"github.com/inconshreveable/muxado"
)

const (
	multiplexerAcceptBacklog = 100
)

type Multiplexer interface {
	Open() (net.Conn, error)
	Accept() (net.Conn, error)
	Close() error
}

func Multiplex(connection io.ReadWriteCloser, server bool) Multiplexer {
	// Create configuration.
	configuration := &muxado.Config{
		AcceptBacklog: multiplexerAcceptBacklog,
	}

	// Create the session.
	if server {
		return muxado.Server(connection, configuration)
	} else {
		return muxado.Client(connection, configuration)
	}
}
