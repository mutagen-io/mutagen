package multiplex

import (
	"bytes"
	"testing"
)

func TestHeaderTransport(t *testing.T) {
	// Define test cases.
	testCases := []header{
		{0, 0},
		{0, 1},
		{0, 12345678},
		{0, maxBlockLength},
		{1, 0},
		{1, 1},
		{1, 12345678},
		{1, maxBlockLength},
		{100, 0},
		{100, 1},
		{100, 12345678},
		{100, maxBlockLength},
		{255, 0},
		{255, 1},
		{255, 12345678},
		{255, maxBlockLength},
	}

	// Create our transport.
	transport := &bytes.Buffer{}

	// Encode them.
	for i, c := range testCases {
		if err := c.write(transport); err != nil {
			t.Fatalf("unable to encode header %d: %v", i, err)
		}
	}

	// Decode them.
	for i, c := range testCases {
		if decoded, err := readHeader(transport); err != nil {
			t.Fatalf("unable to encode header %d: %v", i, err)
		} else if decoded.channel != c.channel || decoded.length != c.length {
			t.Error("header mismatch at index ", i)
		}
	}
}
