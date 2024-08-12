package multiplexing

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"

	"golang.org/x/net/nettest"
)

// makeNetTestMakePipe constructs a nettest.MakePipe with a pair of multiplexers
// operating in opener and acceptor roles.
func makeNetTestMakePipe(opener, acceptor *Multiplexer) nettest.MakePipe {
	return func() (c1, c2 net.Conn, stop func(), err error) {
		var wait sync.WaitGroup
		wait.Add(2)
		var opened, accepted *Stream
		var openErr, acceptErr error
		go func() {
			opened, openErr = opener.OpenStream(context.Background())
			if errors.Is(openErr, ErrMultiplexerClosed) {
				if internalErr := opener.InternalError(); internalErr != nil {
					openErr = fmt.Errorf("multiplexer closed due to internal error: %w", internalErr)
				}
			}
			wait.Done()
		}()
		go func() {
			accepted, acceptErr = acceptor.AcceptStream(context.Background())
			if errors.Is(acceptErr, ErrMultiplexerClosed) {
				if internalErr := acceptor.InternalError(); internalErr != nil {
					acceptErr = fmt.Errorf("multiplexer closed due to internal error: %w", internalErr)
				}
			}
			wait.Done()
		}()
		wait.Wait()
		if openErr != nil || acceptErr != nil {
			if opened != nil {
				opened.Close()
			}
			if accepted != nil {
				accepted.Close()
			}
			if openErr != nil {
				err = openErr
			} else if acceptErr != nil {
				err = acceptErr
			}
			stop = func() {}
		} else {
			c1 = opened
			c2 = accepted
			stop = func() {
				opened.Close()
				accepted.Close()
			}
		}
		return
	}
}

// TestMultiplexer tests Multiplexer.
func TestMultiplexer(t *testing.T) {
	// Create an in-memory pipe to multiplex.
	p1, p2 := net.Pipe()

	// Adapt the connections to serve as carriers.
	p1Carrier := NewCarrierFromStream(p1)
	p2Carrier := NewCarrierFromStream(p2)

	// Perform multiplexing.
	p1Mux := Multiplex(p1Carrier, false, nil)
	p2Mux := Multiplex(p2Carrier, true, nil)

	// Defer multiplexer shutdown.
	defer func() {
		p1Mux.Close()
		p2Mux.Close()
	}()

	// Run tests from p1 to p2.
	nettest.TestConn(t, makeNetTestMakePipe(p1Mux, p2Mux))

	// Run tests from p2 to p1.
	nettest.TestConn(t, makeNetTestMakePipe(p2Mux, p1Mux))
}
