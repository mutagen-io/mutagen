package multiplex

import (
	"bufio"
	"io"

	"github.com/pkg/errors"
)

const (
	// TODO: Figure out if we should set this on a per-machine basis. This value
	// is taken from Go's io.Copy method, which defaults to allocating a 32k
	// buffer if none is provided.
	readMultiplexerCopyBufferSize = 32 * 1024
)

// readMultiplexer implements the multiplexing protocol over an io.Reader. It
// uses a polling Goroutine to read headers and then forwards their data blocks
// to io.Pipe objects that are given out as channels.
type readMultiplexer struct {
	pipes []*io.PipeWriter
}

// closeAllPipesWithError closes all pipes in the multiplexer with the specified
// error, letting all channels know that they will not receive any more data. It
// does not close the underlying reader, which must be done to ensure polling
// shutdown.
func (r *readMultiplexer) closeAllPipesWithError(err error) {
	for _, p := range r.pipes {
		p.CloseWithError(err)
	}
}

// poll polls on the underlying reader and forwards data blocks to their
// respective channels.
//
// This function will work correctly on a raw io.Reader, but the interface
// mandates the reader be buffered to avoid unnecessary overhead on short reads
// during header decoding.
//
// Because poll alternates between reading from the underlying reader and
// writing to the various channel pipes, its termination can only be guaranteed
// when the underlying reader and all pipes have been closed. Additionally, the
// underlying reader must unblock reads when closed, but this is definitely not
// true of all readers, so be careful! It is true for io.Pipe streams, net.Conn
// implementations, and some others, but it is very much not true for OS pipes
// on all systems (for OS pipes, the reader will unblock if the write end is
// closed, but not necessarily if the read end is closed (and trying to close
// the read end during a blocking read can even block the close)).
func (r *readMultiplexer) poll(reader *bufio.Reader) {
	// Create a limiting wrapper around the reader to restrict read lengths.
	limiter := &io.LimitedReader{R: reader}

	// Create a copy buffer we can use for data forwarding.
	buffer := make([]byte, readMultiplexerCopyBufferSize)

	// Loop until there's an error.
	for {
		// Read the next header. If that fails, close all pipes.
		header, err := readHeader(reader)
		if err != nil {
			r.closeAllPipesWithError(err)
			return
		}

		// Verify that the channel is valid and extract its pipe.
		if int(header.channel) >= len(r.pipes) {
			r.closeAllPipesWithError(errors.New("received invalid channel"))
			return
		}

		// Forward this channel's data. io.LimitedReader doesn't treat an early
		// io.EOF as an io.ErrUnexpectedEOF (it's a limiting value, not a
		// minimum value), so we also check that the transmitted length is what
		// we expect (because CopyBuffer is expecting an io.EOF as its
		// termination condition and will just gobble that up).
		limiter.N = int64(header.length)
		if copied, err := io.CopyBuffer(r.pipes[header.channel], limiter, buffer); err != nil {
			r.closeAllPipesWithError(err)
			return
		} else if copied != int64(header.length) {
			r.closeAllPipesWithError(io.ErrUnexpectedEOF)
			return
		}
	}
}

// Close closes all pipes in the multiplexer with io.ErrClosedPipe, letting all
// channels know that they will not receive any more data. It does not close the
// underlying reader, which must be done to ensure polling shutdown.
func (r *readMultiplexer) Close() error {
	r.closeAllPipesWithError(nil)
	return nil
}

// Reader demultiplexes an io.Reader into independent byte streams. Multiplexing
// can be accomplished using the Writer method. Demultiplexing will use an
// additional background Goroutine to poll the underlying reader for data and
// route it appropriately. This Goroutine is only guaranteed to terminate when
// both the returned io.Closer is closed (which terminates data routing) and the
// underlying io.Reader is closed and reads on it unblock. This means that the
// io.Reader provided to this method must unblock when closed, and you should be
// aware that this guarantee is not part of the io.Reader interface and that
// many io.Reader objects don't adhere to this behavior. Be particularly careful
// with OS pipes, because on some platforms reads can only be unblocked by
// closing the write end of the pipe, and closes on the read end while in a
// blocking read can even block the close operation.
func Reader(reader io.Reader, channels uint8) ([]io.Reader, io.Closer) {
	// Create our channel pipes.
	readers := make([]io.Reader, channels)
	pipes := make([]*io.PipeWriter, channels)
	for c := uint8(0); c < channels; c++ {
		r, w := io.Pipe()
		readers[c] = r
		pipes[c] = w
	}

	// Create the multiplexer and start its polling Goroutine.
	multiplexer := &readMultiplexer{pipes}
	go multiplexer.poll(bufio.NewReader(reader))

	// Done.
	return readers, multiplexer
}
