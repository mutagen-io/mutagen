package connectivity

import (
	"net"

	"github.com/pkg/errors"
)

type oneShotListener struct {
	connection net.Conn
}

func NewOneShotListener(connection net.Conn) net.Listener {
	return &oneShotListener{
		connection: connection,
	}
}

func (l *oneShotListener) Accept() (net.Conn, error) {
	// If a connection is present, nil out the record of it and return it.
	if l.connection != nil {
		connection := l.connection
		l.connection = nil
		return connection, nil
	}

	// If there are no connections, we're done.
	return nil, errors.New("no more connections")
}

func (l *oneShotListener) Close() error {
	// No accept calls ever block, so we don't need to do anything.
	return nil
}

func (l *oneShotListener) Addr() net.Addr {
	return newNamedAddress("memory")
}
