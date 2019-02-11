package ssh

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/prompt"
)

// setPrompterVariables sets up environment variables for prompting based on the
// provided prompter identifier. If an empty identifier is provided, then any
// potentially conflicting environment variables (that might cause alternative
// prompting) are removed.
func setPrompterVariables(environment []string, prompter string) ([]string, error) {
	// Handle based on whether or not there's a prompter.
	if prompter == "" {
		// If there is no prompter, then enforce that SSH_ASKPASS is not set,
		// because some systems (e.g. systems with a Cygwin SSH binary) will
		// include default SSH_ASKPASS values that throw up GUIs without any
		// message or context and we don't want that.
		filteredEnvironment := environment[:0]
		for _, e := range environment {
			if !strings.HasPrefix(e, "SSH_ASKPASS=") {
				filteredEnvironment = append(filteredEnvironment, e)
			}
		}
		environment = filteredEnvironment
	} else {
		// Compute the path to the current (mutagen) executable and set it in
		// the SSH_ASKPASS variable.
		if mutagenPath, err := os.Executable(); err != nil {
			return nil, errors.Wrap(err, "unable to determine executable path")
		} else {
			environment = append(environment, fmt.Sprintf("SSH_ASKPASS=%s", mutagenPath))
		}

		// Set the DISPLAY variable to Mutagen.
		environment = append(environment, "DISPLAY=mutagen")

		// Add the prompter environment variable to make Mutagen recognize a
		// prompting invocation.
		environment = append(environment,
			fmt.Sprintf("%s=%s", prompt.PrompterEnvironmentVariable, prompter),
		)
	}

	// Done.
	return environment, nil
}
