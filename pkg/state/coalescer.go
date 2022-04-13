package state

import (
	"context"
	"time"
)

// Coalescer performs coalesced signaling, combining multiple signals that occur
// within a specified time window. A Coalescer is safe for concurrent usage. It
// maintains a background Goroutine that must be terminated using Terminate.
type Coalescer struct {
	// strobes is used to transmit strobes to the run loop.
	strobes chan struct{}
	// signals is the channel on which signals are delivered.
	signals chan struct{}
	// cancel signals termination to the run loop.
	cancel context.CancelFunc
	// done is closed to indicate that the run loop has exited.
	done chan struct{}
}

// NewCoalescer creates a new coalescer that will group signals that occur
// within the specified time window of each other. If window is negative, it
// will be treated as zero.
func NewCoalescer(window time.Duration) *Coalescer {
	// If the specified window is negative, then treat it as zero.
	if window < 0 {
		window = 0
	}

	// Create a cancellable context to regulate the run loop.
	ctx, cancel := context.WithCancel(context.Background())

	// Create the coalescer.
	coalescer := &Coalescer{
		strobes: make(chan struct{}),
		signals: make(chan struct{}, 1),
		cancel:  cancel,
		done:    make(chan struct{}),
	}

	// Start the coalescer's run loop.
	go coalescer.run(ctx, window)

	// Done.
	return coalescer
}

// run implements the signal processing run loop for Coalescer.
func (c *Coalescer) run(ctx context.Context, window time.Duration) {
	// Create the (initially stopped) coalescing timer.
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}

	// Loop and process events until cancelled.
	for {
		select {
		case <-ctx.Done():
			timer.Stop()
			close(c.done)
			return
		case <-c.strobes:
			timer.Stop()
			select {
			case <-timer.C:
			default:
			}
			timer.Reset(window)
		case <-timer.C:
			select {
			case c.signals <- struct{}{}:
			default:
			}
		}
	}
}

// Strobe enqueues a signal to be sent after the coalescing window. If a
// subsequent call to Strobe is made within the coalescing window, then it will
// reset the coalescing timer and an event will only be sent after Strobe hasn't
// been called for the coalescing window period.
func (c *Coalescer) Strobe() {
	select {
	case c.strobes <- struct{}{}:
	case <-c.done:
	}
}

// Signals returns the signal notification channel. This channel is buffered
// with a capacity of 1, so no signaling will ever be lost if it's not actively
// polled. The resulting channel is never closed.
func (c *Coalescer) Signals() <-chan struct{} {
	return c.signals
}

// Terminate shuts down the coalescer's internal run loop and waits for it to
// terminate. It's safe to continue invoking other methods after invoking
// Terminate (including Terminate, which is idempotent), though Strobe will have
// no effect and only previously buffered events will be delivered on the
// channel returned by Signals.
func (c *Coalescer) Terminate() {
	c.cancel()
	<-c.done
}
