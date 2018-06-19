package sync

import (
	"os"
)

const (
	// newDirectoryBaseMode is the base mode for directories created in
	// transitions.
	newDirectoryBaseMode os.FileMode = 0700

	// newFileBaseMode is the base mode for files created in transitions.
	newFileBaseMode os.FileMode = 0600

	// AnyExecutablePermission is the collection of executability bits that
	// indicate executability by the user. If any of these bits are set, the
	// file is considered executable. If a file's entry indicates that it is not
	// executable, all of these permissions should be removed by providers.
	AnyExecutablePermission os.FileMode = 0111

	// UserExecutablePermission is the permission that indicates only a user can
	// execute a file. If a file's entry indicates it is executable, this bit
	// should be set by providers.
	UserExecutablePermission os.FileMode = 0100
)
