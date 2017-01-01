package session

import (
	"crypto/sha1"
	"hash"
)

func (v Version) supported() bool {
	switch v {
	case Version_Version1:
		return true
	default:
		return false
	}
}

func (v Version) hasher() hash.Hash {
	switch v {
	case Version_Version1:
		return sha1.New()
	default:
		panic("unknown session version")
	}
}
