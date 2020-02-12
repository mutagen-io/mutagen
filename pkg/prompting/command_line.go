package prompting

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/mutagen-io/gopass"
)

// PromptCommandLineWithResponseMode performs command line prompting using the
// specified response mode.
func PromptCommandLineWithResponseMode(prompt string, mode ResponseMode) (string, error) {
	// Figure out which getter to use.
	var getter func() ([]byte, error)
	if mode == ResponseModeEcho {
		getter = gopass.GetPasswdEchoed
	} else if mode == ResponseModeMasked {
		getter = gopass.GetPasswdMasked
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

// PromptCommandLine performs command line prompting using an automatically
// determined response mode.
func PromptCommandLine(prompt string) (string, error) {
	return PromptCommandLineWithResponseMode(prompt, determineResponseMode(prompt))
}
