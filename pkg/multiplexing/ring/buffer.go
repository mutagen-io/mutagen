package ring

import (
	"errors"
	"io"
)

var (
	// ErrBufferFull is the error returned by Buffer if a storage operation
	// can't be completed due to a lack of space in the buffer.
	ErrBufferFull = errors.New("buffer full")
)

// min returns the lesser of a or b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Buffer is a fixed-size ring buffer for storing bytes. Its behavior is
// designed to match that of bytes.Buffer as closely as possible. The zero value
// for Buffer is a buffer with zero capacity.
type Buffer struct {
	// storage is the buffer's underlying storage. There are eight possible data
	// layout states within the storage buffer depending on the buffer size and
	// operational history:
	//
	// - [] (Buffers of length 0 only)
	// - [FREE1] (Buffers of length >= 1, start always reset to 0 in this case)
	// - [DATA1] (Buffers of length >= 1)
	// - [DATA1|FREE1] (Buffers of length >= 2)
	// - [FREE1|DATA1] (Buffers of length >= 2)
	// - [DATA2|DATA1] (Buffers of length >= 2)
	//   - The corresponding [FREE2|FREE1] layout is prohibited by an optimizing
	//     reset operation whenever the buffer is fully drained.
	// - [FREE2|DATA1|FREE1] (Buffers of length >= 3)
	// - [DATA2|FREE1|DATA1] (Buffers of length >= 3)
	//
	// No additional states with further fragmentation of data or free space are
	// possible under the invariants of the buffer's algorithms (nor would they
	// be encodable by this data structure).
	storage []byte
	// size is the storage buffer size. It is cached for better performance.
	size int
	// start is the data start index. It is restricted to the range [0, size).
	start int
	// used is the number of bytes currently stored in the buffer. It is
	// restricted to the range [0, size].
	used int
}

// NewBuffer creates a new ring buffer with the specified size. If size is less
// than or equal to 0, then a buffer with zero capacity is created.
func NewBuffer(size int) *Buffer {
	if size <= 0 {
		return &Buffer{}
	}
	return &Buffer{
		storage: make([]byte, size),
		size:    size,
	}
}

// Size returns the size of the buffer.
func (b *Buffer) Size() int {
	return b.size
}

// Used returns how many bytes currently reside in the buffer.
func (b *Buffer) Used() int {
	return b.used
}

// Free returns the unused buffer capacity.
func (b *Buffer) Free() int {
	return b.size - b.used
}

// Reset clears all data within the buffer.
func (b *Buffer) Reset() {
	b.start = 0
	b.used = 0
}

// Write implements io.Writer.Write.
func (b *Buffer) Write(data []byte) (int, error) {
	// Loop until we've consumed the data buffer or run out of storage.
	var result int
	for len(data) > 0 && b.used != b.size {
		// Compute the first available contiguous free storage segment.
		freeStart := (b.start + b.used) % b.size
		free := b.storage[freeStart:min(freeStart+(b.size-b.used), b.size)]

		// Copy data into storage.
		copied := copy(free, data)

		// Update indices and tracking.
		result += copied
		data = data[copied:]
		b.used += copied
	}

	// If we couldn't fully consume the source buffer due to a lack of storage,
	// then we need to return an error.
	if len(data) > 0 && b.used == b.size {
		return result, ErrBufferFull
	}

	// Success.
	return result, nil
}

// WriteByte implements io.ByteWriter.WriteByte.
func (b *Buffer) WriteByte(value byte) error {
	// If there's no space available, then we can't write the byte.
	if b.used == b.size {
		return ErrBufferFull
	}

	// Compute the start of the first available free storage segment.
	freeStart := (b.start + b.used) % b.size

	// Store the byte.
	b.storage[freeStart] = value

	// Update tracking.
	b.used += 1

	// Success.
	return nil
}

