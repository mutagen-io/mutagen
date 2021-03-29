package filesystem

import (
	"time"
)

// Metadata encodes information about a filesystem entry.
type Metadata struct {
	// Name is the base name of the filesystem entry.
	Name string
	// Mode is the mode of the filesystem entry.
	Mode Mode
	// Size is the size of the filesystem entry in bytes.
	Size uint64
	// ModificationTime is the modification time of the filesystem entry.
	ModificationTime time.Time
	// DeviceID is the device ID of the filesystem on which the entry resides.
	// On POSIX systems, this is the value of the st_dev field of stat_t. On
	// Windows, this would most appropriately map to the volume serial number
	// (e.g. the dwVolumeSerialNumber field of BY_HANDLE_FILE_INFORMATION), but
	// this field is always left set to 0 because it can't be cheaply accessed
	// in all cases (e.g. when using FindFirstFile/FindNextFile) and because
	// it's not needed on Windows.
	DeviceID uint64
	// FileID is the file ID for the filesystem entry. On Windows systems it is
	// always 0.
	FileID uint64
}
