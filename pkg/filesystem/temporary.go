package filesystem

import (
	"strings"
)

const (
	// TemporaryNamePrefix is the file name prefix to use for intermediate
	// temporary files created by Mutagen. Using this prefix guarantees that any
	// such files will be ignored by filesystem watching and synchronization
	// scans. It may be suffixed with additional information if desired.
	TemporaryNamePrefix = ".mutagen-temporary-"
)

// IsTemporaryFileName determines whether or not a file name (not a file path)
// is the name of an intermediate temporary file used by Mutagen.
func IsTemporaryFileName(name string) bool {
	return strings.HasPrefix(name, TemporaryNamePrefix)
}