// ReadNFrom is similar to using io.ReaderFrom.ReadFrom with io.LimitedReader,
// but it is designed to support a limited-capacity buffer, which can't reliably
// detect EOF without potentially wasting data from the stream. In particular,
// Buffer can't reliably detect the case that EOF is reached right as its
// storage is filled because io.Reader is not required to return io.EOF until
// the next call, and most implementations (including io.LimitedReader) will
// only return io.EOF on a subsequent call. Moreover, io.Reader isn't required
// to return an EOF indication on a zero-length read, so even a follow-up
// zero-length read can't be used to reliably detect EOF. As such, this method
// provides a more explicit definition of the number of bytes to read, and it
// will return io.EOF if encountered, unless it occurs simultaneously with
// request completion.
func (b *Buffer) ReadNFrom(reader io.Reader, n int) (int, error) {
	// Loop until we've filled completed the read, run out of storage, or
	// encountered a read error.
	var read, result int
	var err error
	for n > 0 && b.used != b.size && err == nil {
		// Compute the first available contiguous free storage segment.
		freeStart := (b.start + b.used) % b.size
		free := b.storage[freeStart:min(freeStart+(b.size-b.used), b.size)]

		// If the storage segment is larger than we need, then truncate it.
		if len(free) > n {
			free = free[:n]
		}

		// Perform the read.
		read, err = reader.Read(free)

		// Update indices and tracking.
		result += read
		b.used += read
		n -= read
	}

	// If we couldn't complete the read due to a lack of storage, then we need
	// to return an error. However, if a read error occurred simultaneously with
	// running out of storage, then we don't overwrite it.
	if n > 0 && b.used == b.size && err == nil {
		err = ErrBufferFull
	}

	// If we encountered io.EOF simultaneously with completing the read, then we
	// can clear the error.
	if err == io.EOF && n == 0 {
		err = nil
	}

	// Done.
	return result, err
}

// Read implements io.Reader.Read.
func (b *Buffer) Read(buffer []byte) (int, error) {
	// If the destination buffer is zero-length, then we return with no error,
	// even if we have no data available. Otherwise, if we don't have any data
	// available, then return EOF.
	if len(buffer) == 0 {
		return 0, nil
	} else if b.used == 0 {
		return 0, io.EOF
	}

	// Loop until we've filled the destination buffer or drained storage.
	var result int
	for len(buffer) > 0 && b.used > 0 {
		// Compute the first available contiguous data segment.
		data := b.storage[b.start:min(b.start+b.used, b.size)]

		// Copy the data.
		copied := copy(buffer, data)

		// Update indices and tracking.
		result += copied
		buffer = buffer[copied:]
		b.start += copied
		b.start %= b.size
		b.used -= copied
	}

	// Reset to an optimal layout if possible.
	if b.used == 0 {
		b.start = 0
	}

	// Success.
	return result, nil
}

// ReadByte implements io.ByteReader.ReadByte.
func (b *Buffer) ReadByte() (byte, error) {
	// If we don't have any data available, then return EOF.
	if b.used == 0 {
		return 0, io.EOF
	}

	// Extract the first byte of data.
	result := b.storage[b.start]

	// Update indices and tracking.
	b.start += 1
	b.start %= b.size
	b.used -= 1

	// Reset to an optimal layout if possible.
	if b.used == 0 {
		b.start = 0
	}

	// Success.
	return result, nil
}

// WriteTo implements io.WriterTo.WriteTo.
func (b *Buffer) WriteTo(writer io.Writer) (int64, error) {
	// Loop until we've drained the storage buffer or encountered a write error.
	var written int
	var result int64
	var err error
	for b.used > 0 && err == nil {
		// Compute the first available contiguous data segment.
		data := b.storage[b.start:min(b.start+b.used, b.size)]

		// Write the data.
		written, err = writer.Write(data)

		// Update indices and tracking.
		result += int64(written)
		b.start += written
		b.start %= b.size
		b.used -= written
	}

	// Reset to an optimal layout if possible.
	if b.used == 0 {
		b.start = 0
	}

	// Done.
	return result, err
}
