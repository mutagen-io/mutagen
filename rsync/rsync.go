package rsync

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"hash"
	"io"

	"github.com/pkg/errors"
)

// OperationTransmitter transmits an operation. Operation data buffers are
// re-used between calls to the transmitter (though other fields and the
// Operation itself are not re-used), so the transmitter should not return until
// it has either transmitted the data buffer (if any) or copied it for later
// transmission.
type OperationTransmitter func(*Operation) error

// OperationReceiver retrieves and returns the next operation in an operation
// stream. When there are no more operations, it should return an io.EOF error.
// Operations are processed between calls, so the receiver may re-used any and
// all parts of an operation message on subsequent calls.
type OperationReceiver func() (*Operation, error)

const (
	// defaultBlockSize is the default block size.
	defaultBlockSize = 10 * 1024
	// defaultMaxOpSize is the default maximum data operation size.
	defaultMaxOpSize = 5 * defaultBlockSize
)

const (
	// m is the weak hash modulus. I think they now recommend that it be the
	// largest prime less than 2^16, but this value is fine as well.
	m = 1 << 16
)

// Engine provides rsync functionality without any notion of transport.
type Engine struct {
	// blockSize is the rsync block size.
	blockSize uint64
	// maxOpSize is the maximum data buffer size that will be sent in an
	// operation.
	maxOpSize uint64
	// hasher is the hashing algorithm that will be used for strong hashes.
	hasher hash.Hash
	// buffer is a re-usable buffer that will be used for reading data and
	// setting up operations. It needs to have blockSize bytes for computing
	// signatures, blockSize + maxOpSize bytes for computing deltas, and
	// blockSize bytes for applying hashes.
	buffer []byte
}

// newEngine creates a new rsync engine with the specified parameters.
func newEngine(blockSize uint64, maxOpSize uint64, hasher hash.Hash) *Engine {
	// TODO: If we want to make this function public, it needs to enforce that
	// blockSize and maxOpSize are non-0, and I suppose that hasher is non-nil.
	return &Engine{
		blockSize: blockSize,
		maxOpSize: maxOpSize,
		hasher:    hasher,
		buffer:    make([]byte, blockSize+maxOpSize),
	}
}

// NewDefaultEngine creates a new rsync engine with sensible default parameters.
func NewDefaultEngine() *Engine {
	return newEngine(defaultBlockSize, defaultMaxOpSize, sha1.New())
}

// weakHash computes a fast checksum that can be rolled (updated without full
// recomputation). This particular hash is detailed on page 55 of Andrew
// Tridgell's rsync thesis (https://www.samba.org/~tridge/phd_thesis.pdf). It is
// not theoretically optimal, but it's fine for our purposes.
func (e *Engine) weakHash(data []byte) (uint32, uint32, uint32) {
	// Compute hash components.
	var r1, r2 uint32
	for i, b := range data {
		r1 += uint32(b)
		r2 += (uint32(e.blockSize) - uint32(i)) * uint32(b)
	}
	r1 = r1 % m
	r2 = r2 % m

	// Compute the hash.
	result := r1 + m*r2

	// Done.
	return result, r1, r2
}

// rollWeakHash updates the checksum computed by weakHash by adding and removing
// a byte.
func (e *Engine) rollWeakHash(r1, r2 uint32, out, in byte) (uint32, uint32, uint32) {
	// Update components.
	r1 = (r1 - uint32(out) + uint32(in)) % m
	r2 = (r2 - uint32(e.blockSize)*uint32(out) + r1) % m

	// Compute the hash.
	result := r1 + m*r2

	// Done.
	return result, r1, r2
}

func (e *Engine) strongHash(data []byte) []byte {
	// Reset the hasher.
	e.hasher.Reset()

	// Digest the data. Writes cannot fail on hash.Hash objects.
	e.hasher.Write(data)

	// Compute the sum.
	return e.hasher.Sum(nil)
}

