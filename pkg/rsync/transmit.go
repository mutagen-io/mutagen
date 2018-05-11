package rsync

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Transmit performs streaming transmission of files (in rsync deltafied form)
// to the specified receiver.
func Transmit(root string, paths []string, signatures []Signature, receiver Receiver) error {
	// Ensure that the receiver is finalized when we're done.
	defer receiver.finalize()

	// Ensure that the transmission request is sane.
	if len(paths) != len(signatures) {
		return errors.New("number of paths does not match number of signatures")
	}

	// Create an rsync engine.
	engine := NewEngine()

	// Handle the requested files.
	for i, p := range paths {
		// Open the file. If this fails, it's a non-terminal error, but we
		// need to inform the receiver. If sending the message fails, that is
		// a terminal error.
		file, err := os.Open(filepath.Join(root, p))
		if err != nil {
			message := Transmission{
				Done:  true,
				Error: errors.Wrap(err, "unable to open file").Error(),
			}
			if err = receiver.Receive(message); err != nil {
				return errors.Wrap(err, "unable to send error message")
			}
			continue
		}

		// Create an operation transmitter for deltafication and track reception
		// errors. We can safely set transmitError on each call because as soon
		// as it's returned non-nil, the transmit function won't be called
		// again.
		var transmitError error
		transmit := func(o Operation) error {
			transmitError = receiver.Receive(Transmission{Operation: o})
			return transmitError
		}

		// Perform deltafication.
		err = engine.Deltafy(file, signatures[i], 0, transmit)

		// Close the file.
		file.Close()

		// Handle any transmission errors. These are terminal.
		if transmitError != nil {
			return errors.Wrap(transmitError, "unable to transmit delta")
		}

		// Inform the client the operation stream for this file is complete. Any
		// internal (non-transmission) errors are non-terminal but should be
		// reported to the receiver.
		message := Transmission{Done: true}
		if err != nil {
			message.Error = errors.Wrap(err, "engine error").Error()
		}
		if err = receiver.Receive(message); err != nil {
			return errors.Wrap(err, "unable to send done message")
		}
	}

	// Success.
	return nil
}
