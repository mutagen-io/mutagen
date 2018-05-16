package ssh

import (
	"encoding/base64"

	"github.com/havoc-io/mutagen/pkg/environment"
	"github.com/havoc-io/mutagen/pkg/process"
)

const (
	PrompterEnvironmentVariable              = "MUTAGEN_SSH_PROMPTER"
	PrompterMessageBase64EnvironmentVariable = "MUTAGEN_SSH_PROMPTER_MESSAGE_BASE64"

	sshAskpassEnvironmentVariable = "SSH_ASKPASS"
	displayEnvironmentVariable    = "DISPLAY"

	mutagenDisplay = "mutagen"
)

func prompterEnvironment(prompter, message string) []string {
	// Create a copy of the current environment.
	result := environment.CopyCurrent()

	// Handle based on whether or not there's a prompter.
	if prompter == "" {
		// If there is no prompter, then enforce that the relevant environment
		// variables are not set, because some systems (e.g. systems with a
		// Cygwin SSH binary) will include default SSH_ASKPASS values that throw
		// up GUIs without any message or context and we don't want that.
		delete(result, sshAskpassEnvironmentVariable)
		delete(result, displayEnvironmentVariable)
	} else {
		// Convert message to base64 encoding so that we can pass it through the
		// environment safely.
		messageBase64 := base64.StdEncoding.EncodeToString([]byte(message))

		// Tell SSH to use Mutagen to perform prompting.
		result[sshAskpassEnvironmentVariable] = process.Current.ExecutablePath
		result[displayEnvironmentVariable] = mutagenDisplay

		// Add environment variables to make Mutagen recognize an SSH prompting
		// invocation.
		result[PrompterEnvironmentVariable] = prompter
		result[PrompterMessageBase64EnvironmentVariable] = messageBase64
	}

	// Convert into the desired format.
	return environment.Format(result)
}
