package mux

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
// closed but not necessarily if the read end is closed (and even the close
// system call can block on the read end if in a read)).
func (r *readMultiplexer) poll(reader *bufio.Reader) {
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

		// Forward this channel's data.
		_, err = copyN(r.pipes[header.channel], reader, int64(header.length), buffer)
		if err != nil {
			r.closeAllPipesWithError(err)
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
// can be accomplished using the Writer method.
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
