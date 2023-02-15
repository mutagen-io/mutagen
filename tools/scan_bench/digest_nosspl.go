//go:build !mutagensspl

package main

import (
	"hash"
)

const (
	// digestFlagOptions are the options to display for the -d/--digest flag.
	digestFlagOptions = "sha1|sha256"

	// xxh128Supported indicated whether or not XXH128 hashing is supported.
	xxh128Supported = false
)

// newXXH128Hasher creates a new XXH128 hash function.
func newXXH128Hasher() hash.Hash {
	panic("XXH128 not supported")
}
