package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/shibukawa/extstat"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/process"
)

const (
	// maximumAgentIdlePeriod is the maximum period of time that an agent binary
	// is allowed to sit on disk without being executed before being deleted.
	maximumAgentIdlePeriod = 30 * 24 * time.Hour
)

// Housekeep performs housekeeping of agent binaries.
func Housekeep() {
	// Compute the path to the agents directory. If we fail, just abort. We
	// don't attempt to create the directory, because if it doesn't exist, then
	// we don't need to do anything and we'll just bail when we fail to list the
	// agent directory below.
	agentsDirectoryPath, err := filesystem.Mutagen(false, agentsDirectoryName)
	if err != nil {
		return
	}

	// Get the list of locally installed agent versions. If we fail, just abort.
	agentDirectoryContents, err := filesystem.DirectoryContentsByPath(agentsDirectoryPath)
	if err != nil {
		return
	}

	// Compute the name of the agent binary.
	agentName := process.ExecutableName(agentBaseName, runtime.GOOS)

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
			os.RemoveAll(filepath.Join(agentsDirectoryPath, agentVersion))
		}
	}
}
