package rsync

import (
	"bytes"
	"context"
	"io"

	"github.com/pkg/errors"

	fs "github.com/havoc-io/mutagen/pkg/filesystem"
)

// EnsureValid ensures that ReceiverStatus' invariants are respected.
func (s *ReceiverStatus) EnsureValid() error {
	// A nil receiver status is valid - it just represents not currently
	// receiving.
	if s == nil {
		return nil
	}

	// Sanity check counts. Any conditions here should be caught by error
	// handling in the receivers and not passed back to any monitoring
	// callbacks.
	if s.Received > s.Total {
		return errors.New("receiver status indicates too many files received")
	}

	// Success.
	return nil
}

// Receiver manages the streaming reception of multiple files. It should be used
// in conjunction with the Transmit function.
type Receiver interface {
	// Receive processes a single message in a transmission stream.
	Receive(*Transmission) error
	// finalize indicates that the transmission stream is completed and that no
	// more messages will be received. This may indicate the successful
	// completion of transmission, but could also indicate that the stream has
	// failed due to an error. In any case, the receiver should use it as an
	// opportunity to close all internal resources. It must be safe to call
	// finalize after an error is returned from Receive.
	finalize() error
}

// Sinker provides the interface for a receiver to store incoming files.
type Sinker interface {
	// Sink should return a new io.WriteCloser for staging the given path. Each
	// result it returns will be closed before Sink is invoked again.
	Sink(path string) (io.WriteCloser, error)
}

// readSeekCloser is the union of io.Reader, io.Seeker, and io.Closer.
type readSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

// emptyReadSeekCloser is an implementation of readSeekCloser that is empty.
type emptyReadSeekCloser struct {
	*bytes.Reader
}

// newEmptyReadSeekCloser constructs a new empty readSeekCloser.
func newEmptyReadSeekCloser() readSeekCloser {
	return &emptyReadSeekCloser{bytes.NewReader(nil)}
}

// Close implements io.Closer for emptyReadSeekCloser.
func (e *emptyReadSeekCloser) Close() error {
	return nil
}

// receiver is a Receiver implementation that actually writes files to disk.
type receiver struct {
	// root is the file root.
	root string
	// paths is the list of paths to receive.
	paths []string
	// signatures is the list of signatures corresponding to the bases for these
	// paths.
	signatures []*Signature
	// opener is the filesystem opener used to open base files.
	opener *fs.Opener
	// sinker is the Sinker to use for staging files.
	sinker Sinker
	// engine is the rsync Engine.
	engine *Engine
	// received is the number of files received.
	received uint64
	// total is the total number of files to receive (the number of paths).
	total uint64
	// finalized indicates whether or not the receiver has been finalized.
	finalized bool
	// burning indicates that the receiver is currently burning operations due
	// to a failed file receiving operation.
	burning bool
	// base is the base for the current file. It should be non-nil if and only
	// if target is non-nil. It should be nil if burning.
	base readSeekCloser
	// target is the destination for the current file. It should be non-nil if
	// and only if base is non-nil. It should be nil if burning.
	target io.WriteCloser
}

// NewReceiver creates a new receiver that stores files on disk. It is the
// responsibility of the caller to ensure that the provided signatures are valid
// by invoking their EnsureValid method. In order for the receiver to perform
// efficiently, paths should be passed in depth-first traversal order.
func NewReceiver(root string, paths []string, signatures []*Signature, sinker Sinker) (Receiver, error) {
	// Ensure that the receiving request is sane.
	if len(paths) != len(signatures) {
		return nil, errors.New("number of paths does not match number of signatures")
	}

	// Create the receiver.
	return &receiver{
		root:       root,
		paths:      paths,
		signatures: signatures,
		opener:     fs.NewOpener(root),
		sinker:     sinker,
		engine:     NewEngine(),
		total:      uint64(len(paths)),
	}, nil
}

