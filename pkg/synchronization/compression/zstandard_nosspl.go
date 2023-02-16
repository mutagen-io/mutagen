//go:build !mutagensspl

package compression

import (
	"io"

	"github.com/mutagen-io/mutagen/pkg/stream"
)

// zstandardSupportStatus returns Zstandard compression support status.
func zstandardSupportStatus() AlgorithmSupportStatus {
	return AlgorithmSupportStatusUnsupported
}

// compressZstandard implements compression for Zstandard streams.
func compressZstandard(compressed io.Writer) stream.WriteFlushCloser {
	panic("Zstandard compression not supported")
}

// decompressZstandard implements decompression for Zstandard streams.
func decompressZstandard(compressed io.Reader) io.ReadCloser {
	panic("Zstandard decompression not supported")
}
