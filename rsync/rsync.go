package rsync

import (
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

type OperationTransmitter func(*Operation) error
type OperationReceiver func() (*Operation, error)

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

func (r *Rsync) Signature(base io.Reader) ([]*BlockHash, error) {
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
	if err := r.rsync.CreateSignature(base, write); err != nil {
		return nil, err
	}

	// Success.
	return result, nil
}

// TODO: Add a very important warning to this function that the operation (and
// its underlying data buffer) that are passed to the transmitter are re-used,
// so they need to be copied if retained.
func (r *Rsync) Deltafy(target io.Reader, baseSignature []*BlockHash, transmit OperationTransmitter) error {
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
	return r.rsync.CreateDelta(target, baseSignatureRsync, write, nil)
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
		return errors.Wrap(receiveError, "unable to receive operation")
	} else if applyError != nil {
		return errors.Wrap(applyError, "unable to apply operation")
	}

	// Success.
	return nil
}
