package configuration

import (
	"github.com/dustin/go-humanize"
)

// ByteSize is a uint64 value that supports unmarshalling from both
// human-friendly string representations and numeric representations. It can be
// cast to a uint64 value, where it represents a byte count.
type ByteSize uint64

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files.
func (s *ByteSize) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Parse and store the value.
	value, err := humanize.ParseBytes(text)
	if err != nil {
		return err
	}
	*s = ByteSize(value)

	// Success.
	return nil
}
