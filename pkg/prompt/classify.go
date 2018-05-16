package prompt

import (
	"strings"
)

type PromptKind uint8

const (
	PromptKindSecret PromptKind = iota
	PromptKindEcho
	PromptKindBinary
)

var binaryPromptSuffixes = []string{
	"(yes/no)? ",
	"(yes/no): ",
}

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
