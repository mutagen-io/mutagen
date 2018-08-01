package prompt

import (
	"strings"
)

// PromptKind represents the type of a prompt and how it should be displayed.
type PromptKind uint8

const (
	// PromptKindSecret indicates a prompt for which responses should not be
	// echoed.
	PromptKindSecret PromptKind = iota
	// PromptKindEcho indicates a prompt for which responses should be echoed.
	PromptKindEcho
	// PromptKindBinary indicates a prompt for which responses should be echoed,
	// and additionally should be restricted to yes/no answers (potentially with
	// an alternative input control in the case of GUI input).
	PromptKindBinary
)

// binaryPromptSuffixes are the list of binary prompt suffixes known to be used
// by OpenSSH.
var binaryPromptSuffixes = []string{
	"(yes/no)? ",
	"(yes/no): ",
}

// Classify classifies a prompt based on its text.
func Classify(prompt string) PromptKind {
	// Check if this is a yes/no prompt.
	for _, suffix := range binaryPromptSuffixes {
		if strings.HasSuffix(prompt, suffix) {
			return PromptKindBinary
		}
	}

	// TODO: Are there any non-binary prompts from OpenSSH with responses that
	// should be echoed? If so, we need to create a white-listed registry of
	// regular expressions to match them.

	// Otherwise assume this is a secret prompt.
	return PromptKindSecret
}
