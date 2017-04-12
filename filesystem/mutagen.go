package filesystem

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	MutagenDirectoryName = ".mutagen"
)

// userHomeDirectory is the cached path to the current user's home directory.
var userHomeDirectory string

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
		userHomeDirectory = currentUser.HomeDir
	}
}

func Mutagen(subpath ...string) (string, error) {
	// Collect path components and compute the result.
	components := make([]string, 0, 2+len(subpath))
	components = append(components, userHomeDirectory, MutagenDirectoryName)
	root := filepath.Join(components...)
	components = append(components, subpath...)
	result := filepath.Join(components...)

	// TODO: Should we iterate through each component and ensure the user hasn't
	// changed the directory permissions? MkdirAll won't reset them. But I
	// suppose the user may have changed them for whatever reason (though I
	// can't imagine any).

	// Perform creation.
	if err := os.MkdirAll(result, 0700); err != nil {
		return "", errors.Wrap(err, "unable to create subpath")
	}

	// Mark the Mutagen root directory as hidden.
	// TODO: Should we only do this when we create the root? If users are
	// intentionally having this shown, then we might not want to override that.
	if err := markHidden(root); err != nil {
		return "", errors.Wrap(err, "unable to hide Mutagen directory")
	}

	// Success.
	return result, nil
}
