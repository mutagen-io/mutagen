package rsync

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"hash"
	"io"
	"math"

	"github.com/pkg/errors"
)

// EnsureValid verifies that block hash invariants are respected.
func (h *BlockHash) EnsureValid() error {
	// A nil block hash is not valid.
	if h == nil {
		return errors.New("nil block hash")
	}

	// Ensure that the strong signature is valid.
	if len(h.Strong) == 0 {
		return errors.New("empty strong signature")
	}

	// Success.
	return nil
}

// EnsureValid verifies that signature invariants are respected.
func (s *Signature) EnsureValid() error {
	// A nil signature is not valid.
	if s == nil {
		return errors.New("nil signature")
	}

	// Ensure that all block hashes are valid.
	for _, h := range s.Hashes {
		if err := h.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid block hash")
		}
	}

	// If the block size is 0, then the last block size should also be 0 and
	// there shouldn't be any hashes.
	if s.BlockSize == 0 {
		if s.LastBlockSize != 0 {
			return errors.New("block size of 0 with non-0 last block size")
		} else if len(s.Hashes) != 0 {
			return errors.New("block size of 0 with non-0 number of hashes")
		}
		return nil
	}

	// If block size is non-0, then the last block size should be non-0 but less
	// than or equal to the block size.
	if s.LastBlockSize == 0 {
		return errors.New("non-0 block size with last block size of 0")
	} else if s.LastBlockSize > s.BlockSize {
		return errors.New("last block size greater than block size")
	}

	// If the block size is non-0, then a non-zero number of blocks should have
	// been hashed.
	if len(s.Hashes) == 0 {
		return errors.New("non-0 block size with no block hashes")
	}

	// Success.
	return nil
}

// isEmpty return true if the signature represents an empty file.
func (s *Signature) isEmpty() bool {
	// In theory, we might also want to test that LastBlockSize == 0 and that
	// there aren't any hashes, but so long as the invariants of signature are
	// maintained, this check if sufficient.
	return s.BlockSize == 0
}

// EnsureValid verifies that operation invariants are respected.
func (o *Operation) EnsureValid() error {
	// A nil operation is not valid.
	if o == nil {
		return errors.New("nil operation")
	}

	// Ensure that the operation parameters are valid.
	if len(o.Data) > 0 {
		if o.Start != 0 {
			return errors.New("data operation with non-0 block start index")
		} else if o.Count != 0 {
			return errors.New("data operation with non-0 block count")
		}
	} else if o.Count == 0 {
		return errors.New("block operation with 0 block count")
	}

	// Success.
	return nil
}

// Copy creates a deep copy of an operation.
func (o *Operation) Copy() *Operation {
	// Make a copy of the operation's data buffer if necessary.
	var data []byte
	if len(o.Data) > 0 {
		data = make([]byte, len(o.Data))
		copy(data, o.Data)
	}

	// Create the copy.
	return &Operation{
		Data:  data,
		Start: o.Start,
		Count: o.Count,
	}
}

// resetToZero resets an Operation to its zero-value, but leaves capacity in the
// data slice. It's worth noting that the zero-value state is not a valid state
// for an Operation.
func (o *Operation) resetToZeroMaintainingCapacity() {
	// Reset the data slice, but maintain its capacity.
	o.Data = o.Data[:0]

	// Reset start and count.
	o.Start = 0
	o.Count = 0
}

// isZeroValue indicates whether or not an Operation has its zero-value. It's
// worth noting that the zero-value state is not a valid state for an Operation.
func (o *Operation) isZeroValue() bool {
	return len(o.Data) == 0 && o.Start == 0 && o.Count == 0
}

