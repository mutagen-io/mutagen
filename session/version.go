package session

import (
	"crypto/sha1"
	"hash"

	"github.com/pkg/errors"
)

func (v Version) supported() bool {
	switch v {
	case Version_Version1:
		return true
	default:
		return false
	}
}

func (v Version) hasher() (hash.Hash, error) {
	switch v {
	case Version_Version1:
		return sha1.New(), nil
	default:
		return nil, errors.New("unknown session version")
	}
}
