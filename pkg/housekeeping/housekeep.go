package housekeeping

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/mutagen-io/extstat"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"

	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/platform"
	"github.com/mutagen-io/mutagen/pkg/sidecar"
)

const (
	// maximumAgentIdlePeriod is the maximum period of time that an agent binary
	// is allowed to sit on disk without being executed before being deleted.
	maximumAgentIdlePeriod = 30 * 24 * time.Hour
	// maximumCacheAge is the maximum allowed cache age.
	maximumCacheAge = 7 * 24 * time.Hour
	// maximumStagingRootAge is the maximum allowed staging root age.
	maximumStagingRootAge = 7 * 24 * time.Hour
)

// Housekeep invokes housekeeping functions on the Mutagen data directory.
func Housekeep(logger *logging.Logger) {
	// Perform housekeeping on agent binaries, but only if we're not in a
	// Mutagen sidecar container. Sidecar containers are particularly
	// susceptible to stale agent access times due to the fact that the agent is
	// baked into the sidecar image and the sidecar image is typically unpacked
	// via OverlayFS on top of ext4 with either relatime or noatime.
	if !sidecar.EnvironmentIsSidecar() {
		housekeepAgents(logger)
	}

	// Perform housekeeping on caches.
	housekeepCaches(logger)

	// Perform housekeeping on staging roots.
	housekeepStaging(logger)
}

// housekeepAgents performs housekeeping of agent binaries.
func housekeepAgents(logger *logging.Logger) {
	// Compute the path to the agents directory. If we fail, just abort. We
	// don't attempt to create the directory, because if it doesn't exist, then
	// we don't need to do anything and we'll just bail when we fail to list the
	// agent directory below.
	agentsDirectoryPath, err := filesystem.Mutagen(false, filesystem.MutagenAgentsDirectoryName)
	if err != nil {
		return
	}

	// Get the list of locally installed agent versions. If we fail, just abort.
	agentDirectoryContents, err := filesystem.DirectoryContentsByPath(agentsDirectoryPath)
	if err != nil {
		return
	}

	// Compute the name of the agent binary.
	agentName := platform.ExecutableName(agent.BaseName, runtime.GOOS)

	// Grab the current time.
	now := time.Now()

	// Loop through each agent version, compute the time it was last launched,
	// and remove it if longer than the maximum allowed period. Skip contents
	// where failures are encountered.
	for _, c := range agentDirectoryContents {
		// TODO: Ensure that the name matches the expected format. Be mindful of
		// the fact that it might contain a tag.
		agentVersion := c.Name()
		if stat, err := extstat.NewFromFileName(filepath.Join(agentsDirectoryPath, agentVersion, agentName)); err != nil {
			continue
		} else if now.Sub(stat.AccessTime) > maximumAgentIdlePeriod {
			fullPath := filepath.Join(agentsDirectoryPath, agentVersion)
			must.Succeed(os.RemoveAll(fullPath),
				fmt.Sprintf("remove all files from %s", fullPath),
				logger,
			)
		}
	}
}

// housekeepCaches performs housekeeping of caches.
func housekeepCaches(logger *logging.Logger) {
	// Compute the path to the caches directory. If we fail, just abort. We
	// don't attempt to create the directory, because if it doesn't exist, then
	// we don't need to do anything and we'll just bail when we fail to list the
	// caches directory contents below.
	// TODO: Move this logic into paths.go? Need to keep it in sync with
	// pathForCache.
	cachesDirectoryPath, err := filesystem.Mutagen(false, filesystem.MutagenSynchronizationCachesDirectoryName)
	if err != nil {
		return
	}

	// Get the list of caches. If we fail, just abort.
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
			must.OSRemove(fullPath, logger)
		}
	}
}

// housekeepStaging performs housekeeping of staging roots.
func housekeepStaging(logger *logging.Logger) {
	// Compute the path to the staging directory (the top-level directory
	// containing all staging roots). If we fail, just abort. We don't attempt
	// to create the directory, because if it doesn't exist, then we don't need
	// to do anything and we'll just bail when we fail to list the staging
	// directory contents below.
	// TODO: Move this logic into paths.go? Need to keep it in sync with
	// pathForStagingRoot and pathForStaging.
	stagingDirectoryPath, err := filesystem.Mutagen(false, filesystem.MutagenSynchronizationStagingDirectoryName)
	if err != nil {
		return
	}

	// Get the list of staging roots. If we fail, just abort.
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
			must.Succeed(os.RemoveAll(fullPath),
				fmt.Sprintf("remove all files from %s", fullPath),
				logger,
			)
		}
	}
}
