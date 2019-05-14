package filesystem

import (
	"syscall"

	"github.com/pkg/errors"
)

// MarkHidden ensures that a path is hidden.
func MarkHidden(path string) error {
	// Convert the path to UTF-16 encoding for the system call.
	path16, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return errors.Wrap(err, "unable to convert path encoding")
	}

	// Get the existing file attributes.
	attributes, err := syscall.GetFileAttributes(path16)
	if err != nil {
		return errors.Wrap(err, "unable to get file attributes")
	}

	// Mark the hidden bit.
	attributes |= syscall.FILE_ATTRIBUTE_HIDDEN

	// Set the updated attributes.
	err = syscall.SetFileAttributes(path16, attributes)
	if err != nil {
		return errors.Wrap(err, "unable to set file attributes")
	}

	// Success.
	return nil
}
