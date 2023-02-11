//go:build sspl

package hashing

import (
	"hash"

	"github.com/mutagen-io/mutagen/sspl/pkg/hashing/xxh128"
)

// newXXH128Factory creates a new hasher factory for XXH128 hashers.
func newXXH128Factory() func() hash.Hash {
	return xxh128.New
}
