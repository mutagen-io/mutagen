package prompting

import (
	"strings"
)

// ResponseMode encodes how a prompt response should be displayed and validated.
type ResponseMode uint8

const (
	// ResponseModeSecret indicates that a prompt response shouldn't be echoed.
	ResponseModeSecret ResponseMode = iota
	// ResponseModeMasked indicates that a prompt response should be masked.
	ResponseModeMasked
	// ResponseModeEcho indicates that a prompt response should be echoed.
	ResponseModeEcho
)

// echoedPromptSuffixes are the list of prompt suffixes known to be used by
// OpenSSH for which responses should be echoed.
var echoedPromptSuffixes = []string{
	"(yes/no)? ",
	"(yes/no): ",
	"(yes/no/[fingerprint])? ",
	"Please type 'yes', 'no' or the fingerprint: ",
}

// determineResponseMode attempts to determine the appropriate response mode for
// a prompt based on the prompt text.
func determineResponseMode(prompt string) ResponseMode {
	// Check if this is an echoed prompt.
	for _, suffix := range echoedPromptSuffixes {
		if strings.HasSuffix(prompt, suffix) {
			return ResponseModeEcho
		}
	}

	// TODO: Are there any non-binary prompts from OpenSSH with responses that
	// should be echoed? If so, we need to create a white-listed registry of
	// regular expressions to match them.

	// Otherwise assume this is a secret prompt.
	return ResponseModeSecret
}