func (e *Engine) Signature(base io.Reader) ([]*BlockHash, error) {
	// Create the result.
	var result []*BlockHash

	// Extract a portion of our buffer with which to read blocks.
	buffer := e.buffer[:e.blockSize]

	// Read blocks and append their hashes until we reach EOF.
	index := uint64(0)
	eof := false
	for !eof {
		// Read the next block and watch for errors. If we receive io.EOF, then
		// nothing was read, and we should break immediately. This means that
		// the base had a length that was a multiple of the block size. If we
		// receive io.ErrUnexpectedEOF, then something was read but we're still
		// at the end of the file, so we should hash this block but not go
		// through the loop again. Other errors are terminal.
		n, err := io.ReadFull(base, buffer)
		if err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF {
			eof = true
		} else if err != nil {
			return nil, errors.Wrap(err, "unable to read data block")
		}

		// Compute hashes for the block. Note that we don't assume we've
		// received a full block - we only hash the portion of the buffer that
		// was filled.
		weak, _, _ := e.weakHash(buffer[:n])
		strong := e.strongHash(buffer[:n])

		// Add the block hash.
		result = append(result, &BlockHash{index, weak, strong})

		// Increment the block index.
		index += 1
	}

	// Success.
	return result, nil
}

func (e *Engine) BytesSignature(base []byte) []*BlockHash {
	// Perform the signature and watch for errors (which shouldn't be able to
	// occur in-memory).
	result, err := e.Signature(bytes.NewReader(base))
	if err != nil {
		panic(errors.Wrap(err, "in-memory signature failure"))
	}

	// Success.
	return result
}

