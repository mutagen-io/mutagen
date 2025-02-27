package rsync

import (
	"errors"
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"
)

// Transmit performs streaming transmission of files (in rsync deltified form)
// to the specified receiver. It is the responsibility of the caller to ensure
// that the provided signatures are valid by invoking their EnsureValid method.
// In order for this function to perform efficiently, paths should be passed in
// depth-first traversal order.
func Transmit(root string, paths []string, signatures []*Signature, receiver Receiver, logger *logging.Logger) error {
	// Ensure that the transmission request is sane.
	if len(paths) != len(signatures) {
		must.Succeed(receiver.finalize(), "finalize receiver", logger)
		return errors.New("number of paths does not match number of signatures")
	}

	// Create a file opener that we can use to safely open files, and defer its
	// closure.
	opener := filesystem.NewOpener(root, logger)
	defer must.Close(opener, logger)

	// Create an rsync engine.
	engine := NewEngine()

	// Create a transmission object that we can re-use to avoid allocating.
	transmission := &Transmission{}

	// Handle the requested files.
	for i, p := range paths {
		// Open the file and extract its size. Failure here is non-terminal, but
		// we need to inform the receiver. If sending the message fails, that is
		// a terminal error.
		file, metadata, err := opener.OpenFile(p)
		if err != nil {
			*transmission = Transmission{
				Done:  true,
				Error: fmt.Errorf("unable to open file: %w", err).Error(),
			}
			if err = receiver.Receive(transmission); err != nil {
				must.Succeed(receiver.finalize(), "finalize receiver", logger)
				return fmt.Errorf("unable to send error transmission: %w", err)
			}
			continue
		}
		fileSize := metadata.Size

		// Create an operation transmitter for deltification and track reception
		// errors. We can safely set transmitError on each call because as soon
		// as it's returned non-nil, the transmit function won't be called
		// again.
		var transmitError error
		transmit := func(o *Operation) error {
			*transmission = Transmission{ExpectedSize: fileSize, Operation: o}
			transmitError = receiver.Receive(transmission)
			fileSize = 0
			return transmitError
		}

		// Perform deltification.
		err = engine.Deltify(file, signatures[i], 0, transmit)

		// Close the file.
		must.Close(file, logger)

		// Handle any transmission errors. These are terminal.
		if transmitError != nil {
			must.Succeed(receiver.finalize(), "finalize receiver", logger)
			return fmt.Errorf("unable to transmit delta: %w", transmitError)
		}

		// Inform the client the operation stream for this file is complete. Any
		// internal (non-transmission) errors are non-terminal but should be
		// reported to the receiver.
		*transmission = Transmission{Done: true}
		if err != nil {
			transmission.Error = fmt.Errorf("engine error: %w", err).Error()
		}
		if err = receiver.Receive(transmission); err != nil {
			must.Succeed(receiver.finalize(), "finalize receiver", logger)
			return fmt.Errorf("unable to send done message: %w", err)
		}
	}

	// Ensure that the receiver is finalized.
	if err := receiver.finalize(); err != nil {
		return fmt.Errorf("unable to finalize receiver: %w", err)
	}

	// Success.
	return nil
}
