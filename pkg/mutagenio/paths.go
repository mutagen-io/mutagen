package mutagenio

import (
	"fmt"
	"path/filepath"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

const (
	// loginFileName is the name of the file that stores the API token within
	// the mutagen.io data directory.
	loginFileName = "login"
)

// loginFilePath computes the path to the login file. If createDataDirectory is
// true, it will attempt to create the mutagen.io data directory if it doesn't
// already exist.
func loginFilePath(createDataDirectory bool) (string, error) {
	// Compute the path to the mutagen.io data directory.
	dataDirectoryPath, err := filesystem.Mutagen(createDataDirectory, filesystem.MutagenIODirectoryName)
	if err != nil {
		return "", fmt.Errorf("unable to compute/create mutagen.io data directory")
	}

	// Compute the path to the login file.
	return filepath.Join(dataDirectoryPath, loginFileName), nil
}
