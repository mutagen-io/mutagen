package rsync

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"hash"
	"io"

	"github.com/pkg/errors"
)

type BlockHash struct {
	Index  uint64
	Weak   uint32
	Strong [sha1.Size]byte
}

type Operation struct {
	// Data encodes raw data for data operations.
	Data []byte
	// Start encodes the 0-indexed starting block for block operations.
	Start uint64
	// Count encodes the number of blocks to copy in block operations.
	Count uint64
}

type OperationTransmitter func(Operation) error

// OperationReceiver retrieves and returns the next operation in an operation
// stream. When there are no more operations, it should return an io.EOF error.
type OperationReceiver func() (Operation, error)

// TODO: If we want to make these configurable, we need to enforce a few things.
// For one, the maximum data size needs to be larger than the block size given
// how our delta generation works. There may be other constraints as well.
const (
	blockSize       = 10 * 1024
	maximumDataSize = 10 * blockSize
)

const (
	// m is the weak hash modulus. I think they now recommend that it be the
	// largest prime less than 2^16, but this value is fine as well.
	m = 1 << 16
)

// weakHash computes a fast checksum that can be rolled (updated without full
// recomputation). This particular hash is detailed on page 55 of Andrew
// Tridgell's rsync thesis (https://www.samba.org/~tridge/phd_thesis.pdf). It is
// not theoretically optimal, but it's fine for our purposes.
func weakHash(data []byte) (uint32, uint32, uint32) {
	// Compute hash components.
	var r1, r2 uint32
	for i, b := range data {
		r1 += uint32(b)
		r2 += (uint32(blockSize) - uint32(i)) * uint32(b)
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
func rollWeakHash(r1, r2 uint32, out, in byte) (uint32, uint32, uint32) {
	// Update components.
	r1 = (r1 - uint32(out) + uint32(in)) % m
	r2 = (r2 - uint32(blockSize)*uint32(out) + r1) % m

	// Compute the hash.
	result := r1 + m*r2

	// Done.
	return result, r1, r2
}

func strongHash(data []byte) [sha1.Size]byte {
	return sha1.Sum(data)
}

func Signature(base io.Reader) ([]BlockHash, error) {
	// Create our hashing buffer on the stack.
	var bufferStorage [blockSize]byte
	buffer := bufferStorage[:]

	// Create the result.
	var result []BlockHash

	// Read blocks and append their hashes until we reach EOF.
	index := uint64(0)
	eof := false
	for !eof {
		// Read the next block and watch for errors. If we receive io.EOF, then
		// nothing was read, and we should break immediately. This means that
		// the base had a length that was a multiple of the block size. If we
		// receive io.ErrUnexpectedEOF, then something was read, but we're still
		// at the end of the base. Thus we should hash this block but not go
		// through the loop again. Other errors are terminal.
		n, err := io.ReadFull(base, buffer)
		if err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF {
			eof = true
		} else {
			return nil, errors.Wrap(err, "unable to read data block")
		}

		// Compute the weak hash. If the buffer is less than a full block
		// length (i.e. it's at the end of the data), then the weak hash won't
		// really be valid or rollable, but it doesn't really matter.
		weak, _, _ := weakHash(buffer[:n])

		// Compute the strong hash.
		strong := strongHash(buffer[:n])

		// Add the block hash.
		result = append(result, BlockHash{index, weak, strong})

		// Increment the block index.
		index += 1
	}

	// Success.
	return result, nil
}

func BytesSignature(base []byte) []BlockHash {
	// Perform the signature and watch for errors (which shouldn't be able to
	// occur in-memory).
	result, err := Signature(bytes.NewReader(base))
	if err != nil {
		panic(errors.Wrap(err, "in-memory signature failure"))
	}

	// Success.
	return result
}

func Deltafy(target io.Reader, baseSignature []BlockHash, transmit OperationTransmitter) error {
	// Create a lookup table that maps weak hashes to all matching block hashes.
	// In the rsync technical report
	// (https://rsync.samba.org/tech_report/node4.html), they actually advocate
	// a 3-tier search (i.e. an additional 16-bit hash layer before the weak
	// hash), but I think this probably isn't necessary with modern hashing
	// algorithms and hardware.
	weakToBlockHashes := make(map[uint32][]BlockHash, len(baseSignature))
	for _, h := range baseSignature {
		weakToBlockHashes[h.Weak] = append(weakToBlockHashes[h.Weak], h)
	}

	// Create a function that will coalesce block operations before sending. It
	// is a simple wrapper around transmit and shares its error semantics. For
	// data operations it will send any pending coalesced block operation and
	// then send the data operation immediately. This function relies on a check
	// below (just before return) to send the last pending operation, if any.
	var pendingOperation Operation
	coalesce := func(operation Operation) error {
		// Handle the operation based on type.
		if len(operation.Data) > 0 {
			// Data operations aren't coalesced, so send and clear the pending
			// block operation, if any, and then send the data operation.
			if pendingOperation.Count > 0 {
				if err := transmit(pendingOperation); err != nil {
					return err
				}
			}
			pendingOperation = Operation{}
			return transmit(operation)
		} else {
			// Check if there is a pending block operation.
			if pendingOperation.Count > 0 {
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
	}

	// Wrap the target in a buffered reader so that we can perform efficient
	// single-byte reads.
	bufferedTarget := bufio.NewReader(target)

	// Create a buffer on the stack that we can use to store data while
	// searching for matches.
	var bufferStorage [maximumDataSize + blockSize]byte
	buffer := bufferStorage[:]

	// Read in a block's worth of data for the initial match test.
	if n, err := io.ReadFull(bufferedTarget, buffer[:blockSize]); err == io.EOF {
		// The target is zero-length, so we don't need to send any operation.
		return nil
	} else if err == io.ErrUnexpectedEOF {
		// The target doesn't even have a block's worth of data, so transmit its
		// contents directly. We don't bother using the coalescer since we'll
		// only have a single operation and it'll be a data operation.
		if err := transmit(Operation{Data: buffer[:n]}); err != nil {
			return errors.Wrap(err, "unable to send short target data")
		}
		return nil
	} else if err != nil {
		return errors.Wrap(err, "unable to perform initial buffer fill")
	}

	// Record the initial buffer occupancy.
	occupancy := blockSize

	// Compute the initial weak hash and its parameters.
	weak, r1, r2 := weakHash(buffer[:blockSize])

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
			strong := strongHash(buffer[occupancy-blockSize : occupancy])
			for _, p := range potentials {
				if p.Strong == strong {
					match = true
					matchIndex = p.Index
					break
				}
			}
		}

		// Handle matches.
		if match {
			// If there's any data before the block at the end of the buffer, we
			// need to send it immediately.
			if occupancy > blockSize {
				if err := coalesce(Operation{Data: buffer[:occupancy-blockSize]}); err != nil {
					return errors.Wrap(err, "unable to transmit data before match")
				}
			}

			// Transmit the operation for the block match.
			if err := coalesce(Operation{Start: matchIndex, Count: 1}); err != nil {
				return errors.Wrap(err, "unable to transmit match")
			}

			// Attempt to refill the buffer with a single block. If we see an
			// EOF error, then we won't have received blockSize bytes, so we
			// won't have even a full block in the buffer and are thus done
			// searching for matches.
			n, err := io.ReadFull(bufferedTarget, buffer[:blockSize])
			occupancy = n
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			} else if err != nil {
				return errors.Wrap(err, "unable to refill buffer after match")
			}

			// Recompute the weak hash and its parameters.
			weak, r1, r2 = weakHash(buffer[:blockSize])

			// Check for matches again.
			continue
		}

		// Check if the buffer is full. If it is, then the data preceeding the
		// block will have the maximum allowed transmission size.
		if occupancy == len(buffer) {
			// Send the data preceeding the block.
			if err := coalesce(Operation{Data: buffer[:occupancy-blockSize]}); err != nil {
				return errors.Wrap(err, "unable to transmit data before truncation")
			}

			// Move the block at the end to the beginning and update occupancy.
			// The weak hash and its parameters still correspond to this block,
			// so they remain unchanged.
			copy(buffer[:blockSize], buffer[occupancy-blockSize:occupancy])
			occupancy = blockSize
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
		weak, r1, r2 = rollWeakHash(r1, r2, buffer[occupancy-blockSize], b)
		buffer[occupancy] = b
		occupancy += 1
	}

	// We're done looking for matches (there can't be any more), so send the
	// remaining data in the buffer as data operations of the maximum allowed
	// size.
	for occupancy > 0 {
		// Compute the size of the next operation.
		sendSize := occupancy
		if sendSize > maximumDataSize {
			sendSize = maximumDataSize
		}

		// Perform the send.
		if err := coalesce(Operation{Data: buffer[:sendSize]}); err != nil {
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
	if pendingOperation.Count > 0 {
		if err := transmit(pendingOperation); err != nil {
			return errors.Wrap(err, "unable to transmit final operation")
		}
	}

	// Success.
	return nil
}

func DeltafyBytes(target []byte, baseSignature []BlockHash) []Operation {
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

	// Compute the delta and watch for errors (which shouldn't occur for for
	// in-memory data).
	if err := Deltafy(reader, baseSignature, transmit); err != nil {
		panic(errors.Wrap(err, "in-memory deltafication failure"))
	}

	// Success.
	return delta
}

func Patch(destination io.Writer, base io.ReadSeeker, receive OperationReceiver, digest hash.Hash) error {
	// Create our block copying buffer on the stack.
	var bufferStorage [blockSize]byte
	buffer := bufferStorage[:]

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
			// TODO: We should technically validate that operation.Start can't
			// overflow an int64. Worst case at the moment it will cause the
			// seek operation to fail.
			if _, err = base.Seek(int64(operation.Start)*blockSize, io.SeekStart); err != nil {
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

func PatchBytes(base []byte, delta []Operation, digest hash.Hash) ([]byte, error) {
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
	if err := Patch(output, baseReader, receive, digest); err != nil {
		return nil, err
	}

	// Success.
	return output.Bytes(), nil
}
