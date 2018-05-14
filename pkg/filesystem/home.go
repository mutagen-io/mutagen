package filesystem

import (
	"os/user"

	"github.com/pkg/errors"
)

// HomeDirectory is the cached path to the current user's home directory.
var HomeDirectory string

func init() {
	// Grab the current user's home directory. Check that it isn't empty,
	// because when compiling without cgo the $HOME environment variable is used
	// to compute the HomeDir field and we can't guarantee something isn't wonky
	// with the environment. We cache this because we don't expect it to change
	// and the underlying getuid system call is surprisingly expensive.
	if currentUser, err := user.Current(); err != nil {
		panic(errors.Wrap(err, "unable to lookup current user"))
	} else if currentUser.HomeDir == "" {
		panic(errors.Wrap(err, "unable to determine home directory"))
	} else {
		HomeDirectory = currentUser.HomeDir
	}
}
