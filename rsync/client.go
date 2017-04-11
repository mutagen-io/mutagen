package rsync

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/message"
)

type readSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type emptyReadSeekCloser struct {
	*bytes.Reader
}

func newEmptyReadSeekCloser() readSeekCloser {
	return &emptyReadSeekCloser{bytes.NewReader(nil)}
}

func (e *emptyReadSeekCloser) Close() error {
	return nil
}

// Sinker provides the interface for Client to store incoming streams.
type Sinker interface {
	// Sink should return a new write closer for staging the given path.
	Sink(string) (io.WriteCloser, error)
}

type StagingStatus struct {
	Path  string
	Index uint64
	Total uint64
	// TODO: Expand this struct with more detailed status information, e.g.
	// failed requests, bandwidth, internal statistics, speedup factor, etc.
}

type UpdateReceiver func(StagingStatus) error

type Client struct {
	stream        message.MessageStream
	root          string
	sinker        Sinker
	updater       UpdateReceiver
	engine        *Engine
	response      response
	receiveError  error
	previousError error
}

func NewClient(connection io.ReadWriter, root string, sinker Sinker, updater UpdateReceiver) *Client {
	return &Client{
		stream:  message.NewMessageStream(connection),
		root:    root,
		sinker:  sinker,
		updater: updater,
		engine:  NewDefaultEngine(),
	}
}

// receive is an OperationReceiver that can be used to read a single operation
// stream. If it encounters an error, it will store the error in the
// receiveError field.
func (c *Client) receive() (Operation, error) {
	// Reset the response message, but leave the data capacity.
	c.response.Done = false
	c.response.Operation.Data = c.response.Operation.Data[:0]
	c.response.Operation.Start = 0
	c.response.Operation.Count = 0
	c.response.Error = ""

	// Receive the next response. We don't have to worry about over-writing
	// the receiveError, because once this function returns an error, other than
	// EndOfOperations, it will never be called again.
	c.receiveError = c.stream.Decode(&c.response)
	if c.receiveError != nil {
		return Operation{}, c.receiveError
	}

	// If the stream is done, then return EndOfOperations. There may be an error
	// in the response as well, but it's only relevant to this stream, and it
	// won't prevent us from closing out the sink.
	// TODO: Should we record response error information and return it in
	// statistics.
	if c.response.Done {
		return Operation{}, EndOfOperations
	}

	// Otherwise just return the operation.
	return c.response.Operation, nil
}

func (c *Client) burnOperationStream() error {
	for {
		if _, err := c.receive(); err == EndOfOperations {
			return nil
		} else if err != nil {
			return errors.Wrap(err, "unable to receive operation")
		}
	}
}

func (c *Client) Stage(paths []string) error {
	// Check if the client is errored.
	if c.previousError != nil {
		return errors.Wrap(c.previousError, "previous error")
	}

	// Convert paths to full paths.
	fullPaths := make([]string, len(paths))
	for i, p := range paths {
		fullPaths[i] = filepath.Join(c.root, p)
	}

	// Compute signatures for the paths. If a path fails to open or we're unable
	// to compute its signature, just give it an empty signature, but record
	// that we shouldn't expect it to have a valid base.
	signatures := make([][]BlockHash, len(paths))
	failedToOpen := make([]bool, len(paths))
	for i, p := range fullPaths {
		if f, err := os.Open(p); err != nil {
			failedToOpen[i] = true
		} else {
			if s, err := c.engine.Signature(f); err != nil {
				failedToOpen[i] = true
			} else {
				signatures[i] = s
			}
			f.Close()
		}
	}

	// Send the request.
	if err := c.stream.Encode(request{paths, signatures}); err != nil {
		c.previousError = errors.Wrap(err, "unable to send request")
		return c.previousError
	}

	// Handle responses.
	for i, p := range fullPaths {
		// Send a progress update.
		status := StagingStatus{paths[i], uint64(i), uint64(len(paths))}
		if err := c.updater(status); err != nil {
			c.previousError = errors.Wrap(err, "unable to send staging update")
			return c.previousError
		}

		// Open the base. If the base previously failed to open, then just
		// create an empty base. If it fails to open now, then we need to burn
		// off the incoming operation stream for this file.
		var base readSeekCloser
		if failedToOpen[i] {
			base = newEmptyReadSeekCloser()
		} else if f, err := os.Open(filepath.Join(c.root, p)); err != nil {
			if err = c.burnOperationStream(); err != nil {
				c.previousError = errors.Wrap(err, "unable to burn operation stream")
				return c.previousError
			}
		} else {
			base = f
		}

		// Create a staging sink.
		sink, err := c.sinker.Sink(paths[i])
		if err != nil {
			base.Close()
			c.previousError = errors.Wrap(err, "unable to create staging sink")
			return c.previousError
		}

		// Receive and apply patch operations.
		// TODO: We ignore patch errors that aren't due to receive errors
		// because they could just be transient disk errors. We should record
		// this information and return it in statistics.
		c.engine.Patch(sink, base, c.receive, nil)

		// Close files.
		sink.Close()
		base.Close()

		// Handle any receive errors. These are terminal.
		if c.receiveError != nil {
			c.previousError = errors.Wrap(err, "unable to receive operation")
			return c.previousError
		}
	}

	// Send a final staging update to clear the state.
	if err := c.updater(StagingStatus{}); err != nil {
		c.previousError = errors.Wrap(err, "unable to send final staging update")
		return c.previousError
	}

	// Success.
	return nil
}
