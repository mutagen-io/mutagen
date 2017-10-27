package session

import (
	"os"
	"path/filepath"
	"time"

	"github.com/havoc-io/mutagen/filesystem"
)

const (
	maximumCacheAge       = 30 * 24 * time.Hour
	maximumStagingRootAge = maximumCacheAge
)

func housekeepCaches() {
	// Compute the path to the caches directory. If we fail, just abort. We
	// don't attempt to create the directory, because if it doesn't exist, then
	// we don't need to do anything and we'll just bail when we fail to list the
	// caches directory contents below.
	// TODO: Move this logic into paths.go? Need to keep it in sync with
	// pathForCache.
	cachesDirectoryPath, err := filesystem.Mutagen(false, cachesDirectoryName)
	if err != nil {
		return
	}

	// Get the list of caches. If we fail, just abort.
	cacheNames, err := filesystem.DirectoryContents(cachesDirectoryPath)
	if err != nil {
		return
	}

	// Grab the current time.
	now := time.Now()

	// Loop through each cache and remove those older than a certain age. Ignore
	// any failures.
	for _, n := range cacheNames {
		fullPath := filepath.Join(cachesDirectoryPath, n)
		if stat, err := os.Stat(fullPath); err != nil {
			continue
		} else if now.Sub(stat.ModTime()) > maximumCacheAge {
			os.Remove(fullPath)
		}
	}
}

func housekeepStaging() {
	// Compute the path to the staging directory (the top-level directory
	// containing all staging roots). If we fail, just abort. We don't attempt
	// to create the directory, because if it doesn't exist, then we don't need
	// to do anything and we'll just bail when we fail to list the staging
	// directory contents below.
	// TODO: Move this logic into paths.go? Need to keep it in sync with
	// pathForStagingRoot and pathForStaging.
	stagingDirectoryPath, err := filesystem.Mutagen(false, stagingDirectoryName)
	if err != nil {
		return
	}

	// Get the list of staging roots. If we fail, just abort.
	stagingRootNames, err := filesystem.DirectoryContents(stagingDirectoryPath)
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
	for _, n := range stagingRootNames {
		fullPath := filepath.Join(stagingDirectoryPath, n)
		if stat, err := os.Stat(fullPath); err != nil {
			continue
		} else if now.Sub(stat.ModTime()) > maximumStagingRootAge {
			os.RemoveAll(fullPath)
		}
	}
}
