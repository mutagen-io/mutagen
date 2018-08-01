package prompt

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/howeyc/gopass"
)

// PromptCommandLine performs prompting on the command line.
func PromptCommandLine(prompt string) (string, error) {
	// Classify the prompt.
	class := Classify(prompt)

	// Figure out which getter to use.
	var getter func() ([]byte, error)
	if class == PromptKindEcho || class == PromptKindBinary {
		getter = gopass.GetPasswdEchoed
	} else {
		getter = gopass.GetPasswd
	}

	// Print the prompt.
	fmt.Print(prompt)

	// Get the result.
	result, err := getter()
	if err != nil {
		return "", errors.Wrap(err, "unable to read response")
	}

	// Success.
	return string(result), nil
}
