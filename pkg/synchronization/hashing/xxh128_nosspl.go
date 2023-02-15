//go:build !mutagensspl

package hashing

import (
	"hash"
)

// xxh128SupportStatus returns XXH128 hashing support status.
func xxh128SupportStatus() AlgorithmSupportStatus {
	return AlgorithmSupportStatusUnsupported
}

// newXXH128Factory creates a new hasher factory for XXH128 hashers.
func newXXH128Factory() func() hash.Hash {
	panic("XXH128 unsupported")
}
