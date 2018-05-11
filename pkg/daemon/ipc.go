package daemon

import (
	"net"
	"time"

	"github.com/havoc-io/mutagen/pkg/rpc"
)

const (
	openerDialTimeout = 1 * time.Second
)

type opener struct{}

func NewOpener() rpc.Opener {
	return &opener{}
}

func (o *opener) Open() (net.Conn, error) {
	return dialTimeout(openerDialTimeout)
}
