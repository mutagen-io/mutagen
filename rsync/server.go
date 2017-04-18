package rsync

import (
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/message"
)

func Serve(connection io.ReadWriter, root string) error {
	// Wrap the connection in a message stream.
	stream := message.NewCompressedMessageStream(connection)

	// Create an rsync engine.
	engine := NewEngine()

	// Server until there's an error.
	for {
		// Receive the next request. In theory, we could re-use request objects
		// for decoding, but in practice it'd be a pain to reset each block hash
		// within each signature in the request (which we'd have to since gob
		// won't transmit and set fields with zero values). We'd have to do this
		// for each block hash in the *capacity* of each signature, which would
		// be error prone and slow. Also, the block hash capacity for each
		// signature is most likely going to be insufficient to avoid
		// reallocation until many requests have been received and each
		// signature has gained a large block hash capacity.
		var request request
		if err := stream.Decode(&request); err != nil {
			return errors.Wrap(err, "unable to decode request")
		}

		// Verify that the request is sane.
		if err := request.ensureValid(); err != nil {
			return errors.Wrap(err, "received invalid request")
		}

		// Handle the requested transmissions.
		for i, p := range request.Paths {
			// Open the file. If this fails, it's a non-terminal error, but we
			// need to inform the client. If sending the response fails, that is
			// a terminal error.
			file, err := os.Open(filepath.Join(root, p))
			if err != nil {
				response := response{
					Done:  true,
					Error: errors.Wrap(err, "unable to open file").Error(),
				}
				if err = stream.Encode(response); err != nil {
					return errors.Wrap(err, "unable to send error response")
				}
				continue
			}

			// Create a transmitter for deltafication and track transmission
			// errors. We can set transmitError on each call because as soon as
			// it's returned non-nil, the transmit function won't be called
			// again.
			var transmitError error
			transmit := func(o Operation) error {
				transmitError = stream.Encode(response{Operation: o})
				return transmitError
			}

			// Perform deltafication.
			err = engine.Deltafy(file, request.Signatures[i], 0, transmit)

			// Close the file.
			file.Close()

			// Handle any transmission errors. These are terminal.
			if transmitError != nil {
				return errors.Wrap(transmitError, "unable to transmit delta")
			}

			// Inform the client the transmission is complete. Any internal
			// errors are non-terminal but should be reported.
			response := response{Done: true}
			if err != nil {
				response.Error = errors.Wrap(err, "engine error").Error()
			}
			if err = stream.Encode(response); err != nil {
				return errors.Wrap(err, "unable to send done response")
			}
		}
	}
}
