package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/mutagen-io/extstat"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/process"
)

const (
	// maximumAgentIdlePeriod is the maximum period of time that an agent binary
	// is allowed to sit on disk without being executed before being deleted.
	maximumAgentIdlePeriod = 30 * 24 * time.Hour
)

// Housekeep performs housekeeping of agent binaries.
func Housekeep() {
	// Compute the path to the agents directory, but don't attempt to create it.
	// If we fail, then just abort.
	agentsDirectoryPath, err := filesystem.Mutagen(false, filesystem.MutagenAgentsDirectoryName)
	if err != nil {
		return
	}

	// Get the list of locally installed agent versions. If we fail, then just
	// abort. Failure here is most likely due to the directory not existing.
	agentDirectoryContents, err := filesystem.DirectoryContentsByPath(agentsDirectoryPath)
	if err != nil {
		return
	}

	// Compute the name of the agent binary.
	agentName := process.ExecutableName(BaseName, runtime.GOOS)

	// Grab the current time.
	now := time.Now()

	// Loop through each agent version, compute the time it was last launched,
	// and remove it if longer than the maximum allowed period. Skip contents
	// where failures are encountered.
	for _, c := range agentDirectoryContents {
		agentVersion := c.Name()
		agentVersionPath := filepath.Join(agentsDirectoryPath, agentVersion)
		agentExecutable := filepath.Join(agentVersionPath, agentName)
		if stat, err := extstat.NewFromFileName(agentExecutable); err != nil {
			continue
		} else if now.Sub(stat.AccessTime) > maximumAgentIdlePeriod {
			os.RemoveAll(agentVersionPath)
		}
	}
}
