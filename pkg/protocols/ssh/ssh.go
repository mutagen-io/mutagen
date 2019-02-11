package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/process"
)

const (
	// connectTimeoutSeconds is the default timeout value (in seconds) to use
	// with SSH-based commands.
	connectTimeoutSeconds = 5
)

// compressionArgument returns a flag that can be passed to scp or ssh to enable
// compression. Note that while SSH does have a CompressionLevel configuration
// option, this only applies to SSHv1. SSHv2 defaults to a DEFLATE level of 6,
// which is what we want anyway.
func compressionArgument() string {
	return "-C"
}

// timeoutArgument returns a option flag that can be passed to scp or ssh to
// limit connection time (though not transfer time or process lifetime). It is
// currently a fixed value, but in the future we might want to make this
// configurable for people with poor connections.
func timeoutArgument() string {
	return fmt.Sprintf("-oConnectTimeout=%d", connectTimeoutSeconds)
}

// findCommand searches for a command with the specified name within the
// specified list of directories. It's similar to os/exec.LookPath, except that
// it allows one to manually specify paths, and it uses a slightly simpler
// lookup mechanism.
func findCommand(name string, paths []string) (string, error) {
	// Iterate through the directories.
	for _, path := range paths {
		// Compute the target name.
		target := filepath.Join(path, process.ExecutableName(name, runtime.GOOS))

		// Check if the target exists and has the correct type.
		// TODO: Should we do more extensive (and platform-specific) testing on
		// the resulting metadata? See, e.g., the implementation of
		// os/exec.LookPath.
		if metadata, err := os.Stat(target); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", errors.Wrap(err, "unable to query file metadata")
		} else if metadata.Mode()&os.ModeType != 0 {
			continue
		} else {
			return target, nil
		}
	}

	// Failure.
	return "", errors.New("unable to locate command")
}

// sshCommand returns the name or path specification to use for invoking ssh. It
// will use the MUTAGEN_SSH_PATH environment variable if provided, otherwise
// falling back to a platform-specific implementation.
func sshCommand() (string, error) {
	// If MUTAGEN_SSH_PATH is specified, then use it to perform the lookup.
	if searchPath := os.Getenv("MUTAGEN_SSH_PATH"); searchPath != "" {
		return findCommand("ssh", []string{searchPath})
	}

	// Otherwise fall back to the platform-specific implementation.
	return sshCommandForPlatform()
}

// scpCommand returns the name or path specification to use for invoking scp. It
// will use the MUTAGEN_SSH_PATH environment variable if provided, otherwise
// falling back to a platform-specific implementation.
func scpCommand() (string, error) {
	// If MUTAGEN_SSH_PATH is specified, then use it to perform the lookup.
	if searchPath := os.Getenv("MUTAGEN_SSH_PATH"); searchPath != "" {
		return findCommand("scp", []string{searchPath})
	}

	// Otherwise fall back to the platform-specific implementation.
	return scpCommandForPlatform()
}
