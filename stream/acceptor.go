package stream

import (
	"net"
)

type Acceptor interface {
	Accept() (net.Conn, error)
}
