package rpc

import (
	"net"
)

type Opener interface {
	Open() (net.Conn, error)
}
