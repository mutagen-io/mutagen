// +build !darwin,!linux

package filesystem

import (
	"github.com/pkg/errors"
)

// QueryFormatByPath queries the filesystem format for the specified path.
func QueryFormatByPath(_ string) (Format, error) {
	return FormatUnknown, errors.New("format queries unsupported")
}

// QueryFormat queries the filesystem format for the specified directory.
func QueryFormat(_ *Directory) (Format, error) {
	return FormatUnknown, errors.New("format queries unsupported")
}
