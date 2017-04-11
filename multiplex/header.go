package multiplex

import (
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
)

const (
	// maxBlockLength is the maximum block length that can be encoded by a
	// header.
	maxBlockLength = 1<<32 - 1
)

// header is a tag-length pair that preceeds all data blocks on the wire. It
// encodes the channel and length for the block.
type header struct {
	// channel is the channel for the subsequent data block.
	channel uint8
	// length is the length of the subsequent data block.
	length uint32
}

// readHeader reads a header from a stream. It is recommended that the stream be
// buffered to avoid the overhead of short reads.
func readHeader(reader io.Reader) (header, error) {
	// Read the channel. If we get an EOF here, then we return it unwrapped,
	// because this is a "natural" EOF boundary.
	var channelBytes [1]byte
	if _, err := io.ReadFull(reader, channelBytes[:]); err != nil {
		if err == io.EOF {
			return header{}, io.EOF
		}
		return header{}, errors.Wrap(err, "unable to read channel for header")
	}
	channel := uint8(channelBytes[0])

	// Read the length.
	var lengthBytes [4]byte
	if _, err := io.ReadFull(reader, lengthBytes[:]); err != nil {
		return header{}, errors.Wrap(err, "unable to read length for header")
	}
	length := binary.BigEndian.Uint32(lengthBytes[:])

	// Success.
	return header{channel, length}, nil
}

// write encodes a header to a stream. It is recommended that the stream be
// buffered to avoid the overhead of short writes.
func (m header) write(writer io.Writer) error {
	// Write the channel.
	channelBytes := [1]byte{byte(m.channel)}
	if _, err := writer.Write(channelBytes[:]); err != nil {
		return errors.Wrap(err, "unable to write channel for header")
	}

	// Write the length.
	var lengthBytes [4]byte
	binary.BigEndian.PutUint32(lengthBytes[:], m.length)
	if _, err := writer.Write(lengthBytes[:]); err != nil {
		return errors.Wrap(err, "unable to write length for header")
	}

	// Success.
	return nil
}
