//go:build !windows

package filesystem

import (
	"os"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"github.com/mutagen-io/mutagen/pkg/parallelism"
)

// readContentMetadata reads filesystem metadata using an fstatat operation with
// the specified directory file descriptor and content name. It does not follow
// symbolic links.
func readContentMetadata(descriptor int, name string) (*Metadata, error) {
	// Query metadata.
	var metadata unix.Stat_t
	if err := fstatatRetryingOnEINTR(descriptor, name, &metadata, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return nil, err
	}

	// Success.
	return &Metadata{
		Name:             name,
		Mode:             Mode(metadata.Mode),
		Size:             uint64(metadata.Size),
		ModificationTime: time.Unix(metadata.Mtim.Unix()),
		DeviceID:         uint64(metadata.Dev),
		FileID:           uint64(metadata.Ino),
	}, nil
}

// readContentMetadataWorkers is an array of worker Goroutines used to perform
// parallel filesystem metadata reads.
var readContentMetadataWorkers *parallelism.SIMDWorkerArray

// readContentMetadataWorkersInitializeOnce guards initialization of
// readContentMetadataWorkers.
var readContentMetadataWorkersInitializeOnce sync.Once

// fstatatSIMDWork implements parallelism.SIMDWork for fstatat operations.
type fstatatSIMDWork struct {
	// descriptor is the directory file descriptor.
	descriptor int
	// names are the names to query within the directory.
	names []string
	// results are the query results. They will be set nil if the corresponding
	// entry didn't exist on the filesystem.
	results []*Metadata
}

// Do implements parallelism.SIMDWork.Do.
func (w *fstatatSIMDWork) Do(index, size int) error {
	// Loop over the names in a strided fashion and populate results.
	for i := index; i < len(w.names); i += size {
		metadata, err := readContentMetadata(w.descriptor, w.names[i])
		if err != nil {
			if os.IsNotExist(err) {
				w.results[i] = nil
				continue
			}
			return err
		}
		w.results[i] = metadata
	}

	// Success.
	return nil
}

// readContentMetadataParallel is a parallelized version of readContentMetadata
// that uses a global pool of worker Goroutines to perform fstatat operations in
// parallel. If any of the names in the specified list don't exist within the
// directory at the time of scanning, then they'll simply be ignored.
func readContentMetadataParallel(descriptor int, names []string) ([]*Metadata, error) {
	// Handle cases that don't benefit from parallelization.
	count := len(names)
	if count == 0 {
		return nil, nil
	} else if count == 1 {
		result, err := readContentMetadata(descriptor, names[0])
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
		return []*Metadata{result}, nil
	}

	// Ensure that the global worker pool is initialized.
	readContentMetadataWorkersInitializeOnce.Do(func() {
		readContentMetadataWorkers = parallelism.NewSIMDWorkerArray(0)
	})

	// Set up the results slice.
	results := make([]*Metadata, count)

	// Perform the workload.
	err := readContentMetadataWorkers.Do(&fstatatSIMDWork{
		descriptor: descriptor,
		names:      names,
		results:    results,
	})
	if err != nil {
		return nil, err
	}

	// Filter out cases of non-existence.
	filtered := results[:0]
	for _, m := range results {
		if m != nil {
			filtered = append(filtered, m)
		}
	}

	// Success.
	return filtered, nil
}
