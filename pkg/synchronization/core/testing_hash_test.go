package core

import (
	"crypto/sha1"
	"hash"
)

// newTestingHasher creates a new hash function to use for testing.
func newTestingHasher() hash.Hash {
	return sha1.New()
}

// testingDigest computes the digest of a string using the same algorithm as the
// hash function returned by newTestingHasher.
func testingDigest(content string) []byte {
	hasher := newTestingHasher()
	hasher.Write([]byte(content))
	return hasher.Sum(nil)
}

// testHashingDetector wraps an instance of and implements hash.Hash, invoking a
// callback if any hashing occurs.
type testHashingDetector struct {
	// Hash is the underlying hash function.
	hash.Hash
	// hashing is a callback invoked on hashing.
	hashing func()
}

// Sum implements hash.Hash.Sum.
func (p *testHashingDetector) Sum(b []byte) []byte {
	p.hashing()
	return p.Hash.Sum(b)
}
