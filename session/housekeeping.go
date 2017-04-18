package session

import (
	"os"
	"path/filepath"
	"time"

	"github.com/havoc-io/mutagen/filesystem"
)

const (
	maximumCacheAge = 30 * 24 * time.Hour
)

func housekeep() {
	// Compute the path to the caches directory. If we fail, just abort. We
	// don't attempt to create the directory, because if it doesn't exist, then
	// we don't need to do anything and we'll just bail when we fail to list the
	// caches directory below.
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

	// Loop through each cache version and remove those older than a certain
	// age. Ignore any failures.
	for _, n := range cacheNames {
		fullPath := filepath.Join(cachesDirectoryPath, n)
		if stat, err := os.Stat(fullPath); err != nil {
			continue
		} else if now.Sub(stat.ModTime()) > maximumCacheAge {
			os.Remove(fullPath)
		}
	}
}
