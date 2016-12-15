package session

import (
	"io"

	"github.com/inconshreveable/muxado"

	"github.com/havoc-io/mutagen/stream"
)

const (
	multiplexerAcceptBacklog = 100
)

type multiplexer interface {
	stream.Opener
	stream.Acceptor
	io.Closer
}

func multiplex(connection io.ReadWriteCloser, server bool) multiplexer {
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
