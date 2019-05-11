package filesystem

import (
	"strconv"

	"github.com/pkg/errors"
)

const (
	// ModePermissionsMask is a bit mask that isolates portable permission bits.
	ModePermissionsMask = Mode(0777)

	// ModePermissionUserRead is the user readable bit.
	ModePermissionUserRead = Mode(0400)
	// ModePermissionUserWrite is the user writable bit.
	ModePermissionUserWrite = Mode(0200)
	// ModePermissionUserExecute is the user executable bit.
	ModePermissionUserExecute = Mode(0100)
	// ModePermissionGroupRead is the group readable bit.
	ModePermissionGroupRead = Mode(0040)
	// ModePermissionGroupWrite is the group writable bit.
	ModePermissionGroupWrite = Mode(0020)
	// ModePermissionGroupExecute is the group executable bit.
	ModePermissionGroupExecute = Mode(0010)
	// ModePermissionOthersRead is the others readable bit.
	ModePermissionOthersRead = Mode(0004)
	// ModePermissionOthersWrite is the others writable bit.
	ModePermissionOthersWrite = Mode(0002)
	// ModePermissionOthersExecute is the others executable bit.
	ModePermissionOthersExecute = Mode(0001)
)

// parseMode parses a user-specified octal string and verifies that it is
// limited to the bits specified in mask. It allows, but does not require, the
// string to begin with a 0 (or several 0s). The provided string must not be
// empty.
func parseMode(value string, mask Mode) (Mode, error) {
	if m, err := strconv.ParseUint(value, 8, 32); err != nil {
		return 0, errors.Wrap(err, "unable to parse numeric value")
	} else if mode := Mode(m); mode&mask != mode {
		return 0, errors.New("mode contains disallowed bits")
	} else {
		return mode, nil
	}
}

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files. It requires that the specified mode bits lie within
// ModePermissionsMask, otherwise an error is returned. If an error is returned,
// the mode is unmodified.
func (m *Mode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Perform parsing. We only allow the mode itself to be modified if parsing
	// is successful.
	if result, err := parseMode(text, ModePermissionsMask); err != nil {
		return err
	} else {
		*m = result
	}

	// Success.
	return nil
}
