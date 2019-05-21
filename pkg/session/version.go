package session

import (
	"crypto/sha1"
	"hash"
	"math"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/filesystem/behavior"
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

// DefaultMaximumEntryCount returns the default maximum entry count for the
// session version.
func (v Version) DefaultMaximumEntryCount() uint64 {
	switch v {
	case Version_Version1:
		return math.MaxUint64
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultMaximumStagingFileSize returns the default maximum staging file size
// for the session version.
func (v Version) DefaultMaximumStagingFileSize() uint64 {
	switch v {
	case Version_Version1:
		return math.MaxUint64
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultProbeMode returns the default probe mode for the session version.
func (v Version) DefaultProbeMode() behavior.ProbeMode {
	switch v {
	case Version_Version1:
		return behavior.ProbeMode_ProbeModeProbe
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultScanMode returns the default scan mode for the session version.
func (v Version) DefaultScanMode() ScanMode {
	switch v {
	case Version_Version1:
		return ScanMode_ScanModeFull
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultStageMode returns the default staging mode for the session version.
func (v Version) DefaultStageMode() StageMode {
	switch v {
	case Version_Version1:
		return StageMode_StageModeMutagen
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultSymlinkMode returns the default symlink mode for the session version.
func (v Version) DefaultSymlinkMode() sync.SymlinkMode {
	switch v {
	case Version_Version1:
		return sync.SymlinkMode_SymlinkModePortable
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultWatchMode returns the default watch mode for the session version.
func (v Version) DefaultWatchMode() WatchMode {
	switch v {
	case Version_Version1:
		return WatchMode_WatchModePortable
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
		return sync.IgnoreVCSMode_IgnoreVCSModePropagate
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
