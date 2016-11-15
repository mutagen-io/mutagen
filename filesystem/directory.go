package filesystem

import (
	"os"
	"sort"

	"github.com/pkg/errors"
)

func DirectoryContents(path string) ([]string, error) {
	// Open the directory and ensure its closure.
	directory, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "unable to open directory")
	}
	defer directory.Close()

	// Grab the directory names.
	names, err := directory.Readdirnames(0)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read directory names")
	}

	// Normalize directory names if necessary.
	if err := normalizeDirectoryNames(path, names); err != nil {
		return nil, errors.Wrap(err, "unable to normalize directory names")
	}

	// Sort the names. This isn't really necessary for our use case, but it is
	// cheap and will make behavior nicer.
	sort.Strings(names)

	// Success.
	return names, nil
}
