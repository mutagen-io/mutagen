package tunneling

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pion/webrtc/v2"

	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/tunneling/webrtcutil"
)

const (
	// heartbeatInterval is the interval at which heartbeats are sent.
	heartbeatInterval = 5 * time.Second
	// heartbeatTimeout is the time after which a connection is considered timed
	// out if not heartbeat request/response is received. Given a specific
	// heartbeatInterval value, this timeout effectively regulates the maximum
	// 1-way network latency.
	heartbeatTimeout = heartbeatInterval + 3*time.Second
	// heartbeatMagic is the magic number used for hearbeats.
	heartbeatMagic = 0x23571113
)

// ensureValid ensures that HeartbeatVersion1's invariants are respected.
func (h *HeartbeatVersion1) ensureValid() error {
	// Ensure that the heartbeat is non-nil.
	if h == nil {
		return errors.New("nil heartbeat")
	}

	// Ensure that the magic number is correct.
	if h.Magic != heartbeatMagic {
		return errors.New("invalid magic number")
	}

	// Success.
	return nil
}

// heartbeat performs a bidirectional heartbeat on the specified data channel,
// erroring out on cancellation, connection failure, or heartbeat timeout.
func heartbeat(ctx context.Context, dataChannel *webrtc.DataChannel, version Version) error {
	// Convert the data channel to a connection and defer its closure.
	connection := webrtcutil.NewConnection(dataChannel, nil)
	defer connection.Close()

	// Wait for the data channel connection to be established (or error out).
	// Connections are lazily initiated once the data channel is opened, so we
	// don't want to count that establishment period against the heartbeat
	// timeout window.
	if err := connection.WaitUntilConnected(); err != nil {
		return fmt.Errorf("connection failure: %w", err)
	}

	// Create a watchdog Goroutine to monitor for timeout or cancellation. Note
	// that we ensure termination of the watchdog timer in the receiver
	// Goroutine since it's responsible for managing the timer.
	watchdogTimer := time.NewTimer(heartbeatTimeout)
	watchdogCtx, watchdogCancel := context.WithCancel(ctx)
	defer watchdogCancel()
	watchdogFailures := make(chan error, 1)
	go func() {
		select {
		case <-watchdogCtx.Done():
			watchdogFailures <- errors.New("cancelled")
		case <-watchdogTimer.C:
			watchdogFailures <- errors.New("heartbeat timeout")
		}
	}()

	// Create a Goroutine to receive heartbeats and monitor for its failure.
	// This Goroutine is also responsible for ensuring termination of the
	// watchdog timer.
	decoder := encoding.NewProtobufDecoder(connection)
	receiverFailures := make(chan error, 1)
	inboundHeartbeat := &HeartbeatVersion1{}
	go func() {
		defer watchdogTimer.Stop()
		for {
			inboundHeartbeat.Magic = 0
			if err := decoder.Decode(inboundHeartbeat); err != nil {
				receiverFailures <- fmt.Errorf("unable to receive heartbeat: %w", err)
				return
			} else if err = inboundHeartbeat.ensureValid(); err != nil {
				receiverFailures <- fmt.Errorf("received invalid heartbeat: %w", err)
				return
			}
			watchdogTimer.Reset(heartbeatTimeout)
		}
	}()

	// Create a Goroutine to send heartbeats and monitor for failure.
	encoder := encoding.NewProtobufEncoder(connection)
	senderFailures := make(chan error, 1)
	outboundHeartbeat := &HeartbeatVersion1{Magic: heartbeatMagic}
	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()
		for {
			<-ticker.C
			if err := encoder.Encode(outboundHeartbeat); err != nil {
				senderFailures <- fmt.Errorf("unable to send heartbeat: %w", err)
				return
			}
		}
	}()

	// Wait for cancellation, timeout, or failure.
	select {
	case <-ctx.Done():
		return errors.New("cancelled")
	case err := <-watchdogFailures:
		return err
	case err := <-receiverFailures:
		return err
	case err := <-senderFailures:
		return err
	}
}
