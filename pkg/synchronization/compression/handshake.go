package compression

import (
	"errors"
	"fmt"
	"io"
)

// errAlgorithmUnsupported indicates that the algorithm requested by the client
// is unsupported by the server.
var errAlgorithmUnsupported = errors.New("algorithm unsupported")

// ClientHandshake performs a client-side compression handshake on the stream.
// It transmits the desired compression algorithm and verifies that this
// algorithm is supported by the server.
func ClientHandshake(stream io.ReadWriter, algorithm Algorithm) error {
	// Verify that the algorithm can be encoded into a single byte.
	if algorithm < 0 || algorithm > 255 {
		return errors.New("invalid algorithm value")
	}

	// Convert the algorithm specification.
	data := [1]byte{byte(algorithm)}

	// Transmit the data.
	if _, err := stream.Write(data[:]); err != nil {
		return fmt.Errorf("unable to transmit algorithm specification: %w", err)
	}

	// Receive the response.
	if _, err := io.ReadFull(stream, data[:]); err != nil {
		return fmt.Errorf("unable to receive response: %w", err)
	}

	// Handle the response.
	switch data[0] {
	case 0:
		return errAlgorithmUnsupported
	case 1:
		return nil
	default:
		return errors.New("invalid response from server")
	}
}

// ServerHandshake performs a server-side compression handshake on the stream.
// It receives the desired compression algorithm from the client, verifies that
// this algorithm is supported, and transmits a response to the client.
func ServerHandshake(stream io.ReadWriter) (Algorithm, error) {
	// Receive the algorithm specification.
	var data [1]byte
	if _, err := io.ReadFull(stream, data[:]); err != nil {
		return Algorithm_AlgorithmDefault, fmt.Errorf("unable to receive algorithm specification: %w", err)
	}

	// Convert the algorithm specification and ensure that it's supported.
	algorithm := Algorithm(data[0])
	supported := algorithm.Supported()

	// Format and transmit the response.
	if supported {
		data[0] = 1
	} else {
		data[0] = 0
	}
	if _, err := stream.Write(data[:]); err != nil {
		return Algorithm_AlgorithmDefault, fmt.Errorf("unable to transmit response: %w", err)
	}

	// Handle unsupported algorithms.
	if !supported {
		return Algorithm_AlgorithmDefault, errAlgorithmUnsupported
	}

	// Success.
	return algorithm, nil
}
