package session

import (
	"crypto/sha1"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/sync"
)

func checksum(entry *sync.Entry) ([]byte, error) {
	// Create the hasher.
	hasher := sha1.New()

	// Only process the entry if non-nil.
	if entry != nil {
		// Serialize the entry.
		serialized, err := entry.Marshal()
		if err != nil {
			return nil, errors.Wrap(err, "unable to serialize entry")
		}

		// Add it to the digest. Note that hash.Hash's Write method can't return
		// an error.
		hasher.Write(serialized)
	}

	// Compute the sum.
	return hasher.Sum(nil), nil
}
