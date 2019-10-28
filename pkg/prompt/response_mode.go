package prompt

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
	// ResponseModeBinary indicates that a prompt response should be echoed and
	// additionally restricted to yes/no answers (potentially with an
	// alternative input control in the case of GUI input).
	ResponseModeBinary
)

// binaryPromptSuffixes are the list of binary prompt suffixes known to be used
// by OpenSSH.
var binaryPromptSuffixes = []string{
	"(yes/no)? ",
	"(yes/no): ",
}

// determineResponseMode attempts to determine the appropriate response mode for
// a prompt based on the prompt text.
func determineResponseMode(prompt string) ResponseMode {
	// Check if this is a yes/no prompt.
	for _, suffix := range binaryPromptSuffixes {
		if strings.HasSuffix(prompt, suffix) {
			return ResponseModeBinary
		}
	}

	// TODO: Are there any non-binary prompts from OpenSSH with responses that
	// should be echoed? If so, we need to create a white-listed registry of
	// regular expressions to match them.

	// Otherwise assume this is a secret prompt.
	return ResponseModeSecret
}
