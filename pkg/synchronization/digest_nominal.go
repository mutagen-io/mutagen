//go:build !sspl

package synchronization

import (
	"hash"
)

const (
	// digestXXH128Supported indicates whether or not XXH128 digests are
	// supported.
	digestXXH128Supported = false
)

// newXXH128Factory creates a new hasher factory for XXH128 hashers.
func newXXH128Factory() func() hash.Hash {
	panic("XXH128 unsupported")
}
