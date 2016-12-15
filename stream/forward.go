package stream

import (
	"io"

	"github.com/pkg/errors"
)

func Forward(incoming Acceptor, outgoing Opener) error {
	// Forward streams until there is an error.
	for {
		// Accept the next stream.
		accepted, err := incoming.Accept()
		if err != nil {
			return errors.Wrap(err, "unable to accept incoming stream")
		}

		// Open an outgoing stream.
		opened, err := outgoing.Open()
		if err != nil {
			accepted.Close()
			return errors.Wrap(err, "unable to open outgoing stream")
		}

		// Start forwarding in separate Goroutines.
		go io.Copy(opened, accepted)
		go io.Copy(accepted, opened)
	}
}
