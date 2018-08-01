package filesystem

import (
	"os"
)

// GetOwnership returns the owning user and group IDs from file metadata. On
// Windows this function always returns 0s with a nil error.
func GetOwnership(_ os.FileInfo) (int, int, error) {
	return 0, 0, nil
}

// SetOwnership is a no-op on Windows.
func SetOwnership(_ string, _, _ int) error {
	return nil
}