// Receive processes incoming messages by storing files to disk.
func (r *receiver) Receive(transmission *Transmission) error {
	// Check that we haven't been finalized.
	if r.finalized {
		panic("receive called on finalized receiver")
	}

	// Make sure that we're not seeing a transmission after receiving all files.
	// If we are, it's a terminal error.
	if r.received == r.total {
		return errors.New("unexpected file transmission")
	}

	// Check if we need to skip this transmission due to burning.
	skip := r.burning

	// Check if this is a done transmission.
	if transmission.Done {
		// TODO: The transmission may have error information here. Should we
		// expose that to whatever is doing the file sinking? It doesn't matter
		// for our application since we have independent hash validation, but it
		// might be useful for some cases.

		// Close out base and target if they're open, because we're done with
		// this file. If they're not open, and we're not burning, it means that
		// we have an empty file. Since we won't have opened any sink for the
		// file (no operations came in for it), open one quickly and close it.
		// Since we're already at the end of the stream for this file, there's
		// no need to start burning operations if this fails.
		if r.base != nil {
			r.base.Close()
			r.base = nil
			r.target.Close()
			r.target = nil
		} else if !r.burning {
			if target, _ := r.sinker.Sink(r.paths[r.received]); target != nil {
				target.Close()
			}
		}

		// Update the received count.
		r.received++

		// Reset burning status.
		r.burning = false

		// Skip the transmission (since it doesn't contain any operation).
		skip = true
	}

	// Skip the transmission if necessary, either due to burning or the fact
	// that it's a done transmission (or both).
	if skip {
		return nil
	}

	// Extract the signature for this file.
	signature := r.signatures[r.received]

	// Check if we are starting a new file stream and need to open the base and
	// target.
	if r.base == nil {
		// Extract the path.
		path := r.paths[r.received]

		// Open the base. If the signature is a zero value, then we just use an
		// empty base. If it's not, then we need to try to open the base. If
		// that fails, then we need to burn this file stream, but it's not a
		// terminal error.
		if signature.isEmpty() {
			r.base = newEmptyReadSeekCloser()
		} else if base, err := r.opener.Open(path); err != nil {
			r.burning = true
			return nil
		} else {
			r.base = base
		}

		// Create a sink. If that fails, then we need to close out the base and
		// burn this file stream, but it's not a terminal error.
		if target, err := r.sinker.Sink(path); err != nil {
			r.base.Close()
			r.base = nil
			r.burning = true
			return nil
		} else {
			r.target = target
		}
	}

	// Apply the operation. If that fails, then we need to close out the base,
	// target, and burn this file stream, but it's not a terminal error.
	if err := r.engine.Patch(r.target, r.base, signature, transmission.Operation); err != nil {
		r.base.Close()
		r.base = nil
		r.target.Close()
		r.target = nil
		r.burning = true
		return nil
	}

	// Success.
	return nil
}

// finalize aborts reception (if still in-progress) closes any open receiver
// resources.
func (r *receiver) finalize() error {
	// Watch for double finalization.
	if r.finalized {
		return errors.New("receiver finalized multiple times")
	}

	// Close any open internal resources.
	if r.base != nil {
		r.base.Close()
		r.base = nil
		r.target.Close()
		r.target = nil
	}

	// Close the file opener.
	r.opener.Close()

	// Mark the receiver as finalized.
	r.finalized = true

	// Success.
	return nil
}

// Monitor is the interface that monitors must implement to capture status
// information from a monitoring receiver. The argument provided to this
// function will be allocated on each update and can be kept by the monitoring
// callback. There's no point in attempting to re-use the allocated argument
// because (a) it would be complicated and the callback would most likely just
// copy it anyway and (b) it will only be allocated once per received file, and
// the per-file allocations are already significantly higher.
type Monitor func(*ReceiverStatus) error

