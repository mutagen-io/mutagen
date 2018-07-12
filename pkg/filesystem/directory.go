package filesystem

import (
	"os"

	"github.com/pkg/errors"
)

func DirectoryContents(path string) ([]os.FileInfo, error) {
	// Open the directory and ensure its closure.
	directory, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "unable to open directory")
	}
	defer directory.Close()

	// Grab the directory contents.
	contents, err := directory.Readdir(0)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read directory contents")
	}

	// Success.
	return contents, nil
}
