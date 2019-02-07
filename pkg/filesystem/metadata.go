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
	// DeviceID is the filesystem device ID on which the filesytem entry
	// resides. On Windows systems it is always 0.
	DeviceID uint64
	// FileID is the file ID for the filesystem entry. On Windows systems it is
	// always 0.
	FileID uint64
}
