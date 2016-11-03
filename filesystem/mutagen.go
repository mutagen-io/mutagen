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

func Mutagen(subpath ...string) (string, error) {
	// Grab the current user's home directory. Check that it isn't empty,
	// because when compiling without cgo the $HOME environment variable is used
	// to computer the HomeDir field and we can't guarantee something isn't
	// wonky with the environment.
	var home string
	if currentUser, err := user.Current(); err != nil {
		return "", errors.Wrap(err, "unable to lookup current user")
	} else if currentUser.HomeDir == "" {
		return "", errors.Wrap(err, "unable to determine home directory")
	} else {
		home = currentUser.HomeDir
	}

	// Collect path components and compute the result.
	components := make([]string, 0, 2+len(subpath))
	components = append(components, home, MutagenDirectoryName)
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
	if err := MarkHidden(root); err != nil {
		return "", errors.Wrap(err, "unable to hide Mutagen directory")
	}

	// Success.
	return result, nil
}
