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

func (v Version) hasher() hash.Hash {
	switch v {
	case Version_Version1:
		return sha1.New()
	default:
		panic("unsupported or unknown session version")
	}
}

func (s *Session) ensureValid() error {
	// Ensure that the session is non-nil.
	if s == nil {
		return errors.New("nil session")
	}

	// Ensure that the session identifier is valid.
	// TODO: Should we validate with a UUID regex here?
	if s.Identifier == "" {
		return errors.New("invalid session identifier")
	}

	// Ensure that the session version is supported.
	if !s.Version.supported() {
		return errors.New("unsupported or unknown session version")
	}

	// Ensure that the creation time is present.
	if s.CreationTime == nil {
		return errors.New("missing creation time")
	}

	// Ensure that the alpha URL is present and valid.
	if s.Alpha == nil {
		return errors.New("nil alpha URL")
	} else if err := s.Alpha.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid alpha URL")
	}

	// Ensure that the beta URL is present and valid.
	if s.Beta == nil {
		return errors.New("nil beta URL")
	} else if err := s.Beta.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid beta URL")
	}

	// Success.
	return nil
}
