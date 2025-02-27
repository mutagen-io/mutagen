package multiplexing

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"
	"golang.org/x/net/nettest"
)

// makeNetTestMakePipe constructs a nettest.MakePipe with a pair of multiplexers
// operating in opener and acceptor roles.
func makeNetTestMakePipe(opener, acceptor *Multiplexer, logger *logging.Logger) nettest.MakePipe {
	return func() (c1, c2 net.Conn, stop func(), err error) {
		var wait sync.WaitGroup
		wait.Add(2)
		var opened, accepted *Stream
		var openErr, acceptErr error
		go func() {
			opened, openErr = opener.OpenStream(context.Background())
			if openErr == ErrMultiplexerClosed {
				if internalErr := opener.InternalError(); internalErr != nil {
					openErr = fmt.Errorf("multiplexer closed due to internal error: %w", internalErr)
				}
			}
			wait.Done()
		}()
		go func() {
			accepted, acceptErr = acceptor.AcceptStream(context.Background())
			if acceptErr == ErrMultiplexerClosed {
				if internalErr := acceptor.InternalError(); internalErr != nil {
					acceptErr = fmt.Errorf("multiplexer closed due to internal error: %w", internalErr)
				}
			}
			wait.Done()
		}()
		wait.Wait()
		if openErr != nil || acceptErr != nil {
			if opened != nil {
				must.Close(opened, logger)
			}
			if accepted != nil {
				must.Close(accepted, logger)
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
				must.Close(opened, logger)
				must.Close(accepted, logger)
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

	errBuf := bytes.Buffer{}
	logger := logging.NewLogger(logging.LevelError, &errBuf)
	// Perform multiplexing.
	p1Mux := Multiplex(p1Carrier, false, nil, logger)
	p2Mux := Multiplex(p2Carrier, true, nil, logger)

	// Defer multiplexer shutdown.
	defer func() {
		must.Close(p1Mux, logger)
		must.Close(p2Mux, logger)
	}()

	// Run tests from p1 to p2.
	nettest.TestConn(t, makeNetTestMakePipe(p1Mux, p2Mux, logger))

	// Run tests from p2 to p1.
	nettest.TestConn(t, makeNetTestMakePipe(p2Mux, p1Mux, logger))

	//TODO: Inspect errBuf here
}
