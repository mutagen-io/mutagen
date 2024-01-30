//go:build mutagensspl

package compression

import (
	"io"

	"github.com/mutagen-io/mutagen/pkg/stream"

	"github.com/mutagen-io/mutagen/sspl/pkg/compression/zstd"
)

// zstandardSupportStatus returns Zstandard compression support status.
func zstandardSupportStatus() AlgorithmSupportStatus {
	return AlgorithmSupportStatusSupported
}

// compressZstandard implements compression for Zstandard streams.
func compressZstandard(compressed io.Writer) stream.WriteFlushCloser {
	return zstd.NewCompressor(compressed)
}

// decompressZstandard implements decompression for Zstandard streams.
func decompressZstandard(compressed io.Reader) io.ReadCloser {
	return zstd.NewDecompressor(compressed)
}
