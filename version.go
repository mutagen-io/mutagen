package mutagen

import (
	"fmt"
)

const (
	// VersionMajor represents the current major version of Mutagen.
	VersionMajor = 0
	// VersionMinor represents the current minor version of Mutagen.
	VersionMinor = 1
	// VersionPatch represents the current patch version of Mutagen.
	VersionPatch = 0
)

func Version() string {
	return fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
}
