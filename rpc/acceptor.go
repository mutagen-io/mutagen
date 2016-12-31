package rpc

import (
	"net"
)

type Acceptor interface {
	Accept() (net.Conn, error)
}
