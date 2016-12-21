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

		// Start forwarding in separate Goroutines. Close both streams once
		// forwarding fails in either direction. This is okay to do since we're
		// using net.Conn objects which will unblock reads/writes on close.
		go func() {
			errors := make(chan error, 2)
			go func() {
				_, err := io.Copy(opened, accepted)
				errors <- err
			}()
			go func() {
				_, err := io.Copy(accepted, opened)
				errors <- err
			}()
			<-errors
			accepted.Close()
			opened.Close()
		}()
	}
}
