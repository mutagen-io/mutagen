// +build !darwin,!linux

package format

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

// QueryByPath queries the filesystem format for the specified path.
func QueryByPath(_ string) (Format, error) {
	return FormatUnknown, errors.New("format queries unsupported")
}

// Query queries the filesystem format for the specified directory.
func Query(_ *filesystem.Directory) (Format, error) {
	return FormatUnknown, errors.New("format queries unsupported")
}
