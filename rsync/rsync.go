package rsync

import (
	"bytes"
	"crypto/sha1"
	"hash"
	"io"

	"github.com/pkg/errors"

	"bitbucket.org/kardianos/rsync"
)

type OpType uint8

const (
	OpBlock OpType = iota
	OpData
	OpHash
	OpBlockRange
)

type Operation struct {
	Type          OpType
	BlockIndex    uint64
	BlockIndexEnd uint64
	Data          []byte
}

type BlockHash struct {
	Index      uint64
	StrongHash []byte
	WeakHash   uint32
}

type OperationTransmitter func(Operation) error

// OperationReceiver retrieves and returns the next operation in an operation
// stream. When there are no more operations, it should return an io.EOF error.
type OperationReceiver func() (Operation, error)

type Rsync struct {
	rsync *rsync.RSync
}

func New() *Rsync {
	return &Rsync{
		rsync: &rsync.RSync{
			UniqueHasher: sha1.New(),
		},
	}
}

func (r *Rsync) Signature(base io.Reader) ([]BlockHash, error) {
	// Create the result.
	var result []BlockHash

	// Create a signature writer.
	write := func(b rsync.BlockHash) error {
		result = append(result, BlockHash{
			Index:      b.Index,
			StrongHash: b.StrongHash,
			WeakHash:   b.WeakHash,
		})
		return nil
	}

	// Perform signature generation.
	if err := r.rsync.CreateSignature(base, write); err != nil {
		return nil, err
	}

	// Success.
	return result, nil
}

func (r *Rsync) BytesSignature(base []byte) ([]BlockHash, error) {
	return r.Signature(bytes.NewReader(base))
}

// TODO: Add a very important warning to this function that the operation (and
// its underlying data buffer) that are passed to the transmitter are re-used,
// so they need to be copied if retained.
func (r *Rsync) Deltafy(target io.Reader, baseSignature []BlockHash, transmit OperationTransmitter) error {
	// Convert the base signature.
	baseSignatureRsync := make([]rsync.BlockHash, len(baseSignature))
	for i, b := range baseSignature {
		baseSignatureRsync[i] = rsync.BlockHash{
			Index:      b.Index,
			StrongHash: b.StrongHash,
			WeakHash:   b.WeakHash,
		}
	}

	// Create a wrapper operation writer.
	write := func(o rsync.Operation) error {
		return transmit(Operation{
			Type:          OpType(o.Type),
			BlockIndex:    o.BlockIndex,
			BlockIndexEnd: o.BlockIndexEnd,
			Data:          o.Data,
		})
	}

	// Perform delta generation.
	return r.rsync.CreateDelta(target, baseSignatureRsync, write, nil)
}

func (r *Rsync) DeltafyBytes(target []byte, baseSignature []BlockHash) ([]Operation, error) {
	// Create an empty result.
	var delta []Operation

	// Create an operation transmitter to populate the result. Note that we copy
	// any operation data buffers because they are re-used.
	transmit := func(operation Operation) error {
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

	// Wrap up the bytes in a reader.
	reader := bytes.NewReader(target)

	// Compute the delta.
	if err := r.Deltafy(reader, baseSignature, transmit); err != nil {
		return nil, err
	}

	// Success.
	return delta, nil
}

func (r *Rsync) Patch(destination io.Writer, base io.ReadSeeker, receive OperationReceiver, digest hash.Hash) error {
	// Create channels to communicate with the ApplyDelta Goroutine.
	operations := make(chan rsync.Operation)
	applyErrors := make(chan error, 1)

	// Start the ApplyDelta operation in a separate Goroutine, recording the
	// hash of the received contents.
	go func() {
		applyErrors <- r.rsync.ApplyDelta(destination, base, operations, digest)
	}()

	// Receive and feed operations into the Goroutine, watching for errors.
	var applyError, receiveError error
	applyExited := false
	for {
		// Grab the next operation. We stop on any error, but io.EOF is an
		// acceptable error because it represents the end of the operation
		// stream.
		operation, err := receive()
		if err != nil {
			if err != io.EOF {
				receiveError = err
			}
			break
		}

		// Convert the operation.
		operationRsync := rsync.Operation{
			Type:          rsync.OpType(operation.Type),
			BlockIndex:    operation.BlockIndex,
			BlockIndexEnd: operation.BlockIndexEnd,
			Data:          operation.Data,
		}

		// Forward the operation while watching for errors.
		select {
		case operations <- operationRsync:
		case applyError = <-applyErrors:
			applyExited = true
			break
		}
	}

	// Tell the ApplyDelta Goroutine that operations are complete. It may have
	// exited already if there was an error, in which case this will have no
	// effect.
	close(operations)

	// Ensure that the ApplyDelta Goroutine has completed.
	if !applyExited {
		applyError = <-applyErrors
	}

	// Check for errors.
	if receiveError != nil {
		return errors.Wrap(receiveError, "unable to receive operation")
	} else if applyError != nil {
		return errors.Wrap(applyError, "unable to apply operation")
	}

	// Success.
	return nil
}

func (r *Rsync) PatchBytes(base []byte, delta []Operation, digest hash.Hash) ([]byte, error) {
	// Wrap up the base bytes in a reader.
	baseReader := bytes.NewReader(base)

	// Create an output buffer.
	output := bytes.NewBuffer(nil)

	// Create an operation receiver that will return delta operations.
	receive := func() (Operation, error) {
		// If there are operations remaining, return the next one and reduce.
		if len(delta) > 0 {
			result := delta[0]
			delta = delta[1:]
			return result, nil
		}

		// Otherwise we're done.
		return Operation{}, io.EOF
	}

	// Perform application.
	if err := r.Patch(output, baseReader, receive, digest); err != nil {
		return nil, err
	}

	// Success.
	return output.Bytes(), nil
}
