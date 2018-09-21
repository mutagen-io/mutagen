package filesystem

import (
	"os/user"

	"github.com/pkg/errors"
)

// mustComputeHomeDirectory computes the user's home directory and panics on
// failure. The os/user package never uses its pure-Go implementation on
// Windows, so we can safely use it (unlike on POSIX).
func mustComputeHomeDirectory() string {
	// Look up the current user.
	currentUser, err := user.Current()
	if err != nil {
		panic(errors.Wrap(err, "unable to lookup current user"))
	}

	// Verify that their home directory is non-empty.
	if currentUser.HomeDir == "" {
		panic(errors.New("empty home directory found"))
	}

	// Success.
	return currentUser.HomeDir
}
