package ipc

import (
	"bytes"
	"context"
	"encoding/gob"
	"path/filepath"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"
)

// TestDialContextNoEndpoint tests that DialContext fails if there is no
// endpoint at the specified path.
func TestDialTimeoutNoEndpoint(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Compute the IPC endpoint path.
	endpoint := filepath.Join(t.TempDir(), "test.sock")

	// Attempt to dial the listener and ensure that doing so fails.
	if c, err := DialContext(context.Background(), endpoint); err == nil {
		must.Close(c, logger)
		t.Error("IPC connection succeeded unexpectedly")
	}
}

// TODO: Add TestDialContextNoListener to test cases where a stale socket has
// been left on disk, ensuring that DialContext fails to connect.

// testIPCMessage is a structure used to test IPC messaging.
type testIPCMessage struct {
	// Name represents a person's name.
	Name string
	// Age represents a person's age.
	Age uint
}

// TestIPC tests that an IPC connection can be established between a listener
// and a dialer.
func TestIPC(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Create a test message.
	expected := testIPCMessage{"George", 67}

	// Compute the IPC endpoint path.
	endpoint := filepath.Join(t.TempDir(), "test.sock")

	// Create a listener and defer its closure.
	listener, err := NewListener(endpoint, logger)
	if err != nil {
		t.Fatal("unable to create listener:", err)
	}
	defer must.Close(listener, logger)

	// Perform dialing and message sending in a separate Goroutine.
	go func() {
		// Dial and defer connection closure.
		connection, err := DialContext(context.Background(), endpoint)
		if err != nil {
			return
		}
		defer must.Close(connection, logger)

		// Create an encoder.
		encoder := gob.NewEncoder(connection)

		// Send a test message.
		must.Encode(encoder, expected, logger)
	}()

	// Accept a connection and defer its closure.
	connection, err := listener.Accept()
	if err != nil {
		t.Fatal("unable to accept connection:", err)
	}
	defer must.Close(connection, logger)

	// Create a decoder.
	decoder := gob.NewDecoder(connection)

	// Receive and validate test message.
	var received testIPCMessage
	if err := decoder.Decode(&received); err != nil {
		t.Fatal("unable to receive test message:", err)
	} else if received != expected {
		t.Error("received message does not match expected:", received, "!=", expected)
	}
}
