package session

import (
	"bytes"
	"crypto/sha1"
	"io"

	"github.com/havoc-io/mutagen/rsync"
)

func snapshotChecksum(snapshotBytes []byte) []byte {
	result := sha1.Sum(snapshotBytes)
	return result[:]
}

func snapshotChecksumMatch(snapshotBytes, expectedSnapshotChecksum []byte) bool {
	return bytes.Equal(snapshotChecksum(snapshotBytes), expectedSnapshotChecksum)
}

func snapshotSignature(baseSnapshotBytes []byte) ([]rsync.BlockHash, error) {
	// Create an rsyncer.
	rsyncer := rsync.New()

	// Wrap up the base snapshot bytes in a reader.
	base := bytes.NewReader(baseSnapshotBytes)

	// Compute the signature.
	return rsyncer.Signature(base)
}

func deltafySnapshot(
	snapshotBytes []byte,
	baseSnapshotSignature []rsync.BlockHash,
) ([]rsync.Operation, error) {
	// Create an empty result.
	var delta []rsync.Operation

	// Create an operation transmitter to populate the result. Note that we copy
	// any operation data buffers because the rsync package re-uses them.
	transmit := func(operation rsync.Operation) error {
		// Copy the operation's data buffer if necessary.
		if len(operation.Data) > 0 {
			dataCopy := make([]byte, len(operation.Data))
			copy(dataCopy, operation.Data)
			operation.Data = dataCopy
		}

		// Record the operation.
		delta = append(delta, operation)

		// Success.
		return nil
	}

	// Create an rsyncer.
	rsyncer := rsync.New()

	// Wrap up the snapshot bytes in a reader.
	reader := bytes.NewReader(snapshotBytes)

	// Compute the delta.
	if err := rsyncer.Deltafy(reader, baseSnapshotSignature, transmit); err != nil {
		return nil, err
	}

	// Success.
	return delta, nil
}

func patchSnapshot(baseSnapshotBytes []byte, delta []rsync.Operation) ([]byte, error) {
	// Create an rsyncer.
	rsyncer := rsync.New()

	// Wrap up the base snapshot bytes in a reader.
	base := bytes.NewReader(baseSnapshotBytes)

	// Create an output buffer.
	output := bytes.NewBuffer(nil)

	// Create an operation receiver that will return delta operations.
	receive := func() (rsync.Operation, error) {
		// If there are operations remaining, return the next one and reduce.
		if len(delta) > 0 {
			result := delta[0]
			delta = delta[1:]
			return result, nil
		}

		// Otherwise we're done.
		return rsync.Operation{}, io.EOF
	}

	// Perform application.
	if err := rsyncer.Patch(output, base, receive, nil); err != nil {
		return nil, err
	}

	// Success.
	return output.Bytes(), nil
}
