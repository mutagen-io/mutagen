//go:build mutagensspl

package hashing

import (
	"hash"

	"github.com/mutagen-io/mutagen/sspl/pkg/hashing/xxh128"
)

// xxh128SupportStatus returns XXH128 hashing support status.
func xxh128SupportStatus() AlgorithmSupportStatus {
	return AlgorithmSupportStatusSupported
}

// newXXH128Factory creates a new hasher factory for XXH128 hashers.
func newXXH128Factory() func() hash.Hash {
	return xxh128.New
}
