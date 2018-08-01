package daemon

import (
	"encoding/gob"
	"testing"
)

// TestDialTimeoutNoListener tests that DialTimeout fails if there is no active
// listener.
func TestDialTimeoutNoListener(t *testing.T) {
	if c, err := DialTimeout(RecommendedDialTimeout); err == nil {
		c.Close()
		t.Error("IPC connection succeeded unexpectedly")
	}
}

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
	// Create a test message.
	expected := testIPCMessage{"George", 67}

	// Create a listener and defer its closure.
	listener, err := NewListener()
	if err != nil {
		t.Fatal("unable to create listener:", err)
	}
	defer listener.Close()

	// Perform dialing and message sending in a separate Goroutine.
	go func() {
		// Dial and defer connection closure.
		connection, err := DialTimeout(RecommendedDialTimeout)
		if err != nil {
			return
		}
		defer connection.Close()

		// Create an encoder.
		encoder := gob.NewEncoder(connection)

		// Send a test message.
		encoder.Encode(expected)
	}()

	// Accept a connection and defer its closure.
	connection, err := listener.Accept()
	if err != nil {
		t.Fatal("unable to accept connection:", err)
	}
	defer connection.Close()

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
