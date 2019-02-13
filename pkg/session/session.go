package session

import (
	"crypto/sha1"
	"hash"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/sync"
)

// Version indicates whether or not the session version is supported.
func (v Version) Supported() bool {
	switch v {
	case Version_Version1:
		return true
	default:
		return false
	}
}

// hasher creates an appropriate hash function for the session version.
func (v Version) Hasher() hash.Hash {
	switch v {
	case Version_Version1:
		return sha1.New()
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultSynchronizationMode returns the default synchronization mode for the
// session version.
func (v Version) DefaultSynchronizationMode() sync.SynchronizationMode {
	switch v {
	case Version_Version1:
		return sync.SynchronizationMode_SynchronizationModeTwoWaySafe
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultSymlinkMode returns the default symlink mode for the session version.
func (v Version) DefaultSymlinkMode() sync.SymlinkMode {
	switch v {
	case Version_Version1:
		return sync.SymlinkMode_SymlinkPortable
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultWatchMode returns the default watch mode for the session version.
func (v Version) DefaultWatchMode() filesystem.WatchMode {
	switch v {
	case Version_Version1:
		return filesystem.WatchMode_WatchModePortable
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultWatchPollingInterval returns the default watch polling interval for
// the session version.
func (v Version) DefaultWatchPollingInterval() uint32 {
	switch v {
	case Version_Version1:
		return 10
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultIgnoreVCSMode returns the default VCS ignore mode for the session
// version.
func (v Version) DefaultIgnoreVCSMode() sync.IgnoreVCSMode {
	switch v {
	case Version_Version1:
		return sync.IgnoreVCSMode_PropagateVCS
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultFileMode returns the default file permission mode for the session
// version.
func (v Version) DefaultFileMode() filesystem.Mode {
	switch v {
	case Version_Version1:
		return filesystem.ModePermissionUserRead |
			filesystem.ModePermissionUserWrite
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultDirectoryMode returns the default directory permission mode for the
// session version.
func (v Version) DefaultDirectoryMode() filesystem.Mode {
	switch v {
	case Version_Version1:
		return filesystem.ModePermissionUserRead |
			filesystem.ModePermissionUserWrite |
			filesystem.ModePermissionUserExecute
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultOwnerSpecification returns the default owner specification for the
// session version.
func (v Version) DefaultOwnerSpecification() string {
	switch v {
	case Version_Version1:
		return ""
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultGroupSpecification returns the default owner group specification for
// the session version.
func (v Version) DefaultGroupSpecification() string {
	switch v {
	case Version_Version1:
		return ""
	default:
		panic("unknown or unsupported session version")
	}
}

// EnsureValid ensures that Session's invariants are respected.
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
	if !s.Version.Supported() {
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
	if err := s.Configuration.EnsureValid(ConfigurationSourceTypeSession); err != nil {
		return errors.Wrap(err, "invalid configuration")
	}

	// Ensure that the alpha-specific configuration is valid.
	if err := s.ConfigurationAlpha.EnsureValid(ConfigurationSourceTypeSessionEndpointSpecific); err != nil {
		return errors.Wrap(err, "invalid alpha-specific configuration")
	}

	// Ensure that the beta-specific configuration is valid.
	if err := s.ConfigurationBeta.EnsureValid(ConfigurationSourceTypeSessionEndpointSpecific); err != nil {
		return errors.Wrap(err, "invalid beta-specific configuration")
	}

	// Success.
	return nil
}
