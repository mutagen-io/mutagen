package rsync

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/message"
)

const (
	// maxOutstandingStagingRequests restricts the number of outstanding staging
	// requests to the server. It shouldn't be huge, but it should allow enough
	// requests to be pipelined to avoid latency overhead.
	// TODO: Should we make this dynamic?
	maxOutstandingStagingRequests = 4
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

type stagingOperation struct {
	path string
	base readSeekCloser
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
	stream          message.MessageStream
	root            string
	sinker          Sinker
	updater         UpdateReceiver
	dispatchEngine  *Engine
	receiveEngine   *Engine
	receiveResponse response
	previousError   error
}

func NewClient(connection io.ReadWriter, root string, sinker Sinker, updater UpdateReceiver) *Client {
	return &Client{
		stream:         message.NewMessageStream(connection),
		root:           root,
		sinker:         sinker,
		updater:        updater,
		dispatchEngine: NewDefaultEngine(),
		receiveEngine:  NewDefaultEngine(),
	}
}

func (c *Client) dispatch(
	context context.Context,
	queued <-chan stagingOperation,
	dispatched chan<- stagingOperation,
) error {
	// Loop over queued operations.
	for operation := range queued {
		// Attempt to open the base. If this fails (which it might if the file
		// doesn't exist), then simply use an empty base.
		if file, err := os.Open(filepath.Join(c.root, operation.path)); err != nil {
			operation.base = newEmptyReadSeekCloser()
		} else {
			operation.base = file
		}

		// Compute the base signature. If there is an error, just abort, because
		// most likely the file is being modified concurrently and we'll have to
		// stage again later. We don't treat this as terminal though.
		signature, err := c.dispatchEngine.Signature(operation.base)
		if err != nil {
			operation.base.Close()
			continue
		}

		// Send the transmit request.
		request := request{
			Path:      operation.path,
			Signature: signature,
		}
		if err := c.stream.Encode(request); err != nil {
			operation.base.Close()
			return errors.Wrap(err, "unable to send request")
		}

		// Add the operation to the dispatched queue while watching for
		// cancellation.
		select {
		case dispatched <- operation:
		case <-context.Done():
			operation.base.Close()
			return errors.New("dispatch cancelled")
		}
	}

	// Close the dispatched channel to indicate completion to the receiver.
	close(dispatched)

	// Success.
	return nil
}

func (c *Client) receive(
	context context.Context,
	dispatched <-chan stagingOperation,
	total uint64,
) error {
	// Track the progress index.
	index := uint64(0)

	// Loop until we're out of operations or cancelled.
	for {
		// Grab the next operation.
		var path string
		var base readSeekCloser
		select {
		case operation, ok := <-dispatched:
			if ok {
				path = operation.path
				base = operation.base
			} else {
				return nil
			}
		case <-context.Done():
			return errors.New("receive cancelled")
		}

		// Increment our progress and send an update.
		index += 1
		if err := c.updater(StagingStatus{path, index, total}); err != nil {
			base.Close()
			return errors.Wrap(err, "unable to send staging update")
		}

		// Create a staging sink.
		sink, err := c.sinker.Sink(path)
		if err != nil {
			base.Close()
			return errors.Wrap(err, "unable to create staging sink")
		}

		// Create an operation receiver that tracks receive errors. We re-use
		// our response message, and most importantly it's operation data
		// buffer, to avoid allocation.
		var receiveError error
		receive := func() (Operation, error) {
			// Reset the response message, but leave the data capacity.
			c.receiveResponse.Done = false
			c.receiveResponse.Operation.Data = c.receiveResponse.Operation.Data[:0]
			c.receiveResponse.Operation.Start = 0
			c.receiveResponse.Operation.Count = 0
			c.receiveResponse.Error = ""

			// Receive the next response.
			receiveError = c.stream.Decode(&c.receiveResponse)
			if receiveError != nil {
				return Operation{}, receiveError
			}

			// If the stream is done, then return io.EOF.
			// TODO: There may be some error information in the response. We
			// should record this information and return it in statistics.
			if c.receiveResponse.Done {
				return Operation{}, io.EOF
			}

			// Otherwise just return the operation.
			return c.receiveResponse.Operation, nil
		}

		// Receive and apply patch operations.
		// TODO: We ignore patch errors that aren't due to receive errors
		// because they could just be transient disk errors. We should record
		// this information and return it in statistics.
		c.receiveEngine.Patch(sink, base, receive, nil)

		// Close files.
		sink.Close()
		base.Close()

		// Handle any receive errors. These are terminal.
		if receiveError != nil {
			return errors.Wrap(receiveError, "unable to transmit delta")
		}
	}
}

func (c *Client) Stage(paths []string) error {
	// If there was a previous error, the client is disabled.
	if c.previousError != nil {
		return errors.Wrap(c.previousError, "previous error")
	}

	// Create a queue of staging operations.
	queued := make(chan stagingOperation, len(paths))
	for _, p := range paths {
		queued <- stagingOperation{p, nil}
	}
	close(queued)

	// Create a cancellable context in which our dispatch/receive operations
	// will execute.
	context, cancel := context.WithCancel(context.Background())

	// Create a queue of dispatched operations awaiting response.
	dispatched := make(chan stagingOperation, maxOutstandingStagingRequests)

	// Start our dispatching/receiving pipeline.
	dispatchErrors := make(chan error, 1)
	receiveErrors := make(chan error, 1)
	go func() {
		dispatchErrors <- c.dispatch(context, queued, dispatched)
	}()
	go func() {
		receiveErrors <- c.receive(context, dispatched, uint64(len(paths)))
	}()

	// Wait for completion from all of the Goroutines. If an error is received,
	// cancel the pipeline and wait for completion. Only record the first error,
	// because we don't want cancellation errors being returned.
	var dispatchDone, receiveDone bool
	var pipelineError error
	for !dispatchDone || !receiveDone {
		select {
		case err := <-dispatchErrors:
			dispatchDone = true
			if err != nil {
				if pipelineError == nil {
					pipelineError = errors.Wrap(err, "dispatch error")
				}
				cancel()
			}
		case err := <-receiveErrors:
			receiveDone = true
			if err != nil {
				if pipelineError == nil {
					pipelineError = errors.Wrap(err, "receive error")
				}
				cancel()
			}
		}
	}

	// Handle any pipeline error. If there was a pipeline error, then there may
	// be outstanding operations in the dispatched queue with open files, so we
	// need to close those out.
	if pipelineError != nil {
		for o := range dispatched {
			o.base.Close()
		}
		c.previousError = pipelineError
		return pipelineError
	}

	// Send a final staging update to clear the state.
	if err := c.updater(StagingStatus{}); err != nil {
		return errors.Wrap(err, "unable to send final staging update")
	}

	// Success.
	return nil
}
