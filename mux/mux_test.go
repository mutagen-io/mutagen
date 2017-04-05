package mux

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"math/rand"
	"testing"

	errorspkg "github.com/pkg/errors"
)

const (
	nChannels                    = 255
	maximumNumberOfMessagesPlus1 = 1000
	maximumValuesPerMessagePlus1 = 30
	maximumBytesPerMessagePlus1  = 100
)

// message is an internal type consisting of various fields that will be
// randomly populated and then send over a multiplexed stream in gob encoding as
// well as over a channel for verification.
type message struct {
	Index   uint32
	Values  []int
	Bytes   []byte
	Message string
}

func newMessage(index uint32, random *rand.Rand) message {
	// Create values.
	nValues := random.Uint32() % maximumValuesPerMessagePlus1
	values := make([]int, nValues)
	for i := uint32(0); i < nValues; i++ {
		values[i] = random.Int()
	}

	// Create bytes. Note that the random read method can never fail.
	nBytes := random.Uint32() % maximumBytesPerMessagePlus1
	bytes := make([]byte, nBytes)
	random.Read(bytes)

	// Create message.
	messageString := fmt.Sprintf("message %d", index)

	// Done.
	return message{
		Index:   index,
		Values:  values,
		Bytes:   bytes,
		Message: messageString,
	}
}

// equal compares one message to another, returning true if they are equal.
func (m message) equal(other message) bool {
	// Check that the Index is the same.
	if m.Index != other.Index {
		return false
	}

	// Check that values are the same.
	if len(m.Values) != len(other.Values) {
		return false
	}
	for i, v := range m.Values {
		if other.Values[i] != v {
			return false
		}
	}

	// Check that bytes are the same.
	if !bytes.Equal(m.Bytes, other.Bytes) {
		return false
	}

	// Check that the message is the same.
	if m.Message != other.Message {
		return false
	}

	// Success.
	return true
}

// sendMessages dispatches random messages over a stream and a channel.
func sendMessages(channel uint8, stream io.Writer, errors chan error) {
	// Create a reproducable random number generator.
	random := rand.New(rand.NewSource(int64(channel)))

	// Create an encoder.
	encoder := gob.NewEncoder(stream)

	// Compute the number of messages to send.
	nMessages := random.Uint32() % maximumNumberOfMessagesPlus1

	// Dispatch messages.
	for i := uint32(0); i < nMessages; i++ {
		if err := encoder.Encode(newMessage(i, random)); err != nil {
			errors <- errorspkg.Wrap(err, "unable to encode message")
			return
		}
	}

	// Success.
	errors <- nil
}

func receiveMessages(channel uint8, stream io.Reader, errors chan error) {
	// Create a reproducable random number generator.
	random := rand.New(rand.NewSource(int64(channel)))

	// Create a decoder.
	decoder := gob.NewDecoder(stream)

	// Compute the number of messages to receive.
	nMessages := random.Uint32() % maximumNumberOfMessagesPlus1

	// Receive and verify messages.
	for i := uint32(0); i < nMessages; i++ {
		expected := newMessage(i, random)
		var actual message
		if err := decoder.Decode(&actual); err != nil {
			errors <- errorspkg.Wrap(err, "unable to decode message")
			return
		} else if !actual.equal(expected) {
			errors <- errorspkg.Errorf("message mismatch at index %d", i)
			return
		}
	}

	// Perform a final read from the stream and ensure that it returns an error.
	var buffer [1]byte
	if n, err := stream.Read(buffer[:]); err == nil {
		errors <- errorspkg.New("read after messages did not error")
		return
	} else if n != 0 {
		errors <- errorspkg.New("read after messages returned data")
		return
	}

	// Success.
	errors <- nil
}

func TestMux(t *testing.T) {
	// Create a transport and multiplex it.
	reader, writer := io.Pipe()
	readers, readersCloser := Reader(reader, nChannels)
	writers := Writer(writer, nChannels)

	// Create an errors channel.
	errors := make(chan error)

	// Start senders.
	for c := uint8(0); c < nChannels; c++ {
		go sendMessages(c, writers[c], errors)
	}

	// Start receivers.
	for c := uint8(0); c < nChannels; c++ {
		go receiveMessages(c, readers[c], errors)
	}

	// Wait for all senders to finish, watching for errors (which could even
	// come from receivers that finish prematurely).
	for c := uint8(0); c < nChannels; c++ {
		err := <-errors
		if err != nil {
			t.Fatal("transmission error:", err)
		}
	}

	// TODO: Can we also close the underlying pipe and verify that the poller
	// Goroutine terminates?

	// Close the receiver pipes and verify that all receivers terminate.
	readersCloser.Close()
	for c := uint8(0); c < nChannels; c++ {
		err := <-errors
		if err != nil {
			t.Fatal("receiver error:", err)
		}
	}
}
