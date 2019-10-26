package mutagenio

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"unicode/utf8"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

// Login stores the provided API token for use in API requests. It replaces any
// existing API token.
func Login(apiToken string) error {
	// Compute the path to the login file.
	loginFilePath, err := loginFilePath(true)
	if err != nil {
		return fmt.Errorf("unable to compute login file path: %w", err)
	}

	// TODO: Perform an authentication test using the token?

	// Write the API token to the path.
	if err := filesystem.WriteFileAtomic(loginFilePath, []byte(apiToken), 0600); err != nil {
		return fmt.Errorf("unable to write login file: %w", err)
	}

	// Success.
	return nil
}

// readAPIToken reads the stored API token, if any.
func readAPIToken() (string, error) {
	// Compute the path to the login file.
	loginFilePath, err := loginFilePath(false)
	if err != nil {
		return "", fmt.Errorf("unable to compute login file path: %w", err)
	}

	// Read the file contents.
	contents, err := ioutil.ReadFile(loginFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// TODO: We probably shouldn't make this determination here.
			return "", errors.New("user not logged in")
		}
		return "", fmt.Errorf("unable to read login file: %w", err)
	}

	// Ensure that the contents are valid.
	// TODO: Should we delete the file if one of these errors occurs? The user
	// can easily remedy the situation with a logout or new login.
	if len(contents) == 0 {
		return "", errors.New("empty login file found")
	} else if !utf8.Valid(contents) {
		return "", errors.New("invalid login file found")
	}

	// Success.
	return string(contents), nil
}

// Logout ensures that any stored API token is cleared. If no existing API token
// is present, Logout is a no-op.
func Logout() error {
	// Compute the path to the login file.
	loginFilePath, err := loginFilePath(false)
	if err != nil {
		return fmt.Errorf("unable to compute login file path: %w", err)
	}

	// Remove the file.
	if err := os.Remove(loginFilePath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("unable to remove login file: %w", err)
		}
	}

	// Success.
	return nil
}
