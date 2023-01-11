//go:build sspl

package synchronization

import (
	"hash"

	"github.com/mutagen-io/mutagen/sspl/pkg/hashing/xxh128"
)

const (
	// digestXXH128Supported indicates whether or not XXH128 digests are
	// supported.
	digestXXH128Supported = true
)

// newXXH128Factory creates a new hasher factory for XXH128 hashers.
func newXXH128Factory() func() hash.Hash {
	return xxh128.New
}
