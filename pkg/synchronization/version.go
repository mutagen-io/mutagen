package synchronization

import (
	"math"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/filesystem/behavior"
	"github.com/mutagen-io/mutagen/pkg/synchronization/compression"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore"
	"github.com/mutagen-io/mutagen/pkg/synchronization/hashing"
)

// DefaultVersion is the default session version.
const DefaultVersion Version = Version_Version1

// Supported indicates whether or not the session version is supported.
func (v Version) Supported() bool {
	switch v {
	case Version_Version1:
		return true
	default:
		return false
	}
}

// DefaultSynchronizationMode returns the default synchronization mode for the
// session version.
func (v Version) DefaultSynchronizationMode() core.SynchronizationMode {
	switch v {
	case Version_Version1:
		return core.SynchronizationMode_SynchronizationModeTwoWaySafe
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultHashingAlgorithm returns the default hashing algorithm for the session
// version.
func (v Version) DefaultHashingAlgorithm() hashing.Algorithm {
	switch v {
	case Version_Version1:
		return hashing.Algorithm_AlgorithmSHA1
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
		return ScanMode_ScanModeAccelerated
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

// DefaultSymbolicLinkMode returns the default symbolic link mode for the
// session version.
func (v Version) DefaultSymbolicLinkMode() core.SymbolicLinkMode {
	switch v {
	case Version_Version1:
		return core.SymbolicLinkMode_SymbolicLinkModePortable
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

// DefaultIgnoreSyntax returns the default ignore syntax for the session
// version.
func (v Version) DefaultIgnoreSyntax() ignore.Syntax {
	// NOTE: Due to the hack listed in Configuration.EnsureValid (regarding the
	// computation of the default ignore syntax), it would be advisable to keep
	// the default here the same for all session versions. If we want this
	// behavior to differ in the future, then we'd need to thread the session
	// version information into Configuration.EnsureValid, because the default
	// can affect the validation of ignore patterns. This hack could be replaced
	// by looser validation on ignore patterns, at least in the scenario where a
	// default ignore syntax is used (which is most cases, unfortunately), but
	// since we don't have any foreseeable reason to change this default across
	// future session versions, we're best off keeping the stricter validation
	// for now. We could also change the signature of Configuration.EnsureValid
	// to accept a session version, but that rapidly spirals into other APIs and
	// it's not even clear how to enforce that the daemon's default session
	// version is what's being used for validation in the command line interface
	// or external tools.
	switch v {
	case Version_Version1:
		return ignore.Syntax_SyntaxMutagen
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultIgnoreVCSMode returns the default VCS ignore mode for the session
// version.
func (v Version) DefaultIgnoreVCSMode() ignore.IgnoreVCSMode {
	switch v {
	case Version_Version1:
		return ignore.IgnoreVCSMode_IgnoreVCSModePropagate
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultPermissionsMode returns the default permissions mode for the session
// version.
func (v Version) DefaultPermissionsMode() core.PermissionsMode {
	// NOTE: Due to the hack listed in Configuration.EnsureValid (regarding the
	// computation of the default permissions mode), it would be advisable to
	// keep the default here the same for all session versions. If we want this
	// behavior to differ in the future, then we'd need to thread the session
	// version information into Configuration.EnsureValid, because the default
	// can affect the validation of default file and directory modes. This hack
	// could be replaced by looser validation on the default file and directory
	// modes, at least in the scenario where a default permissions mode is used
	// (which is most cases, unfortunately), but since we don't have any
	// foreseeable reason to change this default across future session versions,
	// we're best off keeping the stricter validation for now. We could also
	// change the signature of Configuration.EnsureValid to accept a session
	// version, but that rapidly spirals into other APIs and it's not even clear
	// how to enforce that the daemon's default session version is what's being
	// used for validation in the command line interface or external tools.
	switch v {
	case Version_Version1:
		return core.PermissionsMode_PermissionsModePortable
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

// DefaultCompressionAlgorithm returns the default compression algorithm for the
// session version.
func (v Version) DefaultCompressionAlgorithm() compression.Algorithm {
	switch v {
	case Version_Version1:
		return compression.Algorithm_AlgorithmDeflate
	default:
		panic("unknown or unsupported session version")
	}
}
