package connectivity

import (
	"net"
)

type namedAddress struct {
	name string
}

func newNamedAddress(name string) net.Addr {
	return &namedAddress{name}
}

func (a *namedAddress) Network() string {
	return a.name
}

func (a *namedAddress) String() string {
	return a.name
}
