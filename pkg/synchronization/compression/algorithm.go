package compression

import (
	"fmt"
	"io"

	"github.com/mutagen-io/mutagen/pkg/stream"
)

// IsDefault indicates whether or not the algorithm is
// Algorithm_AlgorithmDefault.
func (a Algorithm) IsDefault() bool {
	return a == Algorithm_AlgorithmDefault
}

// MarshalText implements encoding.TextMarshaler.MarshalText.
func (a Algorithm) MarshalText() ([]byte, error) {
	var result string
	switch a {
	case Algorithm_AlgorithmDefault:
	case Algorithm_AlgorithmNone:
		result = "none"
	case Algorithm_AlgorithmDeflate:
		result = "deflate"
	case Algorithm_AlgorithmZstandard:
		result = "zstandard"
	default:
		result = "unknown"
	}
	return []byte(result), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText.
func (a *Algorithm) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a compression algorithm.
	switch text {
	case "none":
		*a = Algorithm_AlgorithmNone
	case "deflate":
		*a = Algorithm_AlgorithmDeflate
	case "zstandard":
		*a = Algorithm_AlgorithmZstandard
	default:
		return fmt.Errorf("unknown compression algorithm specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular compression algorithm is a
// valid, non-default value.
func (a Algorithm) Supported() bool {
	switch a {
	case Algorithm_AlgorithmNone:
		return true
	case Algorithm_AlgorithmDeflate:
		return true
	case Algorithm_AlgorithmZstandard:
		return zstandardSupported
	default:
		return false
	}
}

// Description returns a human-readable description of a compression algorithm.
func (a Algorithm) Description() string {
	switch a {
	case Algorithm_AlgorithmDefault:
		return "Default"
	case Algorithm_AlgorithmNone:
		return "None"
	case Algorithm_AlgorithmDeflate:
		return "DEFLATE"
	case Algorithm_AlgorithmZstandard:
		return "Zstandard"
	default:
		return "Unknown"
	}
}

// Compress creates a compressor that writes compressed output to the specified
// stream using the compression algorithm. If invoked on a default or invalid
// Algorithm value, this method will panic. The Flush and Close methods on the
// resulting compressor only operate on the compressor - they have no effect on
// the compressed stream itself. The compressor should be flushed and/or closed
// before the underlying stream.
func (a Algorithm) Compress(compressed io.Writer) stream.WriteFlushCloser {
	switch a {
	case Algorithm_AlgorithmNone:
		return compressNone(compressed)
	case Algorithm_AlgorithmDeflate:
		return compressDeflate(compressed)
	case Algorithm_AlgorithmZstandard:
		return compressZstandard(compressed)
	default:
		panic("default or unknown compression algorithm")
	}
}

// Decompress creates a decompressor that reads compressed input from the
// specified stream using the compression algorithm. If invoked on a default or
// invalid Algorithm value, this method will panic. The Close method on the
// resulting decompressor releases decompression resources - it has no effect on
// the compressed stream itself. The decompressor should be closed after the
// underlying stream.
func (a Algorithm) Decompress(compressed io.Reader) io.ReadCloser {
	switch a {
	case Algorithm_AlgorithmNone:
		return decompressNone(compressed)
	case Algorithm_AlgorithmDeflate:
		return decompressDeflate(compressed)
	case Algorithm_AlgorithmZstandard:
		return decompressZstandard(compressed)
	default:
		panic("default or unknown compression algorithm")
	}
}
