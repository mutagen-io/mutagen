package agent

import (
	"path"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen"
	"github.com/havoc-io/mutagen/filesystem"
	"github.com/havoc-io/mutagen/process"
)

const (
	agentBundleName     = "mutagen-agents.tar.gz"
	agentsDirectoryName = "agents"
	agentBaseName       = "mutagen-agent"
)

var bundlePath string
var agentSSHCommand string

func init() {
	// Compute the path to the agent bundle.
	bundlePath = filepath.Join(
		process.Current.ExecutableParentPath,
		agentBundleName,
	)

	// Compute the agent SSH command.
	// HACK: This assumes that the SSH user's home directory is used as the
	// default working directory for SSH commands. We have to do this because we
	// don't have a portable mechanism to invoke the command relative to the
	// user's home directory (tilde doesn't work on Windows) and we don't want
	// to do a probe of the remote system before invoking the endpoint. This
	// assumption should be fine for 99.9% of cases, but if it becomes a major
	// issue, the only other options I see are probing before invoking (slow) or
	// using the Go SSH library to do this (painful to faithfully emulate
	// OpenSSH's behavior). Perhaps probing could be hidden behind an option?
	// HACK: We're assuming that none of these path components have spaces in
	// them, but since we control all of them, this is probably okay.
	// HACK: When invoking on Windows systems, we can use forward slashes for
	// the path and leave the "exe" suffix off the target name. This saves us a
	// target check.
	agentSSHCommand = path.Join(
		filesystem.MutagenDirectoryName,
		agentsDirectoryName,
		mutagen.Version(),
		agentBaseName,
	)
}

func installPath() (string, error) {
	// Compute (and create) the path to the agent parent directory.
	parent, err := filesystem.Mutagen(agentsDirectoryName, mutagen.Version())
	if err != nil {
		return "", errors.Wrap(err, "unable to compute parent directory")
	}

	// Compute the agent path.
	return filepath.Join(parent, process.ExecutableName(agentBaseName, runtime.GOOS)), nil
}
