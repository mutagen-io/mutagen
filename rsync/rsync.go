package rsync

import (
	"crypto/sha1"
	"hash"
	"io"

	"bitbucket.org/kardianos/rsync"
)

type OperationTransmitter func(*Operation) error
type OperationReceiver func() (*Operation, error)

type Rsyncer struct {
	rsyncer *rsync.RSync
}

func NewRsyncer() *Rsyncer {
	return &Rsyncer{
		rsyncer: &rsync.RSync{
			UniqueHasher: sha1.New(),
		},
	}
}

func (r *Rsyncer) Signature(base io.Reader) ([]*BlockHash, error) {
	// Create the result.
	var result []*BlockHash

	// Create a signature writer.
	write := func(b rsync.BlockHash) error {
		result = append(result, &BlockHash{
			Index:      b.Index,
			StrongHash: b.StrongHash,
			WeakHash:   b.WeakHash,
		})
		return nil
	}

	// Perform signature generation.
	if err := r.rsyncer.CreateSignature(base, write); err != nil {
		return nil, err
	}

	// Success.
	return result, nil
}

func (r *Rsyncer) Deltafy(target io.Reader, baseSignature []*BlockHash, transmit OperationTransmitter) error {
	// Convert the base signature.
	baseSignatureRsync := make([]rsync.BlockHash, len(baseSignature))
	for i, b := range baseSignature {
		baseSignatureRsync[i] = rsync.BlockHash{
			Index:      b.Index,
			StrongHash: b.StrongHash,
			WeakHash:   b.WeakHash,
		}
	}

	// Create a wrapper operation writer. Re-use the Operation instance to save
	// on allocations. Rsync already re-uses the underlying data buffer, so
	// we're not making things any more dangerous.
	operation := &Operation{}
	write := func(o rsync.Operation) error {
		*operation = Operation{
			Type:          OpType(o.Type),
			BlockIndex:    o.BlockIndex,
			BlockIndexEnd: o.BlockIndexEnd,
			Data:          o.Data,
		}
		return transmit(operation)
	}

	// Perform delta generation.
	return r.rsyncer.CreateDelta(target, baseSignatureRsync, write, nil)
}

func (r *Rsyncer) Patch(destination io.Writer, base io.ReadSeeker, receive OperationReceiver, digest hash.Hash) error {
	// Create channels to communicate with the ApplyDelta Goroutine.
	operations := make(chan rsync.Operation)
	applyErrors := make(chan error, 1)

	// Start the ApplyDelta operation in a separate Goroutine, recording the
	// hash of the received contents.
	go func() {
		applyErrors <- r.rsyncer.ApplyDelta(destination, base, operations, digest)
	}()

	// Receive and feed operations into the Goroutine, watching for errors.
	var applyError, receiveError error
	applyExited := false
	for {
		// Grab the next operation.
		operation, err := receive()
		if err != nil {
			receiveError = err
			break
		}

		// Check if the operation stream is done.
		if operation == nil {
			break
		}

		// Convert the operation.
		operationRsync := rsync.Operation{
			Type:          rsync.OpType(operation.Type),
			BlockIndex:    operation.BlockIndex,
			BlockIndexEnd: operation.BlockIndexEnd,
			Data:          operation.Data,
		}

		// Forward the operation. If there is an error, burn the remaining
		// operations in this stream.
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
		return receiveError
	} else if applyError != nil {
		return applyError
	}

	// Success.
	return nil
}
