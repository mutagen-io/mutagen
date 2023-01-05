//go:build !(sspl && faststaging)

package local

import (
	"hash"

	"github.com/mutagen-io/mutagen/pkg/synchronization/endpoint/local/staging"
)

// newStager creates a new stager using the standard staging implementation.
func newStager(root string, hideRoot bool, hashFactory func() hash.Hash, maximumFileSize uint64) stager {
	return staging.NewStager(root, hideRoot, hashFactory(), maximumFileSize)
}