// monitoringReceiver is a Receiver implementation that can invoke a callback
// with information about the status of transmission.
type monitoringReceiver struct {
	// receiver is the underlying receiver.
	receiver Receiver
	// paths is the list of paths the receiver is expecting.
	paths []string
	// received is the number of paths received so far.
	received uint64
	// total is the total number of files to receive (the number of paths).
	total uint64
	// beginning inidicates whether or not we're at the beginning of the message
	// stream (i.e. that no status updates have yet been sent).
	beginning bool
	// monitor is the monitoring callback.
	monitor Monitor
}

// NewMonitoringReceiver wraps a receiver and provides monitoring information
// via a callback.
func NewMonitoringReceiver(receiver Receiver, paths []string, monitor Monitor) Receiver {
	return &monitoringReceiver{
		receiver:  receiver,
		paths:     paths,
		total:     uint64(len(paths)),
		beginning: true,
		monitor:   monitor,
	}
}

// Receive forwards messages to its underlying receiver and performs status
// updates by invoking the specified monitor.
func (r *monitoringReceiver) Receive(transmission *Transmission) error {
	// Forward the transmission to the underlying receiver.
	if err := r.receiver.Receive(transmission); err != nil {
		return err
	}

	// Make sure that we're not seeing a transmission after receiving all files.
	// If we are, it's a terminal error.
	if r.received == r.total {
		return errors.New("unexpected file transmission")
	}

	// Track whether or not we need to send a status update.
	sendStatusUpdate := false

	// If we're at the start of the stream, i.e. we haven't sent any status
	// updates yet, then we should send an update so that some status
	// information comes through before the first file is finished.
	if r.beginning {
		r.beginning = false
		sendStatusUpdate = true
	}

	// If we're at the end of a file stream, update the receive count and ensure
	// that we send a status update.
	if transmission.Done {
		r.received++
		sendStatusUpdate = true
	}

	// Send a status update if necessary.
	if sendStatusUpdate {
		// Compute the path. We know that received <= total due to our check
		// above. If received == total, we use an empty string, since all paths
		// have been received, otherwise we use the path currently being
		// received.
		var path string
		if r.received < r.total {
			path = r.paths[r.received]
		}

		// Send the status.
		status := &ReceiverStatus{
			Path:     path,
			Received: r.received,
			Total:    r.total,
		}
		if err := r.monitor(status); err != nil {
			return errors.Wrap(err, "unable to send receiving status")
		}
	}

	// Success.
	return nil
}

// finalize invokes finalize on the underlying receiver. It also performs a
// final empty status update, though it doesn't check for an error when doing
// so.
func (r *monitoringReceiver) finalize() error {
	// Perform a final status update. We don't bother checking for an error
	// because it's inconsequential at this point.
	r.monitor(nil)

	// Invoke finalize on the underlying receiver.
	return r.receiver.finalize()
}

// preemptableReceiver is a Receiver implementation that provides preemption
// facilities.
type preemptableReceiver struct {
	// receiver is the underlying receiver.
	receiver Receiver
	// run is the context in which the receiver is receiving.
	run context.Context
}

// NewPreemptableReceiver wraps a receiver and aborts on Receive if the
// specified context has been cancelled.
func NewPreemptableReceiver(receiver Receiver, run context.Context) Receiver {
	return &preemptableReceiver{
		receiver: receiver,
		run:      run,
	}
}

// Receive performs a check for preemption, aborting if the receiver has been
// preempted. If no preemption has occurred, the transmission is forwarded to
// the underlying receiver.
func (r *preemptableReceiver) Receive(transmission *Transmission) error {
	// Check for preemption in a non-blocking fashion.
	select {
	case <-r.run.Done():
		return errors.New("reception cancelled")
	default:
	}

	// Forward the transmission.
	return r.receiver.Receive(transmission)
}

// finalize invokes finalize on the underlying receiver.
func (r *preemptableReceiver) finalize() error {
	return r.receiver.finalize()
}

// Encoder is the interface used by an encoding receiver to forward
// transmissions, usually across a network.
type Encoder interface {
	// Encode encodes and transmits a transmission. The provided transmission
	// will never be nil. The transmission passed to the encoder may be re-used
	// and modified, so the encoder should not hold on to the transmission
	// between calls (it should either transmit it or fully copy it if
	// transmission is going to be delayed).
	Encode(*Transmission) error
	// Finalize is called when the transmission stream is finished. The Encoder
	// can use this call to close any underlying transmission resources.
	Finalize() error
}

