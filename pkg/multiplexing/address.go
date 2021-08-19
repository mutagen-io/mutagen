package multiplexing

import (
	"fmt"
)

// multiplexerAddress implements net.Addr for Multiplexer.
type multiplexerAddress struct {
	// even indicates whether or not this is the even-valued multiplexer.
	even bool
}

// Network implements net.Addr.Network.
func (a *multiplexerAddress) Network() string {
	return "multiplexed"
}

// String implements net.Addr.String.
func (a *multiplexerAddress) String() string {
	if a.even {
		return "multiplexer:even"
	}
	return "multiplexer:odd"
}

// streamAddress implements net.Addr for Stream.
type streamAddress struct {
	// remote indicates whether or not the address is remote.
	remote bool
	// identifier is the stream identifier.
	identifier uint64
}

// Network implements net.Addr.Network.
func (a *streamAddress) Network() string {
	return "multiplexed"
}

// String implements net.Addr.String.
func (a *streamAddress) String() string {
	if a.remote {
		return fmt.Sprintf("remote:%d", a.identifier)
	} else {
		return fmt.Sprintf("local:%d", a.identifier)
	}
}
