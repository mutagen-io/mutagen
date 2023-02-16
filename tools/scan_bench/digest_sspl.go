//go:build mutagensspl

package main

import (
	"hash"

	"github.com/mutagen-io/mutagen/sspl/pkg/hashing/xxh128"
)

const (
	// digestFlagOptions are the options to display for the -d/--digest flag.
	digestFlagOptions = "sha1|sha256|xxh128"

	// xxh128Supported indicated whether or not XXH128 hashing is supported.
	xxh128Supported = true
)

// newXXH128Hasher creates a new XXH128 hash function.
func newXXH128Hasher() hash.Hash {
	return xxh128.New()
}