const (
	// minimumOptimalBlockSize is the minimum block size that will be returned
	// by OptimalBlockSizeForBaseLength. It has to be chosen so that it is at
	// least a few orders of magnitude larger than the size of a BlockHash.
	minimumOptimalBlockSize = 1 << 10
	// maximumOptimalBlockSize is the maximum block size that will be returned
	// by OptimalBlockSizeForBaseLength. It mostly just needs to be bounded by
	// what can fit into a reasonably sized in-memory buffer, particularly if
	// multiple rsync engines are running. maximumBlockSize also needs to be
	// less than or equal to (2^32)-1 for the weak hash algorithm to work.
	maximumOptimalBlockSize = 1 << 16
	// DefaultBlockSize is the default block size that will be used if a zero
	// value is passed into Engine.Signature for the blockSize parameter.
	DefaultBlockSize = 1 << 13
	// DefaultMaximumDataOperationSize is the default maximum data size
	// permitted per operation. The optimal value for this isn't at all
	// correlated with block size - it's just what's reasonable to hold
	// in-memory and pass over the wire in a single transmission. This value
	// will be used if a zero value is passed into Engine.Deltafy or
	// Engine.DeltafyBytes for the maxDataOpSize parameter.
	DefaultMaximumDataOperationSize = 1 << 14
)

// OptimalBlockSizeForBaseLength uses a simpler heuristic to choose a block
// size based on the base length. It starts by choosing the optimal block length
// using the formula given in the rsync thesis. It then enforces that the block
// size is within a sensible range.
// TODO: Should we add rounding to "nice" values, e.g. the nearest multiple of
// 1024 bytes? Would this improve read throughput?
func OptimalBlockSizeForBaseLength(baseLength uint64) uint64 {
	// Compute the optimal block length (see the rsync thesis) assuming one
	// change per file.
	result := uint64(math.Sqrt(24.0 * float64(baseLength)))

	// Ensure it's within the allowed range.
	if result < minimumOptimalBlockSize {
		result = minimumOptimalBlockSize
	} else if result > maximumOptimalBlockSize {
		result = maximumOptimalBlockSize
	}

	// Done.
	return result
}

// OptimalBlockSizeForBase is a convenience function that will determine the
// optimal block size for a base that implements io.Seeker. It calls down to
// OptimalBlockSizeForBaseLength. After determining the base's length, it will
// attempt to reset the base to its original position.
func OptimalBlockSizeForBase(base io.Seeker) (uint64, error) {
	if currentOffset, err := base.Seek(0, io.SeekCurrent); err != nil {
		return 0, errors.Wrap(err, "unable to determine current base offset")
	} else if currentOffset < 0 {
		return 0, errors.Wrap(err, "seek return negative starting location")
	} else if length, err := base.Seek(0, io.SeekEnd); err != nil {
		return 0, errors.Wrap(err, "unable to compute base length")
	} else if length < 0 {
		return 0, errors.New("seek returned negative offset")
	} else if _, err = base.Seek(currentOffset, io.SeekStart); err != nil {
		return 0, errors.Wrap(err, "unable to reset base")
	} else {
		return OptimalBlockSizeForBaseLength(uint64(length)), nil
	}
}

// OperationTransmitter transmits an operation. Operation objects and their data
// buffers are re-used between calls to the transmitter, so the transmitter
// should not return until it has either transmitted the operation or copied it
// for later transmission.
type OperationTransmitter func(*Operation) error

// Engine provides rsync functionality without any notion of transport. It is
// designed to be re-used to avoid heavy buffer allocation.
type Engine struct {
	// buffer is a re-usable buffer that will be used for reading data and
	// setting up operations.
	buffer []byte
	// strongHasher is the strong hash function to use for the engine.
	strongHasher hash.Hash
	// strongHashBuffer is a re-usable buffer that can be used by methods to
	// receive digests.
	strongHashBuffer []byte
	// targetReader is a re-usable bufio.Reader that will be used for delta
	// creation operations.
	targetReader *bufio.Reader
	// operation is a re-usable operation object used for transmissions to avoid
	// allocations.
	operation *Operation
}

// NewEngine creates a new rsync engine.
func NewEngine() *Engine {
	// Create the strong hash function.
	// TODO: We might want to allow users to specify other strong hash functions
	// for the engine to use (e.g. BLAKE2 functions), but for now we just use
	// SHA-1 since it's a good balance of speed and robustness for rsync
	// purposes.
	strongHasher := sha1.New()

	// Create the engine.
	return &Engine{
		strongHasher:     strongHasher,
		strongHashBuffer: make([]byte, strongHasher.Size()),
		targetReader:     bufio.NewReader(nil),
		operation:        &Operation{},
	}
}

