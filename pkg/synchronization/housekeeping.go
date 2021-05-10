package synchronization

import (
	"os"
	"path/filepath"
	"time"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

const (
	// maximumCacheAge is the maximum allowed cache age.
	maximumCacheAge = 30 * 24 * time.Hour
	// maximumStagingRootAge is the maximum allowed staging root age.
	maximumStagingRootAge = 30 * 24 * time.Hour
)

// housekeepCaches performs housekeeping of caches.
func housekeepCaches() {
	// Compute the path to the caches directory, but don't attempt to create it.
	// If we fail, then just abort.
	cachesDirectoryPath, err := filesystem.Mutagen(false, filesystem.MutagenSynchronizationCachesDirectoryName)
	if err != nil {
		return
	}

	// Get the list of caches. If we fail, then just abort. Failure here is most
	// likely due to the directory not existing.
	cachesDirectoryContents, err := filesystem.DirectoryContentsByPath(cachesDirectoryPath)
	if err != nil {
		return
	}

	// Grab the current time.
	now := time.Now()

	// Loop through each cache and remove those older than a certain age. Ignore
	// any failures.
	for _, c := range cachesDirectoryContents {
		cacheName := c.Name()
		fullPath := filepath.Join(cachesDirectoryPath, cacheName)
		if stat, err := os.Stat(fullPath); err != nil {
			continue
		} else if now.Sub(stat.ModTime()) > maximumCacheAge {
			os.Remove(fullPath)
		}
	}
}

// housekeepStaging performs housekeeping of staging roots.
func housekeepStaging() {
	// Compute the path to the staging directory (the top-level directory
	// containing all staging roots), but don't attempt to create it. If we
	// fail, then just abort.
	stagingDirectoryPath, err := filesystem.Mutagen(false, filesystem.MutagenSynchronizationStagingDirectoryName)
	if err != nil {
		return
	}

	// Get the list of staging roots. If we fail, then just abort. Failure here
	// is most likely due to the directory not existing.
	stagingDirectoryContents, err := filesystem.DirectoryContentsByPath(stagingDirectoryPath)
	if err != nil {
		return
	}

	// Grab the current time.
	now := time.Now()

	// Loop through each staging root and remove those older than a certain
	// age. Ignore any failures. This is a little bit more cavalier than cache
	// housekeeping because removal is non-atomic and theoretically a given
	// staging root could be in use. However, a session's staging root is wiped
	// on each successful synchronization cycle, so by using a large maximum
	// staging root age, we're only going to run into trouble if the staging
	// portion of a synchronization cycle starts up, after having failed a long
	// time ago, at the precise moment that we're housekeeping. In that case, it
	// would try to use the existing staging directory from the failed
	// synchronization cycle and there might be a conflict. But even in that
	// statistically unlikely case, the worst case scenario would be triggering
	// an additional synchronization cycle.
	for _, c := range stagingDirectoryContents {
		stagingRootName := c.Name()
		fullPath := filepath.Join(stagingDirectoryPath, stagingRootName)
		if stat, err := os.Stat(fullPath); err != nil {
			continue
		} else if now.Sub(stat.ModTime()) > maximumStagingRootAge {
			os.RemoveAll(fullPath)
		}
	}
}

// Housekeep performs housekeeping of synchronization scan caches and staging
// directories.
func Housekeep() {
	housekeepCaches()
	housekeepStaging()
}
