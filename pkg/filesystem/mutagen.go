package filesystem

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	// mutagenConfigurationName is the name of the Mutagen configuration file
	// inside the user's home directory.
	mutagenConfigurationName = ".mutagen.toml"

	// MutagenDirectoryName is the name of the Mutagen control directory inside
	// the user's home directory.
	MutagenDirectoryName = ".mutagen"

	// mutagenDirectoryPermissions are the permissions for the Mutagen control
	// directory and its subdirectories.
	mutagenDirectoryPermissions os.FileMode = 0700
)

var MutagenConfigurationPath string

func init() {
	MutagenConfigurationPath = filepath.Join(HomeDirectory, mutagenConfigurationName)
}

func Mutagen(create bool, subpath ...string) (string, error) {
	// Collect path components and compute the result.
	components := make([]string, 0, 2+len(subpath))
	components = append(components, HomeDirectory, MutagenDirectoryName)
	root := filepath.Join(components...)
	components = append(components, subpath...)
	result := filepath.Join(components...)

	// If requested, attempt to create the Mutagen directory and the specified
	// subpath. Also ensure that the Mutagen directory is hidden.
	// TODO: Should we iterate through each component and ensure the user hasn't
	// changed the directory permissions? MkdirAll won't reset them. But I
	// suppose the user may have changed them for whatever reason (though I
	// can't imagine any).
	if create {
		if err := os.MkdirAll(result, mutagenDirectoryPermissions); err != nil {
			return "", errors.Wrap(err, "unable to create subpath")
		} else if err := markHidden(root); err != nil {
			return "", errors.Wrap(err, "unable to hide Mutagen directory")
		}
	}

	// Success.
	return result, nil
}