// bufferWithSize lazily allocates the engine's internal buffer, ensuring that
// it is the required size. The capacity of the internal buffer is retained
// between calls to avoid allocations if possible.
func (e *Engine) bufferWithSize(size uint64) []byte {
	// Check if the buffer currently has the required capacity. If it does, then
	// use that space. Note that we're checking *capacity* - you're allowed to
	// slice a buffer up to its capacity, not just its length. Of course, if you
	// don't own the buffer, you could run into problems with accessing data
	// outside the slice that was given to you, but this buffer is completely
	// internal, so that's not a concern.
	if uint64(cap(e.buffer)) >= size {
		return e.buffer[:size]
	}

	// If we couldn't use our existing buffer, create a new one, but store it
	// for later re-use.
	e.buffer = make([]byte, size)
	return e.buffer
}

const (
	// m is the weak hash modulus. I think they now recommend that it be the
	// largest prime less than 2^16, but this value is fine as well.
	m = 1 << 16
)

// weakHash computes a fast checksum that can be rolled (updated without full
// recomputation). This particular hash is detailed on page 55 of the rsync
// thesis. It is not theoretically optimal, but it's fine for our purposes.
func (e *Engine) weakHash(data []byte, blockSize uint64) (uint32, uint32, uint32) {
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
func (e *Engine) rollWeakHash(r1, r2 uint32, out, in byte, blockSize uint64) (uint32, uint32, uint32) {
	// Update components.
	r1 = (r1 - uint32(out) + uint32(in)) % m
	r2 = (r2 - uint32(blockSize)*uint32(out) + r1) % m

	// Compute the hash.
	result := r1 + m*r2

	// Done.
	return result, r1, r2
}

// strongHash computes a slow but strong hash for a block of data. If allocate
// is true, then a new byte slice will be allocated to receive the digest,
// otherwise the engine's internal digest buffer will be used, but then the
// digest will only be valid until the next call to strongHash.
func (e *Engine) strongHash(data []byte, allocate bool) []byte {
	// Reset the hasher.
	e.strongHasher.Reset()

	// Digest the data. The Hash interface guarantees that writes succeed.
	e.strongHasher.Write(data)

	// Compute the output location.
	var output []byte
	if !allocate {
		output = e.strongHashBuffer[:0]
	}

	// Compute the digest.
	return e.strongHasher.Sum(output)
}

// Signature computes the signature for a base stream. If the provided block
// size is 0, this method will attempt to compute the optimal block size (which
// requires that base implement io.Seeker), and failing that will fall back to a
// default block size.
func (e *Engine) Signature(base io.Reader, blockSize uint64) (*Signature, error) {
	// Choose a block size if none is specified. If the base also implements
	// io.Seeker (which most will since they need to for Patch), then use the
	// optimal block size, otherwise use the default.
	if blockSize == 0 {
		if baseSeeker, ok := base.(io.Seeker); ok {
			if s, err := OptimalBlockSizeForBase(baseSeeker); err == nil {
				blockSize = s
			} else {
				blockSize = DefaultBlockSize
			}
		} else {
			blockSize = DefaultBlockSize
		}
	}

	// Create the result.
	result := &Signature{
		BlockSize: blockSize,
	}

	// Create a buffer with which to read blocks.
	buffer := e.bufferWithSize(blockSize)

	// Read blocks and append their hashes until we reach EOF.
	eof := false
	for !eof {
		// Read the next block and watch for errors. If we receive io.EOF, then
		// nothing was read, and we should break immediately. This means that
		// the base had a length that was a multiple of the block size. If we
		// receive io.ErrUnexpectedEOF, then something was read but we're still
		// at the end of the file, so we should hash this block but not go
		// through the loop again. All other errors are terminal.
		n, err := io.ReadFull(base, buffer)
		if err == io.EOF {
			result.LastBlockSize = blockSize
			break
		} else if err == io.ErrUnexpectedEOF {
			result.LastBlockSize = uint64(n)
			eof = true
		} else if err != nil {
			return nil, errors.Wrap(err, "unable to read data block")
		}

		// Compute hashes for the the block that was read. For short blocks, we
		// still use the full block size when computing the weak hash. We could
		// alternatively use the short block length, but it doesn't matter - all
		// that matters is that we keep consistency when we compute the short
		// block weak hash when searching in Deltafy.
		weak, _, _ := e.weakHash(buffer[:n], blockSize)
		strong := e.strongHash(buffer[:n], true)

		// Add the block hash.
		result.Hashes = append(result.Hashes, &BlockHash{
			Weak:   weak,
			Strong: strong,
		})
	}

	// If there are no hashes, then clear out the block sizes.
	if len(result.Hashes) == 0 {
		result.BlockSize = 0
		result.LastBlockSize = 0
	}

	// Success.
	return result, nil
}

// BytesSignature computes the signature for a byte slice.
func (e *Engine) BytesSignature(base []byte, blockSize uint64) *Signature {
	// Perform the signature and watch for errors (which shouldn't be able to
	// occur in-memory).
	result, err := e.Signature(bytes.NewReader(base), blockSize)
	if err != nil {
		panic(errors.Wrap(err, "in-memory signature failure"))
	}

	// Success.
	return result
}

// dualModeReader unifies the io.Reader and io.ByteReader interfaces. It is used
// in deltafy operations to ensure that bytes can be efficiently extracted from
// targets.
type dualModeReader interface {
	io.Reader
	io.ByteReader
}

// min implements simple minimum finding for uint64 values.
func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// transmitData transmits a data operation using the engine's internal operation
// object.
func (e *Engine) transmitData(data []byte, transmit OperationTransmitter) error {
	// Set the operation parameters.
	*e.operation = Operation{
		Data: data,
	}

	// Transmit.
	return transmit(e.operation)
}

// transmitBlock transmits a block operation using the engine's internal
// operation object.
func (e *Engine) transmitBlock(start, count uint64, transmit OperationTransmitter) error {
	// Set the operation parameters.
	*e.operation = Operation{
		Start: start,
		Count: count,
	}

	// Transmit.
	return transmit(e.operation)
}

// chunkAndTransmitAll is a fast-path routine for simply transmitting all data
// in a target stream. This is used when there are no blocks to match because
// the base stream is empty.
func (e *Engine) chunkAndTransmitAll(target io.Reader, maxDataOpSize uint64, transmit OperationTransmitter) error {
	// Verify that maxDataOpSize is sane.
	if maxDataOpSize == 0 {
		maxDataOpSize = DefaultMaximumDataOperationSize
	}

	// Create a buffer to transmit data operations.
	buffer := e.bufferWithSize(maxDataOpSize)

	// Loop until the entire target has been transmitted as data operations.
	for {
		if n, err := io.ReadFull(target, buffer); err == io.EOF {
			return nil
		} else if err == io.ErrUnexpectedEOF {
			if err = e.transmitData(buffer[:n], transmit); err != nil {
				return errors.Wrap(err, "unable to transmit data operation")
			}
			return nil
		} else if err != nil {
			return errors.Wrap(err, "unable to read target")
		} else if err = e.transmitData(buffer, transmit); err != nil {
			return errors.Wrap(err, "unable to transmit data operation")
		}
	}
}

// Deltafy computes delta operations to reconstitute the target data stream
// using the base stream (based on the provided base signature). It streams
// operations to the provided transmission function. The internal engine buffer
// will be resized to the sum of the maximum data operation size plus the block
// size, and retained for the lifetime of the engine, so a reasonable value
// for the maximum data operation size should be provided. For performance
// reasons, this method does not validate that the provided signature satisfies
// expected invariants. It is the responsibility of the caller to verify that
// the signature is valid by calling its EnsureValid method. This is not
// necessary for signatures generated in the same process, but should be done
// for signatures received from untrusted locations (e.g. over the network). An
// invalid signature can result in undefined behavior.
func (e *Engine) Deltafy(target io.Reader, base *Signature, maxDataOpSize uint64, transmit OperationTransmitter) error {
	// Verify that the maximum data operation size is sane.
	if maxDataOpSize == 0 {
		maxDataOpSize = DefaultMaximumDataOperationSize
	}

	// If the base is empty, then there's no way we'll find any matching blocks,
	// so just send the entire file.
	if len(base.Hashes) == 0 {
		return e.chunkAndTransmitAll(target, maxDataOpSize, transmit)
	}

	// Create a set of block and data transmitters that efficiently coalesce
	// adjacent block operations and provide data chunking. Some corresponding
	// finalization logic is required at the end of this function.
	var coalescedStart, coalescedCount uint64
	sendBlock := func(index uint64) error {
		if coalescedCount > 0 {
			if coalescedStart+coalescedCount == index {
				coalescedCount += 1
				return nil
			} else if err := e.transmitBlock(coalescedStart, coalescedCount, transmit); err != nil {
				return nil
			}
		}
		coalescedStart = index
		coalescedCount = 1
		return nil
	}
	sendData := func(data []byte) error {
		if len(data) > 0 && coalescedCount > 0 {
			if err := e.transmitBlock(coalescedStart, coalescedCount, transmit); err != nil {
				return err
			}
			coalescedStart = 0
			coalescedCount = 0
		}
		for len(data) > 0 {
			sendSize := min(uint64(len(data)), maxDataOpSize)
			if err := e.transmitData(data[:sendSize], transmit); err != nil {
				return err
			}
			data = data[sendSize:]
		}
		return nil
	}

	// Ensure that the target implements io.Reader and io.ByteReader. If it can
	// do this natively, great! If not, wrap it in our re-usable buffered
	// reader, but ensure that it is released when we're done so that we don't
	// retain it indefinitely.
	bufferedTarget, ok := target.(dualModeReader)
	if !ok {
		e.targetReader.Reset(target)
		bufferedTarget = e.targetReader
		defer func() {
			e.targetReader.Reset(nil)
		}()
	}

	// Create a lookup table that maps weak hashes to all matching block hashes.
	// If the last block is short, we extract it and hold it separately, because
	// when doing match searches, we assume that all blocks in this map have a
	// full block size worth of data.
	//
	// The rsync technical report (see the section titled "Checksum searching")
	// actually advocates a 3-tier search (i.e. an additional 16-bit hash layer
	// before the weak hash), but I think this probably isn't necessary with
	// modern hardware and hashing algorithms.
	//
	// TODO: This is currently a little expensive because it requires a slice
	// allocation for each map entry. I suspect that the collision rate for weak
	// hashes is actually sufficiently low that we could make each map value a
	// fixed array of int that would limit the number of matches we could try,
	// but save us a lot of allocating. We would have to use an int, because
	// invalid values would likely need to be -1. This might be an unnecessary
	// operation though, because this map is only generated for non-empty bases,
	// which typically don't come in large numbers. For a few files, generating
	// these maps with slice values is fine. It also might be a bit slow since
	// each insertion would require a linear search to find the insertion
	// location within the array.
	hashes := base.Hashes
	haveShortLastBlock := false
	var lastBlockIndex uint64
	var shortLastBlock *BlockHash
	if base.LastBlockSize != base.BlockSize {
		haveShortLastBlock = true
		lastBlockIndex = uint64(len(hashes) - 1)
		shortLastBlock = hashes[lastBlockIndex]
		hashes = hashes[:lastBlockIndex]
	}
	weakToBlockHashes := make(map[uint32][]uint64, len(hashes))
	for i, h := range hashes {
		weakToBlockHashes[h.Weak] = append(weakToBlockHashes[h.Weak], uint64(i))
	}

	// Create a buffer that we can use to load data and search for matches. We
	// start by filling it with a block's worth of data and then continuously
	// appending bytes until we either fill the buffer (at which point we
	// transmit data preceding the block and truncate) or find a match (at
	// which point we transmit data preceding the block and then transmit the
	// block match). Once we're unable to append a new byte or refill with a
	// full block, we terminate our search and send the remaining data
	// (potentially searching for one last short block match at the end of the
	// buffer).
	//
	// We choose the buffer size to hold a chunk of data of the maximum allowed
	// transmission size and a block of data. This size choice is somewhat
	// arbitrary since we have a data chunking function and could load more data
	// before doing a truncation/transmission, but this is also a reasonable
	// amount of data to hold in memory at any given time. We could choose a
	// larger preceding data chunk size to have less frequent truncations, but
	// (a) truncations are cheap and (b) we'll probably be doing a lot of
	// sequential block matching cycles where we just continuously match blocks
	// at the beginning of the buffer and then refill, so truncations won't be
	// all that common.
	buffer := e.bufferWithSize(maxDataOpSize + base.BlockSize)

	// Track the occupancy of the buffer.
	var occupancy uint64

	// Track the weak hash and its parameters for the block at the end of the
	// buffer.
	var weak, r1, r2 uint32

	// Loop over the contents of the file and search for matches.
	for {
		// If the buffer is empty, then we need to read in a block's worth of
		// data (if possible) and calculate the weak hash and its parameters. If
		// the buffer is non-empty but less than a block's worth of data, then
		// we've broken an invariant in our code. Otherwise, we need to move the
		// search block one byte forward and roll the hash.
		if occupancy == 0 {
			if n, err := io.ReadFull(bufferedTarget, buffer[:base.BlockSize]); err == io.EOF || err == io.ErrUnexpectedEOF {
				occupancy = uint64(n)
				break
			} else if err != nil {
				return errors.Wrap(err, "unable to perform initial buffer fill")
			} else {
				occupancy = base.BlockSize
				weak, r1, r2 = e.weakHash(buffer[:occupancy], base.BlockSize)
			}
		} else if occupancy < base.BlockSize {
			panic("buffer contains less than a block worth of data")
		} else {
			if b, err := bufferedTarget.ReadByte(); err == io.EOF {
				break
			} else if err != nil {
				return errors.Wrap(err, "unable to read target byte")
			} else {
				weak, r1, r2 = e.rollWeakHash(r1, r2, buffer[occupancy-base.BlockSize], b, base.BlockSize)
				buffer[occupancy] = b
				occupancy += 1
			}
		}

		// Look for a block match for the block at the end of the buffer.
		potentials := weakToBlockHashes[weak]
		match := false
		var matchIndex uint64
		if len(potentials) > 0 {
			strong := e.strongHash(buffer[occupancy-base.BlockSize:occupancy], false)
			for _, p := range potentials {
				if bytes.Equal(base.Hashes[p].Strong, strong) {
					match = true
					matchIndex = p
					break
				}
			}
		}

		// If there's a match, send any data preceding the match and then send
		// the match. Otherwise, if we've reached buffer capacity, send the data
		// preceding the search block.
		if match {
			if err := sendData(buffer[:occupancy-base.BlockSize]); err != nil {
				return errors.Wrap(err, "unable to transmit data preceding match")
			} else if err = sendBlock(matchIndex); err != nil {
				return errors.Wrap(err, "unable to transmit match")
			}
			occupancy = 0
		} else if occupancy == uint64(len(buffer)) {
			if err := sendData(buffer[:occupancy-base.BlockSize]); err != nil {
				return errors.Wrap(err, "unable to transmit data before truncation")
			}
			copy(buffer[:base.BlockSize], buffer[occupancy-base.BlockSize:occupancy])
			occupancy = base.BlockSize
		}
	}

	// If we have a short last block and the occupancy of the buffer is large
	// enough that it could match, then check for a match.
	if haveShortLastBlock && occupancy >= base.LastBlockSize {
		potentialLastBlockMatch := buffer[occupancy-base.LastBlockSize : occupancy]
		// For short blocks, we still use the full block size when computing the
		// weak hash. We could alternatively use the short block length, but it
		// doesn't matter - all that matters is that we keep consistency when we
		// compute the short block weak hash in Signature.
		if w, _, _ := e.weakHash(potentialLastBlockMatch, base.BlockSize); w == shortLastBlock.Weak {
			if bytes.Equal(e.strongHash(potentialLastBlockMatch, false), shortLastBlock.Strong) {
				if err := sendData(buffer[:occupancy-base.LastBlockSize]); err != nil {
					return errors.Wrap(err, "unable to transmit data")
				} else if err = sendBlock(lastBlockIndex); err != nil {
					return errors.Wrap(err, "unable to transmit operation")
				}
				occupancy = 0
			}
		}
	}

	// Send any data remaining in the buffer.
	if err := sendData(buffer[:occupancy]); err != nil {
		return errors.Wrap(err, "unable to send final data operation")
	}

	// Send any final pending coalesced operation. This can't be done as a defer
	// because we need to watch for errors.
	if coalescedCount > 0 {
		if err := e.transmitBlock(coalescedStart, coalescedCount, transmit); err != nil {
			return errors.Wrap(err, "unable to send final block operation")
		}
	}

	// Success.
	return nil
}

// DeltafyBytes computes delta operations for a byte slice. Unlike the streaming
// Deltafy method, it returns a slice of operations, which should be reasonable
// since the target data can already fit into memory. The internal engine buffer
// will be resized to the sum of the maximum data operation size plus the block
// size, and retained for the lifetime of the engine, so a reasonable value
// for the maximum data operation size should be provided. For performance
// reasons, this method does not validate that the provided signature satisfies
// expected invariants. It is the responsibility of the caller to verify that
// the signature is valid by calling its EnsureValid method. This is not
// necessary for signatures generated in the same process, but should be done
// for signatures received from untrusted locations (e.g. over the network). An
// invalid signature can result in undefined behavior.
func (e *Engine) DeltafyBytes(target []byte, base *Signature, maxDataOpSize uint64) []*Operation {
	// Create an empty result.
	var delta []*Operation

	// Create an operation transmitter to populate the result.
	transmit := func(o *Operation) error {
		delta = append(delta, o.Copy())
		return nil
	}

	// Wrap up the bytes in a reader.
	reader := bytes.NewReader(target)

	// Compute the delta and watch for errors (which shouldn't occur for for
	// in-memory data).
	if err := e.Deltafy(reader, base, maxDataOpSize, transmit); err != nil {
		panic(errors.Wrap(err, "in-memory deltafication failure"))
	}

	// Success.
	return delta
}

// Patch applies a single operation against a base stream to reconstitute the
// target into the destination stream. For performance reasons, this method does
// not validate that the provided signature and operation satisfy expected
// invariants. It is the responsibility of the caller to verify that the
// signature and operation are valid by calling their respective EnsureValid
// methods. This is not necessary for signatures and operations generated in the
// same process, but should be done for signatures and operations received from
// untrusted locations (e.g. over the network). An invalid signature or
// operation can result in undefined behavior.
func (e *Engine) Patch(destination io.Writer, base io.ReadSeeker, signature *Signature, operation *Operation) error {
	// Handle the operation based on type.
	if len(operation.Data) > 0 {
		// Write data operations directly to the destination.
		if _, err := destination.Write(operation.Data); err != nil {
			return errors.Wrap(err, "unable to write data")
		}
	} else {
		// Seek to the start of the requested block in base.
		// TODO: We should technically validate that operation.Index
		// multiplied by the block size can't overflow an int64. Worst case
		// at the moment it will cause the seek operation to fail.
		if _, err := base.Seek(int64(operation.Start)*int64(signature.BlockSize), io.SeekStart); err != nil {
			return errors.Wrap(err, "unable to seek to base location")
		}

		// Copy the requested number of blocks.
		for c := uint64(0); c < operation.Count; c++ {
			// Compute the size to copy.
			copyLength := signature.BlockSize
			if operation.Start+c == uint64(len(signature.Hashes)-1) {
				copyLength = signature.LastBlockSize
			}

			// Create a buffer of the required size.
			buffer := e.bufferWithSize(copyLength)

			// Copy the block.
			if _, err := io.ReadFull(base, buffer); err != nil {
				return errors.Wrap(err, "unable to read block data")
			} else if _, err = destination.Write(buffer); err != nil {
				return errors.Wrap(err, "unable to write block data")
			}
		}
	}

	// Success.
	return nil
}

// PatchBytes applies a series of operations against a base byte slice to
// reconstitute the target byte slice. For performance reasons, this method does
// not validate that the provided signature and operation satisfy expected
// invariants. It is the responsibility of the caller to verify that the
// signature and operation are valid by calling their respective EnsureValid
// methods. This is not necessary for signatures and operations generated in the
// same process, but should be done for signatures and operations received from
// untrusted locations (e.g. over the network). An invalid signature or
// operation can result in undefined behavior.
func (e *Engine) PatchBytes(base []byte, signature *Signature, delta []*Operation) ([]byte, error) {
	// Wrap up the base bytes in a reader.
	baseReader := bytes.NewReader(base)

	// Create an output buffer.
	output := bytes.NewBuffer(nil)

	// Perform application.
	for _, o := range delta {
		if err := e.Patch(output, baseReader, signature, o); err != nil {
			return nil, err
		}
	}

	// Success.
	return output.Bytes(), nil
}
