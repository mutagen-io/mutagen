package daemon

import (
	"net"
	"time"

	"github.com/havoc-io/mutagen/stream"
)

const (
	openerDialTimeout = 1 * time.Second
)

type opener struct{}

func NewOpener() stream.Opener {
	return &opener{}
}

func (o *opener) Open() (net.Conn, error) {
	return dialTimeout(openerDialTimeout)
}
