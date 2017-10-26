package ssh

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/howeyc/gopass"

	"github.com/havoc-io/mutagen/environment"
	"github.com/havoc-io/mutagen/process"
)

const (
	sshAskpassEnvironmentVariable = "SSH_ASKPASS"
	sshDisplayEnvironmentVariable = "DISPLAY"

	mutagenDisplay = "mutagen"

	PrompterEnvironmentVariable              = "MUTAGEN_PROMPTER"
	PrompterMessageBase64EnvironmentVariable = "MUTAGEN_PROMPTER_MESSAGE_BASE64"
)

type PromptClass uint8

const (
	PromptClassSecret PromptClass = iota
	PromptClassEcho
	PromptClassBinary
)

var binaryPromptSuffixes = []string{
	"(yes/no)? ",
	"(yes/no): ",
}

func ClassifyPrompt(prompt string) PromptClass {
	// Check if this is a yes/no prompt.
	for _, suffix := range binaryPromptSuffixes {
		if strings.HasSuffix(prompt, suffix) {
			return PromptClassBinary
		}
	}

	// TODO: Are there any non-binary prompts from OpenSSH with responses that
	// should be echoed? If so, we need to create a white-listed registry of
	// regular expressions to match them.

	// Otherwise assume this is a secret prompt.
	return PromptClassSecret
}

func PromptCommandLine(message, prompt string) (string, error) {
	// Classify the prompt.
	class := ClassifyPrompt(prompt)

	// Figure out which getter to use.
	var getter func() ([]byte, error)
	if class == PromptClassEcho || class == PromptClassBinary {
		getter = gopass.GetPasswdEchoed
	} else {
		getter = gopass.GetPasswd
	}

	// Print the message (if any) and the prompt.
	if message != "" {
		fmt.Println(message)
	}
	fmt.Print(prompt)

	// Get the result.
	result, err := getter()
	if err != nil {
		return "", errors.Wrap(err, "unable to read response")
	}

	// Success.
	return string(result), nil
}

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
		delete(result, sshDisplayEnvironmentVariable)
	} else {
		// Convert message to base64 encoding so that we can pass it through the
		// environment safely.
		messageBase64 := base64.StdEncoding.EncodeToString([]byte(message))

		// Insert necessary environment variables.
		result[sshAskpassEnvironmentVariable] = process.Current.ExecutablePath
		result[sshDisplayEnvironmentVariable] = mutagenDisplay
		result[PrompterEnvironmentVariable] = prompter
		result[PrompterMessageBase64EnvironmentVariable] = messageBase64
	}

	// Convert into the desired format.
	return environment.Format(result)
}
