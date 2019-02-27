package ssh

import (
	"fmt"
	"os"

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

// sshCommand returns the name or path specification to use for invoking ssh. It
// will use the MUTAGEN_SSH_PATH environment variable if provided, otherwise
// falling back to a platform-specific implementation.
func sshCommand() (string, error) {
	// If MUTAGEN_SSH_PATH is specified, then use it to perform the lookup.
	if searchPath := os.Getenv("MUTAGEN_SSH_PATH"); searchPath != "" {
		return process.FindCommand("ssh", []string{searchPath})
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
		return process.FindCommand("scp", []string{searchPath})
	}

	// Otherwise fall back to the platform-specific implementation.
	return scpCommandForPlatform()
}