func (e *Engine) Deltafy(target io.Reader, baseSignature []*BlockHash, transmit OperationTransmitter) error {
	// Create a lookup table that maps weak hashes to all matching block hashes.
	// In the rsync technical report
	// (https://rsync.samba.org/tech_report/node4.html), they actually advocate
	// a 3-tier search (i.e. an additional 16-bit hash layer before the weak
	// hash), but I think this probably isn't necessary with modern hashing
	// algorithms and hardware.
	weakToBlockHashes := make(map[uint32][]*BlockHash, len(baseSignature))
	for _, h := range baseSignature {
		weakToBlockHashes[h.Weak] = append(weakToBlockHashes[h.Weak], h)
	}

	// Create a function that will coalesce block operations before sending. It
	// is a simple wrapper around transmit and shares its error semantics. For
	// data operations it will send any pending coalesced block operation and
	// then send the data operation immediately. This function relies on a check
	// below (just before return) to send the last pending operation, if any.
	var pendingOperation *Operation
	coalesce := func(operation *Operation) error {
		if len(operation.Data) > 0 {
			// Data operations aren't coalesced, so send and clear the pending
			// block operation, if any, and then send the data operation.
			if pendingOperation != nil {
				if err := transmit(pendingOperation); err != nil {
					return err
				}
			}
			pendingOperation = nil
			return transmit(operation)
		} else if pendingOperation != nil {
			// There is a pending block operation, check if we can coalesce.
			if pendingOperation.Start+pendingOperation.Count == operation.Start {
				// We can coalesce with the previous operation.
				pendingOperation.Count += operation.Count
				return nil
			} else {
				// We can't coalesce because the previous operation isn't
				// adjacent, so send the previous operation and mark this
				// one as pending.
				if err := transmit(pendingOperation); err != nil {
					return err
				}
				pendingOperation = operation
				return nil
			}
		} else {
			// There is no previous operation, so just record this one for
			// future coalescing.
			pendingOperation = operation
			return nil
		}
	}

	// Wrap the target in a buffered reader so that we can perform efficient
	// single-byte reads.
	bufferedTarget := bufio.NewReader(target)

	// Extract a portion of our buffer that we can use to store data while
	// searching for matches.
	buffer := e.buffer[:e.maxOpSize+e.blockSize]

	// Read in a block's worth of data for the initial match test.
	if n, err := io.ReadFull(bufferedTarget, buffer[:e.blockSize]); err == io.EOF {
		// The target is zero-length, so we don't need to send any operation.
		return nil
	} else if err == io.ErrUnexpectedEOF {
		// The target doesn't even have a block's worth of data, so transmit its
		// contents directly. We don't bother using the coalescer since we'll
		// only have a single operation and it'll be a data operation.
		if err := transmit(&Operation{Data: buffer[:n]}); err != nil {
			return errors.Wrap(err, "unable to send short target data")
		}
		return nil
	} else if err != nil {
		return errors.Wrap(err, "unable to perform initial buffer fill")
	}

	// Record the initial buffer occupancy.
	occupancy := e.blockSize

	// Compute the initial weak hash and its parameters.
	weak, r1, r2 := e.weakHash(buffer[:e.blockSize])

	// Loop until we've searched the entire target for matches. At the start of
	// each iteration of this loop, we know that occupancy >= blockSize, and the
	// values of weak, r1, and r2 correspond to the block at the end of the
	// occupied bytes.
	for {
		// Look for a block match for the block at the end of the buffer.
		potentials := weakToBlockHashes[weak]
		match := false
		var matchIndex uint64
		if len(potentials) > 0 {
			strong := e.strongHash(buffer[occupancy-e.blockSize : occupancy])
			for _, p := range potentials {
				if bytes.Equal(p.Strong, strong) {
					match = true
					matchIndex = p.Index
					break
				}
			}
		}

		// Handle the case where there's a match.
		if match {
			// If there's any data before the block at the end of the buffer, we
			// need to send it immediately.
			if occupancy > e.blockSize {
				if err := coalesce(&Operation{Data: buffer[:occupancy-e.blockSize]}); err != nil {
					return errors.Wrap(err, "unable to transmit data before match")
				}
			}

			// Transmit the operation for the block match.
			if err := coalesce(&Operation{Start: matchIndex, Count: 1}); err != nil {
				return errors.Wrap(err, "unable to transmit match")
			}

			// At this point, all of the data in the buffer has been handled, so
			// it has an effective occupancy of 0. Attempt to refill the buffer
			// with a single block. If we see an EOF error of any type, then we
			// won't have received blockSize bytes, so we won't have even a full
			// block in the buffer and are thus done searching for matches.
			n, err := io.ReadFull(bufferedTarget, buffer[:e.blockSize])
			occupancy = uint64(n)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			} else if err != nil {
				return errors.Wrap(err, "unable to refill buffer after match")
			}

			// Recompute the weak hash and its parameters.
			weak, r1, r2 = e.weakHash(buffer[:e.blockSize])

			// Check for matches again.
			continue
		}

		// Check if the buffer is full. If it is, then the data preceeding the
		// block will have the maximum allowed transmission size.
		if occupancy == uint64(len(buffer)) {
			// Send the data preceeding the block.
			if err := coalesce(&Operation{Data: buffer[:occupancy-e.blockSize]}); err != nil {
				return errors.Wrap(err, "unable to transmit data before truncation")
			}

			// Move the block at the end to the beginning and update occupancy.
			// The weak hash and its parameters still correspond to this block,
			// so they remain unchanged.
			copy(buffer[:e.blockSize], buffer[occupancy-e.blockSize:occupancy])
			occupancy = e.blockSize
		}

		// Read the next byte from the target, watching for errors. If we reach
		// the end of the file, then we're done looking for matches.
		b, err := bufferedTarget.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			return errors.Wrap(err, "unable to read target byte")
		}

		// Roll the weak hash, add the byte to the buffer, and update occupancy.
		weak, r1, r2 = e.rollWeakHash(r1, r2, buffer[occupancy-e.blockSize], b)
		buffer[occupancy] = b
		occupancy += 1
	}

	// We're done looking for matches (there can't be any more), so send the
	// remaining data in the buffer as data operations of the maximum allowed
	// size (we might have more than the maximum data operation size worth of
	// data in the buffer at this point).
	for occupancy > 0 {
		// Compute the size of the next operation.
		sendSize := occupancy
		if sendSize > e.maxOpSize {
			sendSize = e.maxOpSize
		}

		// Perform the send.
		if err := coalesce(&Operation{Data: buffer[:sendSize]}); err != nil {
			return errors.Wrap(err, "unable to send remaining data")
		}

		// Truncate the buffer. At this point we can just reduce the buffer
		// slice at each iteration of the loop rather than tracking its start
		// index.
		occupancy -= sendSize
		buffer = buffer[sendSize:]
	}

	// Check if there is a remaining block operation that wasn't coalesced, and
	// send it if so.
	if pendingOperation != nil {
		if err := transmit(pendingOperation); err != nil {
			return errors.Wrap(err, "unable to transmit final operation")
		}
	}

	// Success.
	return nil
}

