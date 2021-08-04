package ssh

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/prompting"
)

// SetPrompterVariables sets up environment variables for prompting based on the
// provided prompter identifier. If an empty identifier is provided, then any
// potentially conflicting environment variables (that might cause alternative
// prompting) are removed.
func SetPrompterVariables(environment []string, prompter string) ([]string, error) {
	// Handle based on whether or not there's a prompter.
	if prompter == "" {
		// If there is no prompter, then enforce that SSH_ASKPASS is not set,
		// because some systems (e.g. systems with a Cygwin SSH binary) set
		// default SSH_ASKPASS values that throw up GUIs without any message or
		// context, which we don't want.
		filteredEnvironment := environment[:0]
		for _, e := range environment {
			if !strings.HasPrefix(e, "SSH_ASKPASS=") {
				filteredEnvironment = append(filteredEnvironment, e)
			}
		}
		environment = filteredEnvironment
	} else {
		// Compute the path to the current (mutagen) executable and store it in
		// the SSH_ASKPASS variable.
		if mutagenPath, err := os.Executable(); err != nil {
			return nil, errors.Wrap(err, "unable to determine executable path")
		} else {
			environment = append(environment, fmt.Sprintf("SSH_ASKPASS=%s", mutagenPath))
		}

		// Ensure that the SSH_ASKPASS mechanism is going to be used, even if
		// the process has a controlling terminal (e.g. when running the daemon
		// manually in the foreground). Prior to OpenSSH 8.4, all that was
		// needed was to set the DISPLAY variable to a non-empty value. In
		// OpenSSH 8.4 and later, the controlling terminal (if one is set) will
		// be used for prompting even if SSH_ASKPASS and DISPLAY are set, but an
		// additional environment variable, SSH_ASKPASS_REQUIRE, has been added
		// to provide more granular control of SSH_ASKPASS behavior, so we use
		// that to force usage of the SSH_ASKPASS mechanism in this case.
		environment = append(environment,
			"DISPLAY=mutagen",
			"SSH_ASKPASS_REQUIRE=force",
		)

		// Add the prompter environment variable to make Mutagen recognize a
		// prompting invocation.
		environment = append(environment,
			fmt.Sprintf("%s=%s", prompting.PrompterEnvironmentVariable, prompter),
		)
	}

	// Done.
	return environment, nil
}
