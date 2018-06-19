package session

import (
	"crypto/sha1"
	"hash"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/sync"
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
		panic("unknown or unsupported session version")
	}
}

func (v Version) DefaultSymlinkMode() sync.SymlinkMode {
	switch v {
	case Version_Version1:
		return sync.SymlinkMode_SymlinkPortable
	default:
		panic("unknown or unsupported session version")
	}
}

func (v Version) DefaultWatchMode() filesystem.WatchMode {
	switch v {
	case Version_Version1:
		return filesystem.WatchMode_WatchPortable
	default:
		panic("unknown or unsupported session version")
	}
}

func (v Version) DefaultIgnoreVCSMode() sync.IgnoreVCSMode {
	switch v {
	case Version_Version1:
		return sync.IgnoreVCSMode_IgnoreVCS
	default:
		panic("unknown or unsupported session version")
	}
}

func (s *Session) EnsureValid() error {
	// A nil session is not valid.
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
		return errors.New("unknown or unsupported session version")
	}

	// Ensure that the creation time is present.
	if s.CreationTime == nil {
		return errors.New("missing creation time")
	}

	// Ensure that the alpha URL is valid.
	if err := s.Alpha.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid alpha URL")
	}

	// Ensure that the beta URL is valid.
	if err := s.Beta.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid beta URL")
	}

	// Ensure that the configuration is valid.
	if err := s.Configuration.EnsureValid(ConfigurationSourceSession); err != nil {
		return errors.Wrap(err, "invalid configuration")
	}

	// Success.
	return nil
}