// encodingReceiver is a Receiver implementation that encodes messages to an
// arbitrary encoder.
type encodingReceiver struct {
	// encoder is the Encoder to use for encoding messages.
	encoder Encoder
	// finalized indicates whether or not the receiver has been finalized.
	finalized bool
}

// NewEncodingReceiver creates a new receiver that handles messages by encoding
// them with the specified Encoder. It is designed to be used with
// DecodeToReceiver.
func NewEncodingReceiver(encoder Encoder) Receiver {
	return &encodingReceiver{
		encoder: encoder,
	}
}

// Receive encodes the specified transmission using the underlying encoder.
func (r *encodingReceiver) Receive(transmission *Transmission) error {
	return errors.Wrap(r.encoder.Encode(transmission), "unable to encode transmission")
}

// finalize finalizes the encoding receiver, which means that it calls Finalize
// on its underlying Encoder.
func (r *encodingReceiver) finalize() error {
	// Watch for double finalization.
	if r.finalized {
		return errors.New("receiver finalized multiple times")
	}

	// Mark ourselves as finalized
	r.finalized = true

	// Finalize the encoder.
	if err := r.encoder.Finalize(); err != nil {
		return errors.Wrap(err, "unable to finalize encoder")
	}

	// Success.
	return nil
}

// Encoder is the interface used by DecodeToReceiver to receive transmissions,
// usually across a network.
type Decoder interface {
	// Decoder decodes a transmission encoded by an encoder. The transmission
	// should be decoded into the specified Transmission object, which will be a
	// non-nil zero-valued Transmission object. The decoder is *not* responsible
	// for validating that the transmission is valid before returning it.
	// TODO: We should really elaborate on the semantics of Decoder, in
	// particular how it is allowed to re-use existing allocations within the
	// Transmission object.
	Decode(*Transmission) error
	// Finalize is called when decoding is finished. The Decoder can use this
	// call to close any underlying transmission resources.
	Finalize() error
}

// DecodeToReceiver decodes messages from the specified Decoder and forwards
// them to the specified receiver. It must be passed the number of files to be
// received so that it knows when forwarding is complete. It is designed to be
// used with an encoding receiver, such as that returned by NewEncodingReceiver.
// It finalizes the provided receiver before returning.
func DecodeToReceiver(decoder Decoder, count uint64, receiver Receiver) error {
	// Allocate the transmission object that we'll use to receive into.
	transmission := &Transmission{}

	// Loop until we've seen all files come in.
	for count > 0 {
		// Loop, decode, and forward until we see a done message.
		for {
			// Receive the next message.
			transmission.resetToZeroMaintainingCapacity()
			if err := decoder.Decode(transmission); err != nil {
				decoder.Finalize()
				receiver.finalize()
				return errors.Wrap(err, "unable to decode transmission")
			}

			// Validate the transmission.
			if err := transmission.EnsureValid(); err != nil {
				decoder.Finalize()
				receiver.finalize()
				return errors.Wrap(err, "invalid transmission received")
			}

			// Forward the message.
			if err := receiver.Receive(transmission); err != nil {
				decoder.Finalize()
				receiver.finalize()
				return errors.Wrap(err, "unable to forward message to receiver")
			}

			// If the message indicates completion, we're done receiving
			// messages for this file.
			if transmission.Done {
				break
			}
		}

		// Update the count.
		count--
	}

	// Ensure that the decoder is finalized.
	if err := decoder.Finalize(); err != nil {
		receiver.finalize()
		return errors.Wrap(err, "unable to finalize decoder")
	}

	// Ensure that the receiver is finalized.
	if err := receiver.finalize(); err != nil {
		return errors.Wrap(err, "unable to finalize receiver")
	}

	// Done.
	return nil
}
