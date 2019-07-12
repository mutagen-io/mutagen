package rsync

import (
	"github.com/pkg/errors"

	fs "github.com/havoc-io/mutagen/pkg/filesystem"
)

// Transmit performs streaming transmission of files (in rsync deltafied form)
// to the specified receiver. It is the responsibility of the caller to ensure
// that the provided signatures are valid by invoking their EnsureValid method.
// In order for this function to perform efficiently, paths should be passed in
// depth-first traversal order.
func Transmit(root string, paths []string, signatures []*Signature, receiver Receiver) error {
	// Ensure that the transmission request is sane.
	if len(paths) != len(signatures) {
		receiver.finalize()
		return errors.New("number of paths does not match number of signatures")
	}

	// Create a file opener that we can use to safely open files, and defer its
	// closure.
	opener := fs.NewOpener(root)
	defer opener.Close()

	// Create an rsync engine.
	engine := NewEngine()

	// Create a transmission object that we can re-use to avoid allocating.
	transmission := &Transmission{}

	// Handle the requested files.
	for i, p := range paths {
		// Open the file. If this fails, it's a non-terminal error, but we
		// need to inform the receiver. If sending the message fails, that is
		// a terminal error.
		file, err := opener.Open(p)
		if err != nil {
			*transmission = Transmission{
				Done:  true,
				Error: errors.Wrap(err, "unable to open file").Error(),
			}
			if err = receiver.Receive(transmission); err != nil {
				receiver.finalize()
				return errors.Wrap(err, "unable to send error transmission")
			}
			continue
		}

		// Create an operation transmitter for deltafication and track reception
		// errors. We can safely set transmitError on each call because as soon
		// as it's returned non-nil, the transmit function won't be called
		// again.
		var transmitError error
		transmit := func(o *Operation) error {
			*transmission = Transmission{Operation: o}
			transmitError = receiver.Receive(transmission)
			return transmitError
		}

		// Perform deltafication.
		err = engine.Deltafy(file, signatures[i], 0, transmit)

		// Close the file.
		file.Close()

		// Handle any transmission errors. These are terminal.
		if transmitError != nil {
			receiver.finalize()
			return errors.Wrap(transmitError, "unable to transmit delta")
		}

		// Inform the client the operation stream for this file is complete. Any
		// internal (non-transmission) errors are non-terminal but should be
		// reported to the receiver.
		*transmission = Transmission{Done: true}
		if err != nil {
			transmission.Error = errors.Wrap(err, "engine error").Error()
		}
		if err = receiver.Receive(transmission); err != nil {
			receiver.finalize()
			return errors.Wrap(err, "unable to send done message")
		}
	}

	// Ensure that the receiver is finalized.
	if err := receiver.finalize(); err != nil {
		return errors.Wrap(err, "unable to finalize receiver")
	}

	// Success.
	return nil
}
