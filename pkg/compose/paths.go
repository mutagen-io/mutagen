package compose

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// defaultConfigurationFileNames are the file names used when searching the
// current directory and parent directories for Docker Compose configuration
// files.
var defaultConfigurationFileNames = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
}

// defaultConfigurationOverrideFileNames are the names used when searching for
// override configuration files alongside the nominal configuration file.
var defaultConfigurationOverrideFileNames = []string{
	"docker-compose.override.yml",
	"docker-compose.override.yaml",
}

// findDefaultConfigurationFileInPathOrParent searches the specified path and
// its parent directories for a default Docker Compose configuration file,
// stopping after the first match. It returns the path at which the match was
// found and the matching file name. It will return os.ErrNotExist if no match
// is found, as well as any other error that occurs while traversing the
// filesystem. The specified path will be converted to an absolute path and
// cleaned, and thus any resulting path will also be absolute and cleaned. This
// function roughly models the logic of the find_candidates_in_parent_dirs
// function in Docker Compose. It's worth noting that
// find_candidates_in_parent_dirs will allow multiple matches (unlike
// get_default_override_file) and will just use the first match (while printing
// a warning). This function does the same, though it doesn't print a warning.
func findDefaultConfigurationFileInPathOrParent(path string) (string, string, error) {
	// Ensure that the path is absolute and cleaned so that filesystem root
	// detection works.
	path, err := filepath.Abs(path)
	if err != nil {
		return "", "", fmt.Errorf("unable to compute absolute path: %w", err)
	}

	// Loop until a match is found or the filesystem root is reached.
	for {
		// Check for matches at this level.
		for _, name := range defaultConfigurationFileNames {
			if _, err := os.Stat(filepath.Join(path, name)); err == nil {
				return path, name, nil
			}
		}

		// Compute the parent directory. If we're already at the filesystem
		// root, then there's nowhere left to search.
		if parent := filepath.Dir(path); parent == path {
			return "", "", os.ErrNotExist
		} else {
			path = parent
		}
	}
}

// findDefaultConfigurationOverrideFileInPath searches the target path for a
// default Docker Compose configuration override file and returns the matching
// file name. It will return an error if multiple override files exist and
// os.ErrNotExist if no match is found. This function roughly models the logic
// of the get_default_override_file function in Docker Compose.
func findDefaultConfigurationOverrideFileInPath(path string) (string, error) {
	// Perform the search and watch for multiple matches.
	var result string
	for _, name := range defaultConfigurationOverrideFileNames {
		if _, err := os.Stat(filepath.Join(path, name)); err == nil {
			if result != "" {
				return "", errors.New("multiple configuration override files found")
			}
			result = name
		}
	}

	// Handle the case of no match.
	if result == "" {
		return "", os.ErrNotExist
	}

	// Success.
	return result, nil
}
