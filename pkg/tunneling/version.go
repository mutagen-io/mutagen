package tunneling

import (
	"crypto/sha256"
	"hash"
)

// Supported indicates whether or not the tunnel version is supported.
func (v Version) Supported() bool {
	switch v {
	case Version_Version1:
		return true
	default:
		return false
	}
}

// secretLength returns the secret length for the tunnel version.
func (v Version) secretLength() int {
	switch v {
	case Version_Version1:
		return 32
	default:
		panic("unknown or unsupported tunnel version")
	}
}

// hmacHash returns the HMAC hash constructor for the tunnel version.
func (v Version) hmacHash() func() hash.Hash {
	switch v {
	case Version_Version1:
		return sha256.New
	default:
		panic("unknown or unsupported tunnel version")
	}
}
