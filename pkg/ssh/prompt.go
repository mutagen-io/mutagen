package ssh

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
)

const (
	PrompterEnvironmentVariable              = "MUTAGEN_SSH_PROMPTER"
	PrompterMessageBase64EnvironmentVariable = "MUTAGEN_SSH_PROMPTER_MESSAGE_BASE64"
)

func addPrompterVariables(environment []string, prompter, message string) ([]string, error) {
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
		// Convert message to base64 encoding so that we can pass it through the
		// environment safely.
		messageBase64 := base64.StdEncoding.EncodeToString([]byte(message))

		// Compute the path to the current (mutagen) executable and set it in
		// the SSH_ASKPASS variable.
		if mutagenPath, err := os.Executable(); err != nil {
			return nil, errors.Wrap(err, "unable to determine executable path")
		} else {
			environment = append(environment, fmt.Sprintf("SSH_ASKPASS=%s", mutagenPath))
		}

		// Set the DISPLAY variable to Mutagen.
		environment = append(environment, "DISPLAY=mutagen")

		// Add environment variables to make Mutagen recognize an SSH prompting
		// invocation.
		environment = append(environment, fmt.Sprintf("%s=%s", PrompterEnvironmentVariable, prompter))
		environment = append(environment, fmt.Sprintf("%s=%s", PrompterMessageBase64EnvironmentVariable, messageBase64))
	}

	// Done.
	return environment, nil
}
