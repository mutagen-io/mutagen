package session

import (
	"crypto/sha1"
	"hash"

	"github.com/pkg/errors"
)

func (v SessionVersion) supported() bool {
	switch v {
	case SessionVersion_Version1:
		return true
	default:
		return false
	}
}

func (v SessionVersion) hasher() (hash.Hash, error) {
	switch v {
	case SessionVersion_Version1:
		return sha1.New(), nil
	default:
		return nil, errors.New("unknown session version")
	}
}
