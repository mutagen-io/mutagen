//go:build !sspl

package hashing

import (
	"hash"
)

// xxh128Supported indicates whether or not XXH128 hashing is supported.
const xxh128Supported = false

// newXXH128Factory creates a new hasher factory for XXH128 hashers.
func newXXH128Factory() func() hash.Hash {
	panic("XXH128 unsupported")
}
