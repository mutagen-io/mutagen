//go:build sspl

package hashing

import (
	"hash"

	"github.com/mutagen-io/mutagen/sspl/pkg/hashing/xxh128"
)

// xxh128Supported indicates whether or not XXH128 hashing is supported.
const xxh128Supported = true

// newXXH128Factory creates a new hasher factory for XXH128 hashers.
func newXXH128Factory() func() hash.Hash {
	return xxh128.New
}
