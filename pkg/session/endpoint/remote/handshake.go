package remote

import (
	"fmt"
	"io"
)

// magicNumberBytes is a type capable of holding a Mutagen magic byte sequence.
type magicNumberBytes [3]byte

// serverMagicNumber is a byte sequence that is sent by an endpoint server to
// identify the start of a Mutagen protocol stream. It is intentionally composed
// of bytes that are not (all) printable ASCII characters. The purpose of adding
// this magic number to the beginning of streams is to work around agent-style
// transports where the underlying transport executable may write error output
// to standard output, which would otherwise be interpreted as version
// information. By identifying this magic number, we can be sure that we're
// talking to a Mutagen stream before we start exchanging version information.
var serverMagicNumber = magicNumberBytes{0x05, 0x27, 0x87}

// clientMagicNumber serves the same purpose as serverMagicNumber, but it is
// send by the endpoint client to the endpoint server. It is not as necessary,
// since whatever connects to the server should already know what it's doing,
// but it serves as an extra sanity check in the world of agent-style
// transports.
var clientMagicNumber = magicNumberBytes{0x87, 0x27, 0x05}

// sendMagicNumber sends the Mutagen magic byte sequence to the specified
// writer.
func sendMagicNumber(writer io.Writer, magicNumber magicNumberBytes) error {
	_, err := writer.Write(magicNumber[:])
	return err
}

// receiveAndCompareMagicNumber reads a Mutagen magic byte sequence from the
// specified reader and verifies that it matches what's expected.
func receiveAndCompareMagicNumber(reader io.Reader, expected magicNumberBytes) (bool, error) {
	// Read the bytes.
	var received magicNumberBytes
	if _, err := io.ReadFull(reader, received[:]); err != nil {
		return false, err
	}

	// Compare the bytes.
	return received == expected, nil
}

// handshakeTransportError indicates a handshake error due to a transport
// failure.
type handshakeTransportError struct {
	// underlying is the underlying error that we know is due to a transport
	// failure during the handshake process.
	underlying error
}

// Error returns a formatted version of the transport error.
func (e *handshakeTransportError) Error() string {
	return fmt.Sprintf("handshake transport error: %v", e.underlying)
}

// IsHandshakeTransportError indicates whether or not an error value is a
// handshake transport error.
func IsHandshakeTransportError(err error) bool {
	_, ok := err.(*handshakeTransportError)
	return ok
}
