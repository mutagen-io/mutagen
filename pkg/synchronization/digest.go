package synchronization

import (
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
)

// IsDefault indicates whether or not the digest is Digest_DigestDefault.
func (d Digest) IsDefault() bool {
	return d == Digest_DigestDefault
}

// MarshalText implements encoding.TextMarshaler.MarshalText.
func (d Digest) MarshalText() ([]byte, error) {
	var result string
	switch d {
	case Digest_DigestDefault:
	case Digest_DigestSHA1:
		result = "sha1"
	case Digest_DigestSHA256:
		result = "sha256"
	case Digest_DigestXXH128:
		result = "xxh128"
	default:
		result = "unknown"
	}
	return []byte(result), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText.
func (d *Digest) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a digest.
	switch text {
	case "sha1":
		*d = Digest_DigestSHA1
	case "sha256":
		*d = Digest_DigestSHA256
	case "xxh128":
		*d = Digest_DigestXXH128
	default:
		return fmt.Errorf("unknown digest algorithm specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular digest is a valid,
// non-default value.
func (d Digest) Supported() bool {
	switch d {
	case Digest_DigestSHA1:
		return true
	case Digest_DigestSHA256:
		return true
	case Digest_DigestXXH128:
		return digestXXH128Supported
	default:
		return false
	}
}

// Description returns a human-readable description of a digest.
func (d Digest) Description() string {
	switch d {
	case Digest_DigestDefault:
		return "Default"
	case Digest_DigestSHA1:
		return "SHA-1"
	case Digest_DigestSHA256:
		return "SHA-256"
	case Digest_DigestXXH128:
		return "XXH128"
	default:
		return "Unknown"
	}
}

// Factory returns a constructor for the digest algorithm. If invoked on a
// default or invalid Digest value, it will panic.
func (d Digest) Factory() func() hash.Hash {
	switch d {
	case Digest_DigestSHA1:
		return sha1.New
	case Digest_DigestSHA256:
		return sha256.New
	case Digest_DigestXXH128:
		return newXXH128Factory()
	default:
		panic("default or unknown digest algorithm")
	}
}
