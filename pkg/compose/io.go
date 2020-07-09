package compose

import (
	"fmt"
	"io"
	"os"
)

// storeStandardInput reads the entirety of the standard input stream into a
// file at the specified path. The file is created with user-only permissions
// and must not already exist. The output file may be created even in the case
// of failure (for example if an error occurs during standard input copying). It
// is the responsibility of the caller to remove the file.
func storeStandardInput(target string) error {
	// Create the file and defer its closure.
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("unable to open output file: %w", err)
	}
	defer file.Close()

	// Copy contents.
	if _, err := io.Copy(file, os.Stdin); err != nil {
		return err
	}

	// Success.
	return nil
}
