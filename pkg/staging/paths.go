package staging

import (
	"crypto/sha1"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
)

// pathForStaging computes the staging path for the specified path/digest. It
// returns the prefix directory name but does not ensure that it's been created.
func pathForStaging(root, path string, digest []byte) (string, string, error) {
	// Compute the prefix for the entry.
	if len(digest) == 0 {
		return "", "", errors.New("entry digest too short")
	}
	prefix := fmt.Sprintf("%x", digest[:1])

	// Compute the staging name.
	stagingName := fmt.Sprintf("%x_%x", sha1.Sum([]byte(path)), digest)

	// Success.
	return filepath.Join(root, prefix, stagingName), prefix, nil
}
