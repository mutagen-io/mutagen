package sync

import (
	"os"

	"github.com/pkg/errors"

	fs "github.com/havoc-io/mutagen/pkg/filesystem"
)

// IsDefault indicates whether or not the permission exposure level is
// PermissionExposureLevel_PermissionExposureLevelDefault.
func (l PermissionExposureLevel) IsDefault() bool {
	return l == PermissionExposureLevel_PermissionExposureLevelDefault
}

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files.
func (l *PermissionExposureLevel) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a VCS mode.
	switch text {
	case "user":
		*l = PermissionExposureLevel_PermissionExposureLevelUser
	case "group":
		*l = PermissionExposureLevel_PermissionExposureLevelGroup
	case "other":
		*l = PermissionExposureLevel_PermissionExposureLevelOther
	default:
		return errors.Errorf("unknown permission exposure level specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular permission exposure level is
// a valid, non-default value.
func (l PermissionExposureLevel) Supported() bool {
	switch l {
	case PermissionExposureLevel_PermissionExposureLevelUser:
		return true
	case PermissionExposureLevel_PermissionExposureLevelGroup:
		return true
	case PermissionExposureLevel_PermissionExposureLevelOther:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of a permission exposure
// level.
func (l PermissionExposureLevel) Description() string {
	switch l {
	case PermissionExposureLevel_PermissionExposureLevelDefault:
		return "Default"
	case PermissionExposureLevel_PermissionExposureLevelUser:
		return "User"
	case PermissionExposureLevel_PermissionExposureLevelGroup:
		return "Group"
	case PermissionExposureLevel_PermissionExposureLevelOther:
		return "Other"
	default:
		return "Unknown"
	}
}

func (l PermissionExposureLevel) newDirectoryBaseMode() os.FileMode {
	switch l {
	case PermissionExposureLevel_PermissionExposureLevelUser:
		return 0700
	case PermissionExposureLevel_PermissionExposureLevelGroup:
		return 0770
	case PermissionExposureLevel_PermissionExposureLevelOther:
		return 0777
	default:
		panic("unsupported permission exposure level")
	}
}

func (l PermissionExposureLevel) newFileBaseMode() os.FileMode {
	switch l {
	case PermissionExposureLevel_PermissionExposureLevelUser:
		return 0600
	case PermissionExposureLevel_PermissionExposureLevelGroup:
		return 0660
	case PermissionExposureLevel_PermissionExposureLevelOther:
		return 0666
	default:
		panic("unsupported permission exposure level")
	}
}

// anyExecutableBitSet returns true if any executable bit is set on the file,
// false otherwise.
func anyExecutableBitSet(mode fs.Mode) bool {
	return mode&(fs.ModePermissionUserExecutable|fs.ModePermissionGroupExecutable|fs.ModePermissionOthersExecutable) != 0
}

// stripExecutableBits strips all executability bits from the specified file
// mode.
func stripExecutableBits(mode os.FileMode) os.FileMode {
	return mode &^ 0111
}

// markExecutableForReaders sets the executable bit for the mode for any case
// where the corresponding read bit is set.
func markExecutableForReaders(mode os.FileMode) os.FileMode {
	// Set the user executable bit if necessary.
	if mode&0400 != 0 {
		mode |= 0100
	}

	// Set the group executable bit if necessary.
	if mode&0040 != 0 {
		mode |= 0010
	}

	// Set the other executable bit if necessary.
	if mode&0004 != 0 {
		mode |= 0001
	}

	// Done.
	return mode
}