func (e *Engine) DeltafyBytes(target []byte, baseSignature []*BlockHash) []*Operation {
	// Create an empty result.
	var delta []*Operation

	// Create an operation transmitter to populate the result. Note that we copy
	// any operation data buffers because they are re-used.
	transmit := func(operation *Operation) error {
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

	// Compute the delta and watch for errors (which shouldn't occur for for
	// in-memory data).
	if err := e.Deltafy(reader, baseSignature, transmit); err != nil {
		panic(errors.Wrap(err, "in-memory deltafication failure"))
	}

	// Success.
	return delta
}

func (e *Engine) Patch(destination io.Writer, base io.ReadSeeker, receive OperationReceiver, digest hash.Hash) error {
	// Extract a buffer for block copying.
	buffer := e.buffer[:e.blockSize]

	// If a digest has been provided, fold it into the destination. Since the
	// digest writes can't fail, there's no danger in doing this.
	if digest != nil {
		destination = io.MultiWriter(destination, digest)
	}

	// Loop until the operation stream is finished or errored.
	for {
		// Grab the next operation, watching for completion or errors.
		operation, err := receive()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return errors.Wrap(err, "unable to receive operation")
		} else if operation == nil {
			return errors.New("nil operation received")
		}

		// Handle the operation based on type.
		if len(operation.Data) > 0 {
			// Write data operations directly to the destination.
			if _, err = destination.Write(operation.Data); err != nil {
				return errors.Wrap(err, "unable to write data")
			}
		} else if operation.Count == 0 {
			// Otherwise we assume this is a block operation, so ensure it has a
			// sensible block count.
			return errors.New("received zero-length block operation")
		} else {
			// Seek to the start of the first requested block in base.
			// TODO: We should technically validate that operation.Start
			// multiplied by the block size can't overflow an int64. Worst case
			// at the moment it will cause the seek operation to fail.
			if _, err = base.Seek(int64(operation.Start)*int64(e.blockSize), io.SeekStart); err != nil {
				return errors.Wrap(err, "unable to seek to base location")
			}

			// Copy the requested number of blocks.
			for c := uint64(0); c < operation.Count; c++ {
				// Read the block. Watch for errors and make sure we only write
				// the subset of the buffer that actually read into. The only
				// error we allow here is io.ErrUnexpectedEOF, and only on when
				// we're copying the last requested block, because that
				// (probably) means we're trying to read the last block of the
				// base and the base didn't have a length that was a multiple of
				// the block size. If we aren't at the last requested block,
				// then the sender thought the base was longer than it actually
				// is, so that's an error. Also, io.EOF isn't acceptable,
				// because with io.ReadFull it means we were trying to read a
				// block just after the end of the base, which is likewise
				// invalid.
				var data []byte
				if n, err := io.ReadFull(base, buffer); err == nil {
					data = buffer
				} else if err == io.ErrUnexpectedEOF && c == (operation.Count-1) {
					data = buffer[:n]
				} else {
					return errors.Wrap(err, "unable to read base block")
				}

				// Write the block.
				if _, err = destination.Write(data); err != nil {
					return errors.Wrap(err, "unable to write data")
				}
			}
		}
	}
}

func (e *Engine) PatchBytes(base []byte, delta []*Operation, digest hash.Hash) ([]byte, error) {
	// Wrap up the base bytes in a reader.
	baseReader := bytes.NewReader(base)

	// Create an output buffer.
	output := bytes.NewBuffer(nil)

	// Create an operation receiver that will return delta operations.
	receive := func() (*Operation, error) {
		// If there are operations remaining, return the next one and reduce.
		if len(delta) > 0 {
			result := delta[0]
			delta = delta[1:]
			return result, nil
		}

		// Otherwise we're done.
		return nil, io.EOF
	}

	// Perform application.
	if err := e.Patch(output, baseReader, receive, digest); err != nil {
		return nil, err
	}

	// Success.
	return output.Bytes(), nil
}
