//go:build mutagensspl

// Copyright (c) 2023-present Docker, Inc.
//
// This program is free software: you can redistribute it and/or modify it under
// the terms of the Server Side Public License, version 1, as published by
// MongoDB, Inc.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE. See the Server Side Public License for more
// details.
//
// You should have received a copy of the Server Side Public License along with
// this program. If not, see
// <http://www.mongodb.com/licensing/server-side-public-license>.

package zstd

import (
	"io"

	"github.com/klauspost/compress/zstd"

	"github.com/mutagen-io/mutagen/pkg/stream"
)

// NewDecompressor creates a new Zstandard decompressor that reads from the
// specified stream with a default configuration.
func NewDecompressor(compressed io.Reader) io.ReadCloser {
	// Create the decompressor. We check for errors, but we don't include them
	// as part of the interface because they can only occur with an invalid
	// decompressor configuration (which can't occur when we only use defaults).
	decompressor, err := zstd.NewReader(compressed)
	if err != nil {
		panic("Zstandard decompressor construction failed")
	}

	// Adapt the decompressor to the expected interface.
	return decompressor.IOReadCloser()
}

// NewCompressor creates a new Zstandard compressor that writes to the specified
// stream with a default configuration.
func NewCompressor(compressed io.Writer) stream.WriteFlushCloser {
	// Create the compressor. We check for errors, but we don't include them as
	// part of the interface because they can only occur with an invalid
	// compressor configuration (which can't occur when we only use defaults).
	compressor, err := zstd.NewWriter(compressed)
	if err != nil {
		panic("Zstandard compressor construction failed")
	}

	// Success.
	return compressor
}
